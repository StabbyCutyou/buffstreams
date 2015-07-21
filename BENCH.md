Benchmarks
==========

Currently, I use the provided test_server and test_client to run benchmarks between instances in EC2 in the same region. Over the course of my testing, I've been able to consistently maintain throughputs of over 1 Million messages per second, which includes time for the server to deserialize a sample payload of data.

There are also some benchmark tests in the test suite, however by default they run local to local, which is not necessarily a great representation of network performance.

When running the test_server with a GOMAXPROCS of 8 listening on a single socket, and running 16 test_clients processes each writing to the servers socket as fast as they can with the same 110 byte protobuffs payload in a single go routine, I'm able to achieve the following throughput:

```
Average Messages per Second: 1129942.6507177034
Maximum Messages per Second 1277944
Minimum Messages per Second 1006401
Average Bytes per Second 124293691.57894738
Average MegaBytes per Second 118.53570135016192
Average Megabits per Second 948.2856108012953
```

Visually, this looks like so

![sample benchmark](http://i.imgur.com/TWoqXYj.png "Sample Benchmark")
