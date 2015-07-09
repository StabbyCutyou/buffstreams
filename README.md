BuffStreams [![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/StabbyCutyou/buffstreams)
====================

Streaming Protocol Buffers messages over TCP in Golang

What is BuffStreams?
=====================

BuffStreams is a manager for streaming TCP connections that write data in a format involving the length of the message + the message payload itself.

BuffStreams gives you a simple interface to start a nonblocking listener on a given port, which will stream arrays of raw bytes into a callback you provide it. In this way, BuffStreams is not so much a daemon, but a library to build networked services that can  communicate over TCP using Protocol Buffer messages.

Why BuffStreams
====================

I was writing a few different projects for fun in Golang, and kept writing code something like what is in the library, but less organized. I decided to focus on the networking code, pulling it out and improving it so I knew it could be trusted to perform reliably across projects. 

How does it work?
=================

Since protobuff messages lack any kind of natural delimeter, BuffStreams uses the method of adding a fixed header of bytes (which is configurable) that describes the size of the actual payload. This is handled for you, by the call to write. You never need to pack on the size yourself.

On the server side, it will listen for these payloads, read the fixed header, and then the subsequent message. The server must have the same maximum size as the client for this to work. BuffStreams will then pass the byte array to a callback you provided for handling messages received on that port. Deserializing the messages and interpreting their value is up to you.

One important note is that internally, BuffStreams does not actually use or rely on the Protocol Buffers library itself in any way. All serialization / deserialization is handled by the client prior to / after interactions with BuffStreams. In this way, you could theoretically use this library to stream any data over TCP that uses the same strategy of a fixed header of bytes + a subsequent message body. 

Currently, I have only used it for ProtocolBuffers messages.

Naming Strategy
=======================

I will apologize in advance for the pretty terrible names I chose for this library. It's way better than the original set of names I had for it. But that isn't saying much.

Logging
=======================

You can optionally enable logging of errors, although this naturally comes with a performance penalty under extreme load.

Benchmarks
==========

I've tried very hard to optimize BuffStreams as best as possible, striving to keep it's averages above 1M messages per second, with no errors during transit.

See [Bench](https://github.com/StabbyCutyou/buffstreams/blob/master/BENCH.md)

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
Now, create a new BuffManager using those options.

```go
buffM := buffstreams.New(cfg)
```

From there, you can begin writing over a socket. You need an address in the format of "name:port". You can use a helper method to generate one, if you want

```go
  address := buffstreams.FormatAddress("127.0.0.1", strconv.Itoa(5031))
```

Using BuffManager to Write Streaming Messages
==============================================

Once you have a BuffManager, you can now write data over the socket

```go
bytesWritten, err := bm.WriteTo(address, msgBytes, true)
```

BuffStreams will store a reference to the connection internally, and synchronize access to it. In this way, a single BuffManager should be considered safe to use across goroutines.

The third argument to WriteTo controls whether or not to close the connection after the write. By keeping the connection open, you're able to treat the socket as a stream, continuously writing to it as fast as you can.

```go
  bytesWritten, err := buffM.WriteTo("127.0.0.1", "5031", msg, true)
```

If you provide false, the connection is closed immediately after the write, and will be reopened the next time you attempt to use it. In this way, you can use BuffManager to make short, one time calls to other servers.

```go
  bytesWritten, err := buffM.WriteTo("127.0.0.1", "5031", msg, false)
```

If there is an error in writing, that connection will be closed and be reopened on the next write. There is no guarantee if any the bytesWritten value will be >0 or not.

Using BuffManager to Receive Streaming Messages

Additionally, a BuffManager can listen on local ports for incoming requests. 

```go
buffM.StartListening("5031", ListenCallbackExample)
```

Again, BuffManager will keep hold of this socket, and all incoming connections internally to itself. It is nonblocking, so your program or library must continue to run while BuffStreams is listening and handling connections. It will not self-daemonize.

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

Roadmap
=======
* Release proper set of benchmarks, including more real-world cases
* Further improvements to the readme
* Further library optimizations via tools such as pprof
* Reference implementation
* Various TODO improvements littering the code to be taken care of
* Have a proper Roadmap

LISCENSE
=========
Apache v2 - See LICENSE