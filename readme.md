# OptimumP2P – Local Development Setup

This repository provides a full-stack setup for running OptimumP2P, a high-performance RLNC-enhanced pubsub protocol, along with multiple proxies for scalable message routing.
This repository provides a sample Docker Compose setup for deploying the OptimumP2P messaging infrastructure locally.
It demonstrates how partners can configure proxies and P2P nodes, and serves as a reference architecture for integration, testing, and scaling into production.

## Architecture

!['Proxy-P2P Node Communication'](./docs/intro.png)

### How it works

* Clients (like CLI, dApps, or backend services) interact with Proxies using REST/WebSocket.
* Proxies handle user subscriptions, publish requests, and stream delivery.
* Proxies connect to multiple P2P nodes via gRPC sidecar connections.
* P2P nodes form an RLNC-enhanced mesh network, forwarding messages via coded shards.
* Messages propagate based on configurable thresholds and shard redundancy.

> **Note:** OptP2P refers to the main set of P2P nodes forming the core mesh network. The Proxy acts as a proxy, providing controlled and secure access to all P2P nodes for external clients and integrations (e.g., Matrix, collaboration tools, etc.). For native integrations or advanced use cases, you can interact directly with the P2P mesh, bypassing the Proxy for full flexibility and performance.

**Important:** Proxies are stateless and horizontally scalable. P2P nodes form a resilient gossip + RLNC mesh.

## Purpose

This setup is not production-ready but is designed to:

* Show how to run multiple P2P nodes and proxies
* Demonstrate typical configuration options
* Help partners bootstrap their own network using OptimumP2P

**You are expected to modify this template based on your environment, infrastructure, and security needs.**

## What It Includes

* 4 P2P Nodes running the OptimumP2P
* 2 Proxies for client-facing APIs (HTTP/WebSocket)
* Static IP overlay (optimum-network) for deterministic internal addressing
* .env-based dynamic peer identity setup
* Optional Auth0 support (disabled by default)

## gRPC Flow Control Issue & Solution

> **Background:**
> Previously, direct gRPC subscriptions to a P2P node would stop working due to the default gRPC receive window (64KB) filling up. This caused the server to block on `Send()` and eventually deadlock, especially under high-throughput or real-time streaming.

**Solution:**
The P2P client now sets the gRPC per-stream and connection-level receive buffer to 1GB:
```go
grpc.WithInitialWindowSize(1024 * 1024 * 1024),
grpc.WithInitialConnWindowSize(1024 * 1024 * 1024),
```
This allows the subscriber to drain messages without stalling the sender, even under very high message volumes.

**New CLI Flags:**
- `-count` — Number of messages to publish (for stress testing)
- `-sleep` — Delay between publishes (e.g., 100ms)
- `-keepalive-interval` — gRPC keepalive ping interval (default: 2m)
- `-keepalive-timeout` — gRPC keepalive ping timeout (default: 20s)

---

## Example: Stress Test with 10,000+ Messages

```sh
# Subscribe (in one terminal)
sh ./script/p2p_client.sh 127.0.0.1:33221 subscribe my-topic -keepalive-interval=1m

# Publish 10,000 messages (in another terminal)
sh ./script/p2p_client.sh 127.0.0.1:33221 publish my-topic "random" -count=10000 -sleep=100ms
```

You can now reliably test with 10,000, 50,000, or more messages without the subscriber stalling.

---

## Repository Structure

```sh
optimum-dev-setup-guide/
├── keygen/                 # Key generation utilities
│   └── generate_p2p_key.go # P2P key generation implementation
├── grpc_p2p_client/        # P2P client implementation
│   ├── grpc/              # gRPC implementation
│   │   ├── p2p_stream.pb.go        # Generated protobuf message types
│   │   └── p2p_stream_grpc.pb.go   # Generated gRPC service definitions
│   ├── proto/             # Protocol buffer definitions
│   │   └── p2p_stream.proto        # P2P stream protocol definition
│   └── p2p_client.go      # Main P2P client implementation
├── grpc_proxy_client/    # Proxy client implementation
│   ├── grpc/              # gRPC implementation
│   │   ├── proxy_stream.pb.go        # Generated protobuf message types
│   │   └── proxy_stream_grpc.pb.go   # Generated gRPC service definitions
│   ├── proto/             # Protocol buffer definitions
│   │   └── proxy_stream.proto        # Proxy stream protocol definition
│   └── proxy_client.go   # Main Proxy client implementation
├── script/                # Utility scripts
│   ├── p2p_client.sh       # P2P client setup script
│   ├── proxy_client.sh   # Proxy client setup script
│   └── generate_p2p_key.sh # Key generation script
├── docker-compose.yml     # Docker compose configuration
├── test_suite.sh         # Test suite script
└── test_keepalive_fix.sh # gRPC keepalive testing script
```

## Prerequisites

Before getting started, ensure you have the following tools installed:

* Docker and Docker Compose
* Node.js and npm (for WebSocket testing)
* Golang (required for key generation script)
* wscat (WebSocket client for testing)

To install wscat:

```sh
npm install -g wscat
```

> **Note:** For key management, check out the key ring in the `keygen/` directory.

## Getting Started

### 1. Generate Bootstrap Peer Identity

This is required before launching the network.

```sh
sh ./script/generate_p2p_key.sh
```

It creates a file at `identity/p2p.key` and prints:

```sh
Peer ID: 12D3KooWJ5wcJWsfPmy6ssqonno14baQMozmteSkRGKxAzB3k2t8
```

Set it in your .env file:

```sh
cp .env.example .env
```

Edit .env:

```sh
BOOTSTRAP_PEER_ID=12D3KooWJ5wcJWsfPmy6ssqonno14baQMozmteSkRGKxAzB3k2t8
```

### 2. Launch the Sample Network

```sh
docker-compose up --build
```

#### Configuration

Default values are provided, but it's important to understand what each variable does.

##### Proxy Variables

* `CLUSTER_ID`: Proxy instance ID
* `PROXY_PORT`: Port on which the proxy serves REST/WebSocket API
* `P2P_NODES`: Comma-separated list of gRPC sidecar endpoints (e.g., `host:port`)
* `ENABLE_AUTH`: If true, JWT Auth0 is required; if false, API is open (local only) (default: false)
* `LOG_LEVEL`: Log verbosity level (e.g., `debug`)

##### P2P Node Variables

* `CLUSTER_ID`: Logical ID of the node
* `NODE_MODE`: `optimum` or `gossipsub` mode (should be `optimum`)
* `SIDECAR_PORT`: gRPC bidirectional port to which proxies connect (default: `33212`)
* `API_PORT`: HTTP port exposing internal node APIs (default: `9090`)
* `IDENTITY_DIR`: Directory containing node identity (p2p.key) (needed only for bootstrap node; each node generates its own on start)
* `BOOTSTRAP_PEERS`: Comma-separated list of peer multiaddrs with /p2p/ ID for initial connection
* `OPTIMUM_PORT`: Port used by the OptimumP2P (default: 7070)
* `OPTIMUM_MAX_MSG_SIZE`: Max message size in bytes (default: `1048576` → 1MB)
* `OPTIMUM_MESH_MIN`: Min number of mesh-connected peers (default: `3`)
* `OPTIMUM_MESH_MAX`: Max number of mesh-connected peers (default: `12`)
* `OPTIMUM_MESH_TARGET`: Target number of peers to connect to (default: `6`)
* `OPTIMUM_SHARD_FACTOR`: Number of shards per message (default: 4)
* `OPTIMUM_SHARD_MULT`: Shard size multiplier (default: 1.5)
* `OPTIMUM_THRESHOLD`: Minimum % of shard redundancy before forwarding message (e.g., 0.75 = 75%)

If you want to learn about mesh, see how [Eth2 is using gossipsub](https://github.com/LeastAuthority/eth2.0-specs/blob/dev/specs/phase0/p2p-interface.md#the-gossip-domain-gossipsub).

## Use Cases

You can use this stack to:

* Run local tests with mump2p-cli
* Validate publish/subscribe mechanics via REST or WebSocket
* Simulate client/proxy/node interaction
* Customize clustering, shard behavior, and thresholds

## Ways of Using It

There are two main ways to use this setup:

1. **Proxy:** Connects to all P2P nodes and provides responses based on a threshold (e.g., whether 1-100% of nodes have received the message).
2. **P2P Nodes:** Interact directly with a P2P node.

## Proxy API

Proxies provide the user-facing interface to OptimumP2P.

### Subscribe to Topic

```sh
curl -X POST http://localhost:8081/api/subscribe \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "your-client-id", 
    "topic": "example-topic",
    "threshold": 0.7
  }'
```

* `client_id`: Used to identify the client across WebSocket sessions. Auth0 user_id recommended if JWT is used.
* `threshold`: Forward message to this client only if X% of nodes have received it.

Here, `client_id` is your WebSocket connection identifier. Usually, we use Auth0 `user_id` to have a controlled environment, but here you can use any random string. Make sure to use the same string when making the WebSocket connection to receive the message.

### WebSocket Connection

```sh
wscat -c "ws://localhost:8081/api/ws?client_id=your-client-id"
```

This is how clients receive messages published to their subscribed topics.

> **Important:** WebSocket has limitations, and you may experience unreliable delivery when publishing message bursts. A gRPC connection will be provided in a future update.

### Publish Message

```sh
curl -X POST http://localhost:8081/api/publish \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "your-client-id",
    "topic": "example-topic",
    "message": "Hello, world!"
  }'
```

> **Important:** The `client_id` field is required for all publish requests. This should be the same ID used when subscribing to topics. If you're using WebSocket connections, use the same `client_id` for consistency.

## Proxy gRPC Streaming

Clients can use gRPC to stream messages from the Proxy.

Protobuf: `proxy_stream.proto`

```proto
service ProxyStream {
  rpc ClientStream (stream ProxyMessage) returns (stream ProxyMessage);
}

message ProxyMessage {
  string client_id = 1;
  bytes message = 2;
  string topic = 3;
  string message_id = 4;
  string type = 5; 
}
```

* Bidirectional streaming.
* First message must include only client_id.
* All subsequent messages are sent by Proxy on subscribed topics.

### Example

```sh
sh ./script/proxy_client.sh subscribe mytopic 0.7
sh ./script/proxy_client.sh publish mytopic 0.6 10
```

## gRPC Proxy Client Implementation

> **Note:** The provided client code in `grpc_proxy_client/proxy_client.go` is a SAMPLE implementation intended for demonstration and testing purposes only. It is **not production-ready** and should not be used as-is in production environments. Please review, adapt, and harden the code according to your security, reliability, and operational requirements before any production use.

A new Go-based gRPC client implementation is available in `grpc_proxy_client/proxy_client.go` that provides:

### Features

* **Bidirectional gRPC Streaming**: Establishes persistent connection with the proxy
* **REST API Integration**: Uses REST for subscription and publishing
* **Automatic Client ID Generation**: Generates unique client identifiers
* **Configurable Keepalive**: Optimized gRPC keepalive settings
* **Message Publishing Loop**: Automated message publishing with configurable delays
* **Signal Handling**: Graceful shutdown on interrupt

### Usage

```sh
# Build the client
cd grpc_proxy_client
go build -o proxy_client proxy_client.go

# Subscribe only (receive messages)
./proxy_client -subscribeOnly -topic=test -threshold=0.7

# Subscribe and publish messages
./proxy_client -topic=test -threshold=0.7 -count=10 -delay=2s

# Custom keepalive settings
./proxy_client -topic=test -keepalive-interval=5m -keepalive-timeout=30s
```

### Command Line Flags

* `-topic`: Topic name to subscribe/publish (default: "demo")
* `-threshold`: Delivery threshold 0.0 to 1.0 (default: 0.1)
* `-subscribeOnly`: Only subscribe and receive messages
* `-count`: Number of messages to publish (default: 5)
* `-delay`: Delay between message publishing (default: 2s)
* `-keepalive-interval`: gRPC keepalive interval (default: 2m)
* `-keepalive-timeout`: gRPC keepalive timeout (default: 20s)

### Protocol Flow

1. **Subscription**: Client subscribes to topic via REST API
2. **gRPC Connection**: Establishes bidirectional stream with proxy
3. **Client ID Registration**: Sends client_id as first message
4. **Message Reception**: Receives messages on subscribed topics
5. **Message Publishing**: Publishes messages via REST API (optional)

### Generated Protobuf Files

The gRPC clients use auto-generated protobuf files:

**Proxy Client:**
* `grpc_proxy_client/grpc/proxy_stream.pb.go`: Message type definitions
* `grpc_proxy_client/grpc/proxy_stream_grpc.pb.go`: gRPC service definitions

**P2P Client:**
* `grpc_p2p_client/grpc/p2p_stream.pb.go`: Message type definitions
* `grpc_p2p_client/grpc/p2p_stream_grpc.pb.go`: gRPC service definitions

To regenerate these files:

**Proxy Client:**
```sh
cd grpc_proxy_client
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/proxy_stream.proto
```

**P2P Client:**
```sh
cd grpc_p2p_client
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/p2p_stream.proto
```

## Using P2P Nodes Directly (Optional – No Proxy)

If you prefer to interact directly with the P2P mesh, bypassing the proxy, you can use a minimal client script to publish and subscribe directly over the gRPC sidecar interface of the nodes.

This is useful for:

* Low-level integration
* Bypassing HTTP/WebSocket stack
* Simulating internal services or embedded clients

### Example (sample implementation)

#### Subscribe to a Topic

```sh
sh ./script/p2p_client.sh localhost:33221 subscribe mytopic
```

> **Note:** Here, `localhost:33221` is the mapped port for `p2pnode-1` (33221:33212) in docker-compose.

response

```sh
Subscribed to topic "mytopic", waiting for messages…
Received message: "random"
Received message: "random1"
Received message: "random2"
```

#### Publish to a Topic

```sh
sh ./script/p2p_client.sh localhost:33222 publish mytopic random
```

> **Note:** Here, `localhost:33222` is the mapped port for `p2pnode-2` (33222:33212) in docker-compose.

response

```sh
Published "random" to "mytopic"
```

* --addr refers to the sidecar gRPC port exposed by the P2P node (e.g., 33221, 33222, etc.)
* Messages published here will still follow RLNC encoding, mesh forwarding, and threshold policies
* Proxy(s) will pick these up only if enough nodes receive the shards (threshold logic)

## gRPC Keepalive Configuration

The P2P client has been updated to handle gRPC keepalive settings properly to avoid connection issues. The default settings have been optimized to prevent "too_many_pings" errors that can occur with aggressive keepalive configurations.

### Default Settings

The client now uses these improved default keepalive settings:

* **Ping Interval**: 2 minutes (instead of 30 seconds)
* **Ping Timeout**: 20 seconds
* **Permit Without Stream**: true

### Customizing Keepalive Settings

You can customize the keepalive behavior using command-line flags:

```sh
# Use 5-minute ping intervals
./p2p_client/p2p-client -mode=subscribe -topic=test --addr=127.0.0.1:33221 -keepalive-interval=5m

# Use 10-second ping timeout
sh ./script/p2p_client.sh 127.0.0.1:33221 subscribe my-topic -keepalive-timeout=10s

# Combine both settings
./p2p_client/p2p-client -mode=subscribe -topic=test --addr=127.0.0.1:33221 -keepalive-interval=3m -keepalive-timeout=15s
```

### Available Keepalive Flags

* `-keepalive-interval`: gRPC keepalive ping interval (default: 2m0s)
* `-keepalive-timeout`: gRPC keepalive ping timeout (default: 20s)

### Stress Testing Flags

* `-count`: Number of messages to publish (default: 1)
* `-sleep`: Delay between publishes (e.g., 100ms, 1s)

### Example: High-Volume Publish

```sh
sh ./script/p2p_client.sh 127.0.0.1:33221 publish my-topic "random" -count=50000 -sleep=50ms
```

### Troubleshooting Keepalive & Flow Control Issues

If you encounter keepalive-related errors:

1. **"too_many_pings" error**: Increase the `-keepalive-interval` value
2. **Connection timeouts**: Decrease the `-keepalive-timeout` value
3. **Server compatibility**: Some servers have strict ping limits; use conservative settings

If you encounter message stalls at high volume, ensure you are using the latest client with the gRPC window size fix (see above).

## Inspecting P2P Nodes

### Get Node Health

```sh
curl http://localhost:9091/api/v1/health
```

response:

```json
{"cpu_used":"0.29","disk_used":"84.05","memory_used":"6.70","mode":"optimum","status":"ok"}
```

### Get Node State

```sh
curl http://localhost:9091/api/v1/node-state
```

response:

```json
{
  "pub_key": "12D3KooWMwzQYKhRvLRsiKignZ2vAtkr1nPYiDWA7yeZvRqC9ST9",
  "peers": [
    "12D3KooWDLm7bSFnoqP4mhoJminiCixbR2Lwyqu9sU5EDKVvXM5j",
    "12D3KooWJrPmTdXj9hirigHs88BHe6DApLpdXiKrwF1V8tNq9KP7",
    "12D3KooWAykKBmimGzgFCC6EL3He3tTqcy2nbVLGCa1ENrG2x5QP"
  ],
  "addresses": ["/ip4/172.28.0.12/tcp/7070"],
  "topics": []
}
```

### Get Node Version

```sh
curl http://localhost:9091/api/v1/version
```

response:

```sh
{"commit_hash":"6d3d086","version":""}
```

## Developer Tools

You can use CLI for testing as well that connects to proxy

See CLI guide: [mump2p-cli](https://github.com/getoptimum/mump2p-cli)

## Testing

### Automated Test Suite

Run the comprehensive test suite to validate API endpoints and edge cases:

```sh
./test_suite.sh
```

**What it tests:**
- Proxy API endpoints (subscribe, publish, health, state, version)
- Input validation (empty fields, invalid JSON)
- Rapid request handling (5x publish test)
- WebSocket connection (if wscat is installed)
- Edge cases and error handling



### gRPC Keepalive Testing

Test the gRPC keepalive fix to ensure stable connections:

```sh
./test_keepalive_fix.sh
```

**What it tests:**
- Default keepalive settings (2m interval)
- Custom keepalive configurations (5m interval)
- Script compatibility with keepalive flags
- Publish functionality with keepalive settings

