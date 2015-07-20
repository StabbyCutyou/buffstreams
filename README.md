BuffStreams [![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/StabbyCutyou/buffstreams) [![Build Status](https://travis-ci.org/StabbyCutyou/buffstreams.svg?branch=feature%2Fstart_of_tests)](https://travis-ci.org/StabbyCutyou/buffstreams)
====================

Streaming Protocol Buffers messages over TCP in Golang

What is BuffStreams?
=====================

BuffStreams is a set of abstraction over TCPConns for streaming  connections that write data in a format involving the length of the message + the message payload itself (like Protocol Buffers, hence the name).

BuffStreams gives you a simple interface to start a (blocking or non) listener on a given port, which will stream arrays of raw bytes into a callback you provide it. In this way, BuffStreams is not so much a daemon, but a library to build networked services that can communicate over TCP using Protocol Buffer messages.

Why BuffStreams
====================

I was writing a few different projects for fun in Golang, and kept writing code something like what is in the library, but less organized. I decided to focus on the networking code, pulling it out and improving it so I knew it could be trusted to perform reliably across projects.

Ethos
=====

There is nothing special or magical about Buffstreams, or the code in here. The idea isn't that it's a better, faster socket abstraction - it's to do as much of the boilerplate for you when handling streaming data like protobuff messages, with as little impact to performance as possible. Currently, Buffstreams is able to do over 1.1 million messsages per second, at 110 bytes per message on a single listening socket which saturates a 1gig nic.

The idea of Buffstreams is to do the boring parts and handle common errors, enabling you to write systems on top of it, while performing with as little overhead as possible.

How does it work?
=================

Since protobuff messages lack any kind of natural delimeter, BuffStreams uses the method of adding a fixed header of bytes (which is configurable) that describes the size of the actual payload. This is handled for you, by the call to write. You never need to pack on the size yourself.

On the server side, it will listen for these payloads, read the fixed header, and then the subsequent message. The server must have the same maximum size as the client for this to work. BuffStreams will then pass the byte array to a callback you provided for handling messages received on that port. Deserializing the messages and interpreting their value is up to you.

One important note is that internally, BuffStreams does not actually use or rely on the Protocol Buffers library itself in any way. All serialization / deserialization is handled by the client prior to / after interactions with BuffStreams. In this way, you could theoretically use this library to stream any data over TCP that uses the same strategy of a fixed header of bytes + a subsequent message body.

Currently, I have only used it for ProtocolBuffers messages.

Logging
=======================

You can optionally enable logging of errors, although this naturally comes with a performance penalty under extreme load.

Benchmarks
==========

I've tried very hard to optimize BuffStreams as best as possible, striving to keep it's averages above 1M messages per second, with no errors during transit.

See [Bench - OUTDATED](https://github.com/StabbyCutyou/buffstreams/blob/master/BENCH.md)

How do I use it?
===================

Download the library

```go
go get "github.com/StabbyCutyou/buffstreams"
```

Import the library

```go
import "github.com/StabbyCutyou/buffstreams"
```

Listening for connections
=========================

One of the core objects in Buffstreams is the BuffTCPListener. This struct allows you to open a socket on a local port, and begin waiting for clients to connect. Once a connection is made, each full message written by the client will be received by the Listener, and a callback you define will be invoked with the message contents (an array of bytes).

To begin listening, first create a BuffTCPListenerConfig object to define how the listener should behave. A sample BuffTCPListenerConfig might look like this:

```go
cfg := BuffTCPListenerConfig {
  EnableLogging: false, // true will have log messages printed to stdout/stderr, via log
  MaxMessageSize: 4098,
  Callback: func(byte[])error{return nil} // Any function type that adheres to this signature, you'll need to deserialize in here if need be
  Address: FormatAddress("", strconv.Itoa(5031)) // Any address with the pattern ip:port. The FormatAddress helper is here for convenience. For listening, you normally don't want to provide an ip unless you have a reason.
}
```

```go
btl, err := buffstreams.ListenBuffTCP(cfg)
```
Once you've opened a listener this way, the socket is now in use, but the listener itself has not yet begun to accept connections.

To do so, you have two choices. By default, this operation will block the current thread. If you want to avoid that, and use a fire and forget approach, you can call

```go
err := btl.StartListeningAsync()
```

If there is an error while starting up, it will be returned by this method. Alternatively, if you want to handle running the call yourself, or don't care that it blocks, you can call

```go
err := btl.StartListening()
```

The ListenCallback
==================

The way Buffstreams handles acting over the incoming messages is to let you provide a callback to operate on the bytes. ListenCallback takes in an array/slice of bytes, and returns an error.

```go
type ListenCallback func([]byte) error
```

The callback will receive the raw bytes for a given protobuff message. The header containing the size will have been removed. It is the callbacks responsibility to deserialize and act upon the message.

The Listener gets the message, your callback does the work.

A sample callback might start like so:

```go
  func ListenCallbackExample ([]byte data) error {
    msg := &message.ImportantProtoBuffStreamingMessage{}
    err := proto.Unmarshal(data, msg)
    // Now you do some stuff with msg
    ...
  }
```

The callback is currently run in it's own goroutine, which also handles reading from the connection until the reader disconnects, or there is an error. Any errors reading from a connection incoming will be up to the client to handle.

Writing messages
================

To begin writing messages, you'll need to dial a BuffTCPWriter using BuffTCPWriterConfig

```go
cfg := BuffTCPWriterConfig {
  EnableLogging: false, // true will have log messages printed to stdout/stderr, via log
  MaxMessageSize: 4098, // You want this to match the MaxMessageSize the server expects for messages on that socket
  Address: FormatAddress("127.0.0.1", strconv.Itoa(5031)) // Any address with the pattern ip:port. The FormatAddress helper is here for convenience.
}
```

Once you have a configuration object, you can Dial out.

```go
btw, err := buffstreams.Dial(cfg)
```

This will open a connection to the endpoint at the specified location. From there, you can write your data

```go
  bytesWritten, err := btw.Write(msgBytes, true)
```

If there is an error in writing, that connection will be closed and be reopened on the next write. There is no guarantee if any the bytesWritten value will be >0 or not in the event of an error which results in a reconnect.

BuffManager
===========

There is a third option, the provided BuffManager class. This class will give you a simple but effective Manager abstraction over dialing and listening over ports, managing the connections for you. You provide the normal configuration for dialing out or listening for incoming connections, and let the manager hold onto the references.

Creating a BuffManager

```go
bm := buffstreams.NewBuffManager()
```

Listening on a port. BuffManager always makes this asyncrhonous and non blocking

```go
// Assuming you've got a configuration object cfg, see above
err := bm.StartListening(cfg)
```

Dialing out to a remote endpoint

```go
// Assuming you've got a configuration object cfg, see above
err := bm.Dial(cfg)
```

Having opened a connection, writing to that connection in a constant fashion

```go
bytesWritten, err := bm.Write("127.0.0.1:5031", dataBytes)
```

The BuffManager will use locks to internally maintain threadsafety

Roadmap
=======
* Release proper set of benchmarks, including more real-world cases
* Configurable retry for the client, configurable errored-message queue for user to define failover process to handle.
* Further library optimizations via tools such as pprof
* gb maybe?

LICENSE
=========
Apache v2 - See LICENSE
