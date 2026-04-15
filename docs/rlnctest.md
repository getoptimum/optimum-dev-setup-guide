# P2P Binary Deployment & Testing Guide

## Step 1 — Push Changes to `optimum-p2p`

Ensure all your changes have been pushed to the `optimum-p2p` repository before proceeding.

---

## Step 2 — Update the Dependency in `optimum-proxy`

In the `optimum-proxy` repository, open `go.mod` and update the `require` section to reference the newly pushed version of `optimum-p2p`:

```
github.com/getoptimum/optimum-p2p <new-version>
```

Replace `<new-version>` with the version tag you just pushed (e.g. `v0.0.1-rc13`), then run:

```bash
go get github.com/getoptimum/optimum-p2p@<new-version>
go mod tidy
```

---

## Step 3 — Build the Binary

From the `optimum-proxy` repository root, build the P2P binary:

```bash
make p2p
```

---

## Step 4 — Deploy the Binary

Move the compiled binary to the infrastructure repository:

```bash
mv ./p2p optimum-infra/optimump2p-native/data/p2pnode-rlnc-arch-test
```

---

## Step 5 — Start Local Nodes

Run 50 local nodes:

```bash
./run-local 50
```

---

## Step 6 — Start the RLNC Server

Build and start the RLNC server:

```bash
go run cmd/server/main.go
```

---

## Step 7 — Verify All Nodes Are Running

```bash
./run-local list
```

Confirm that all 50 processes appear in the output before proceeding.

---

## Step 8 — Test Publish/Subscribe via gRPC

Use the publish/subscribe tools from the `optimum-dev-local-setup` repository. See below for full usage instructions.

---

# P2P Client Multi-Stream Tools

Tools for publishing and subscribing to P2P network topics via gRPC streams.

## Prerequisites

- Go compiler installed
- A file containing P2P node addresses in `IP:PORT` format, one per line (e.g. `ip.p2pnode.bnb.tsv`)

---

## Building the Binaries

Build both tools before use:

```bash
go build -o p2p-multi-publish p2p_client_multi_streams_publish.go
go build -o p2p-multi-subscribe p2p_client_multi_streams_subscribe.go
```

This produces two executables:

| Binary | Role |
|---|---|
| `p2p-multi-publish` | Publishes messages to a topic across multiple nodes |
| `p2p-multi-subscribe` | Subscribes to a topic and listens for messages across multiple nodes |

---

## Publisher

Connects to multiple P2P nodes and publishes messages to a specified topic.

### Usage

```bash
./p2p-multi-publish [options]
```

### Options

| Flag | Type | Default | Description |
|---|---|---|---|
| `-topic` | string | *(required)* | Topic name to publish to |
| `-ipfile` | string | — | Path to file containing node addresses |
| `-count` | int | 1 | Number of messages to publish per node |
| `-datasize` | int | 100 | Size of random message payload in bytes |
| `-start-index` | int | 0 | First index to use from the IP file |
| `-end-index` | int | 10000 | Last index to use from the IP file |
| `-output` | string | — | File to write outgoing message hashes |
| `-sleep` | duration | 50ms | Delay between consecutive publishes (e.g. `1s`, `500ms`) |
| `-poisson` | bool | false | Enable Poisson arrival distribution for publishing |

### Examples

**Basic** — publish 2 messages to the first 5 nodes:
```bash
./p2p-multi-publish \
  -count 2 -datasize 100 \
  -start-index 0 -end-index 5 \
  -ipfile ip.p2pnode.bnb.tsv \
  -output outgoing-hash.txt \
  -sleep 500ms -topic test-topic
```

**Production** — publish 3 messages to 20 nodes:
```bash
./p2p-multi-publish \
  -count 3 -datasize 12222 \
  -start-index 0 -end-index 20 \
  -ipfile ip.p2pnode.bnb.tsv \
  -output outgoing-data-hash.txt \
  -sleep 1s -topic mytopic
```

### Output Format

The output file is tab-separated (TSV):

```
sender                  size    sha256(msg)
34.127.26.15:33212      119     03fb89f1791f9bd70fa959a8acbae9dd...
```

---

## Subscriber

Connects to multiple P2P nodes and listens for messages on a specified topic. Runs indefinitely until interrupted with `Ctrl+C`.

### Usage

```bash
./p2p-multi-subscribe [options]
```

### Options

| Flag | Type | Default | Description |
|---|---|---|---|
| `-topic` | string | *(required)* | Topic name to subscribe to |
| `-ipfile` | string | — | Path to file containing node addresses |
| `-start-index` | int | 0 | First index to use from the IP file |
| `-end-index` | int | 10000 | Last index to use from the IP file |
| `-output-data` | string | — | File to write received message hashes |
| `-output-trace` | string | — | File to write trace information |

### Examples

**Basic** — subscribe on first 5 nodes:
```bash
./p2p-multi-subscribe \
  -end-index 5 \
  -ipfile ip.p2pnode.bnb.tsv \
  -output-data incoming-hash.txt \
  -topic test-topic
```

**Production** — subscribe on 20 nodes:
```bash
./p2p-multi-subscribe \
  -end-index 20 \
  -ipfile ip.p2pnode.bnb.tsv \
  -output-data incoming-data-hash.txt \
  -topic mytopic
```

### Output Format

The output file is tab-separated (TSV):

```
receiver                sender          size    sha256(msg)
34.187.253.56:33212     [binary data]   12453   1b4bca4ab3703d64...
```

---

## End-to-End Test

**Terminal 1 — start the subscriber:**
```bash
./p2p-multi-subscribe \
  -end-index 5 \
  -ipfile ip.p2pnode.bnb.tsv \
  -output-data test-incoming.txt \
  -topic test-topic
```

**Terminal 2 — run the publisher:**
```bash
./p2p-multi-publish \
  -count 2 -datasize 100 \
  -end-index 5 \
  -ipfile ip.p2pnode.bnb.tsv \
  -output test-outgoing.txt \
  -sleep 500ms -topic test-topic
```

Verify that message hashes in `test-outgoing.txt` appear in `test-incoming.txt`.

