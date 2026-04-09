# How to Test P2P Client Multi-Stream Tools

This directory contains tools for publishing and subscribing to P2P network topics via gRPC streams.

## Prerequisites

- Go compiler installed
- A file with IP addresses of P2P nodes (format: IP:PORT, one per line), such as `ip.p2pnode.bnb.tsv`

## Building the Binaries

**Before using the tools, you must first build them:**

```bash
go build -o p2p-multi-publish p2p_client_multi_streams_publish.go
go build -o p2p-multi-subscribe p2p_client_multi_streams_subscribe.go
```

This will create two executables:
- `p2p-multi-publish` is the Publisher tool
- `p2p-multi-subscribe` is the Subscriber tool

## Publisher Tool

The publisher connects to multiple P2P nodes and publishes messages to a specified topic.

### Usage
```
./p2p-multi-publish [options]
```

### Options
  -count int         Number of messages to publish per node (default 1)
  -datasize int      Size of random message payload in bytes (default 100)
  -end-index int     Last IP index to use from the IP file (default 10000)
  -start-index int   First IP index to use from the IP file (default 0)
  -ipfile string     Path to file containing P2P node addresses
  -output string     File to write outgoing message hashes
  -poisson           Enable Poisson arrival distribution for publishing
  -sleep duration    Delay between consecutive publishes (e.g., 1s, 500ms) (default 50ms)
  -topic string      Topic name for publishing (required)

### Example Commands

Basic publish (2 messages to first 5 nodes):
```
./p2p-multi-publish -count 2 -datasize 100 -end-index 5 \
  -ipfile ip.p2pnode.bnb.tsv -output outgoing-hash.txt \
  -sleep 500ms -topic=test-topic
```

Production publish (3 messages to 20 nodes):
```
./p2p-multi-publish -count 3 -datasize 12222 -end-index 20 \
  -ipfile ip.p2pnode.bnb.tsv -output outgoing-data-hash.txt \
  -sleep 1s -topic=mytopic
```

### Output Format

The output file contains tab-separated values (tsv):
```
sender                  size    sha256(msg)
34.127.26.15:33212     119     03fb89f1791f9bd70fa959a8acbae9dd...
```

## Subscriber Tool

The subscriber connects to multiple P2P nodes and listens for messages on a specified topic.

### Usage
```
./p2p-multi-subscribe [options]
```

### Options
  -end-index int       Last IP index to use from the IP file (default 10000)
  -start-index int     First IP index to use from the IP file (default 0)
  -ipfile string       Path to file containing P2P node addresses
  -output-data string  File to write received message hashes
  -output-trace string File to write trace information
  -topic string        Topic name to subscribe to (required)

### Example Commands

Basic subscribe (first 5 nodes):
```
./p2p-multi-subscribe -end-index 5 -ipfile ip.p2pnode.bnb.tsv \
  -output-data incoming-hash.txt -topic=test-topic
```

Production subscribe (20 nodes):
```
./p2p-multi-subscribe -end-index 20 -ipfile ip.p2pnode.bnb.tsv \
  -output-data incoming-data-hash.txt -topic=mytopic
```

Note: The subscriber runs indefinitely until interrupted (Ctrl+C).

### Output Format

The output file contains tab-separated values:
```
receiver                sender          size    sha256(msg)
34.187.253.56:33212    [binary data]   12453   1b4bca4ab3703d64...
```

## Testing the Tools

1. In the first terminal, start the subscriber:
```
./p2p-multi-subscribe -end-index 5 -ipfile ip.p2pnode.bnb.tsv \
  -output-data test-incoming.txt -topic=test-topic
```

2. In another terminal, run the publisher:
```
./p2p-multi-publish -count 2 -datasize 100 -end-index 5 \
  -ipfile ip.p2pnode.bnb.tsv -output test-outgoing.txt \
  -sleep 500ms -topic=test-topic
```

3. Verify messages were received by checking the output files (test-outgoing.txt and test-incoming.txt)

