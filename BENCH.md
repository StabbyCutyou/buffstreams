Benchmarks
==========

Currently, I use the provided test_server and test_client to run benchmarks between instances in EC2 in the same region. Over the course of my testing, I've been able to consistently maintain throughputs of over 1 Million messages per second, which includes time for the server to deserialize a sample payload of data.

Although I don't have a thoroughly rigorous set of benchmarks yet, I have a sample output from a tool I wrote to analyze logs from the test_client to track how many messages per second they're writing. I'll eventually be publishing this tool as well, but for the time being here is a sample set of results.

When running the test_server with a GOMAXPROCS of 256 listening on a single socket, and running 16 test_clients each writing to the servers socket as fast as they can with the same 110 byte protobuffs payload, I'm able to achieve the following throughput:

Average Messages per Second: 1015612.3127317677
Maximum Messages per Second 1150467
Minimum Messages per Second 562772
Average Bytes per Second 111717354.40049444
Average MegaBytes per Second 106.5419715886063
Average Megabits per Second 852.3357727088504

Visually, this looks like so

![sample benchmark](http://i.imgur.com/RFtdPtW.png "Sample Benchmark")

