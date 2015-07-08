BuffStreams
====================

Streaming Protocol Buffers messages over TCP in Golang

What is BuffStreams?
=====================

BuffStreams is a manager for streaming TCP connections to a server that receives protocol buffers (or in the case of a non-protobuffs server, one that uses the same wire scheme of a fixed header + N bytes per message).

BuffStreams gives you a simple interface to start a nonblocking listener on a given port, which will stream arrays of raw bytes into a callback you provide it. In this way, BuffStreams is not so much a daemon, but a library to build networked services that can  communicate over TCP using Protocol Buffer messages.

Why BuffStreams
====================

I was writing a few different projects for fun in Golang, and kept writing code something like what is in the library, but less organized. I decided to focus on the networking code, pulling it out and improving it so I knew it could be trusted to perform reliably across projects. 

How does it work?
=================

Since protobuff messages lack any kind of natural delimeter, BuffStreams uses the method of adding a fixed header of bytes (which is configurable) that describes the size of the actual payload. This is handled for you, by the call to write. You never need to pack on the size yourself.

On the server side, it will listen for these payloads, read the fixed header, and then the subsequent message. The server must have the same maximum size as the client for this to work. BuffStreams will then pass the byte array to a callback you provided for handling messages received on that port. Deserializing the messages and interpreting their value is up to you.

Naming Strategy
=======================

I will apologize in advance for the pretty terrible names I chose for this library. It's way better than the original set of names I had for it. But that isn't saying much.

Logging
=======================

You can optionally enable logging of errors, although this naturally comes with a performance penalty under extreme load.

How do I use it?
===================

Import the library

```go
import "github.com/StabbyCutyou/buffstreams"
```

Create a set of options for the BuffManager
```go
  cfg := buffstreams.BuffManagerConfig{
    MaxMessageSize: 2048,
    EnableLogging:  true,
  }
```
BuffStreams is all about the BuffManager.

```go
buffM := buffstreams.New(cfg)
```

The BuffManager has one primary function managing two kinds of TCP connections (Reading and Writing), designed to handle streaming data in the protocol buffers format.

The BuffManager can dial out to remote TCP endpoints

```go
buffM.DialOut("127.0.0.1", "5031")
```
It will store a reference to the connection internally, and synchronize access to it. In this way, a single BuffManager should be considered thread safe question marks???

And either write in a single-message fashion (which closed the connection)

```go
  bytesWritten, err := buffM.WriteTo("127.0.0.1", "5031", msg, false)
```

Or with a persistent connection in a streaming fashion

```go
  bytesWritten, err := buffM.WriteTo("127.0.0.1", "5031", msg, true)
```

If there is an error in writing, that connection will be closed and removed from the collection.

On the next write attempt, it will be opened anew

Additionally, BuffManager can listen on local ports for incoming requests. 

```go
buffM.StartListening("5031", ListenCallbackExample)
```

Again, BuffManager will keep hold of this socket, and all incoming connections internally to itself. It is nonblocking, so your program or library must continue to run while BuffStreams is listening and handling connections. It is not a daemon.

To listen requires a function delegate to be passed in, which meets the following interface:

```go
type ListenCallback func([]byte) error
```

The callback will receive the raw bytes for a given protobuff message. The header containing the size will have been removed. It is the callbacks responsibility to deserialize and act upon the message.

BuffManager only gets you there, you have to do the work.

A sample callback might start like so:

```go
  func ListenCallbackExample ([]byte data) error {
    msg := &message.ImportantProtoBuffStreamingMessage{}
    err := proto.Unmarshal(data, msg)
    // Now you do some stuff with msg
  }
```

The callback is currently run in it's own goroutine, which also handles reading from the connection until the reader disconnects, or there is an error. Any errors reading from a connection incoming will be up to the client to handle.