BuffStreams
====================

Still actively in development. Real readme coming soon.

What is BuffStreams?
=====================

It's a TCP connection manager for streaming Protobuffs, either in or out.

That's it.

Why ProtocolBuffers
====================

I was writing a few different projects for fun in Go, and kept writing code something like what is in the library. I decided to just generalize my case a little bit with a function pointer, as 95% of the rest of the code was the same. It uses protobuffs for no reason other than that is what I'm using at the time.

Since protobuff data lacks any kind of delimeter, you have a few options when it comes to sending messages in a streaming manner - provide your own delimeter, or use a pattern of sending a fixed set of bytes describing a message size, and a proceeding set of bytes as delimeted by the size in the header.

Buffstreams takes the latter approach for the time being. Maybe the other way is better. This is just what I did.

Although - Nothing in BuffStreams actually uses protobuffs in any way. It's just the pattern I came up with for handling streaming protobuffs - you could probably use it with any format that needs you to calculate how big it is and send it as a header. That's probably a thing.

What's it supposed to do?
========================

You can listen on ports, or dial out to other servers, and then keep a persistent connection where you shove data or pull data as fast as you can.

Long term it'll do some work to try and maintain connections it's dialed out (or maybe not, maybe thats a bad idea).

Really, who knows.

An Apology
=======================

I will apologize in advance for the pretty terrible names I chose for this library. It's way better than the original set of names I had for it. But that isn't saying much.

How do I use it?
===================

Import the library

```go
import "github.com/StabbyCutyou/buffstreams"
```

BuffStreams is all about the BuffManager.

```go
buffM := buffstreams.New()
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
buffM.StartListening("5031", TestCallback)
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

Logging
=======================
Logrus is in there right now (because it's great) to help me with testing.

It wont necessarily be a real dependency of the library, I want to offer the ability to sub in your logger of choice along with a proper way to configure the BuffManager class itself.