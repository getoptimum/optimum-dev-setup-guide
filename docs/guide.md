# OptimumP2P Development Setup - Complete Guide

*Complete guide for setting up and using the OptimumP2P development environment.*

## Table of Contents

- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
  - [Generate Bootstrap Identity](#1-generate-bootstrap-identity)
  - [Start Services](#2-start-services)
  - [Test Everything](#3-test-everything)
- [Configuration](#configuration)
  - [Proxy Variables](#proxy-variables)
  - [P2P Node Variables](#p2p-node-variables)
  - [One-Command Setup](#one-command-setup-alternative)
- [Use Cases](#use-cases)
- [Two Ways to Connect](#two-ways-to-connect)
- [API Reference](#api-reference)
  - [Proxy API](#proxy-api)
  - [Proxy gRPC Streaming](#proxy-grpc-streaming)
  - [P2P Node API](#p2p-node-api)
- [Client Tools](#client-tools)
  - [gRPC Proxy Client](#grpc-proxy-client-implementation)
  - [Using P2P Nodes Directly](#using-p2p-nodes-directly-optional--no-proxy)
  - [Publishing Options](#publishing-options)
  - [Inspecting P2P Nodes](#inspecting-p2p-nodes)
  - [Collecting Trace Data for Experiments](#collecting-trace-data-for-experiments)
- [Troubleshooting](#troubleshooting)
- [Advanced Configuration](#advanced-configuration)
- [Monitoring and Telemetry](#monitoring-and-telemetry)

---

## Prerequisites

Install these tools before starting:

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

### 1. Generate Bootstrap Identity

```sh
./script/generate-identity.sh
```

This creates the bootstrap peer identity needed for P2P node discovery.

### 2. Start Services

```sh
docker-compose up --build
```

### 3. Test Everything

```sh
./test_suite.sh
```

## Build and Development Commands

### Makefile Commands

The project includes a Makefile with convenient shortcuts for common development tasks:

```sh
# Show all available commands and usage examples
make help

# Build all client binaries
make build

# Generate P2P identity (if missing)
make generate-identity

# Clean build artifacts
make clean
```

### Direct Binary Usage

After building with `make build`, you can use the binaries directly:

```sh
# P2P Client Help
./grpc_p2p_client/p2p-client --help

# Subscribe to a topic
./grpc_p2p_client/p2p-client -mode=subscribe -topic=testtopic --addr=127.0.0.1:33221

# Publish messages
./grpc_p2p_client/p2p-client -mode=publish -topic=testtopic -msg=HelloWorld --addr=127.0.0.1:33222

# Publish multiple messages with delay
./grpc_p2p_client/p2p-client -mode=publish -topic=testtopic --addr=127.0.0.1:33222 -count=5 -sleep=1s
```

**Example Output:**
```
# Subscribe output:
Connecting to node at: 127.0.0.1:33221…
Trying to subscribe to topic testtopic…
Subscribed to topic "testtopic", waiting for messages…
[1] Received message: "HelloWorld"

# Publish output:
Connecting to node at: 127.0.0.1:33222…
Published "HelloWorld" to "testtopic"
```

## Configuration

Default values are provided, but it's important to understand what each variable does.

### Proxy Variables

* `CLUSTER_ID`: Proxy instance ID
* `PROXY_PORT`: Port on which the proxy serves REST/WebSocket API
* `P2P_NODES`: Comma-separated list of gRPC sidecar endpoints (e.g., `host:port`)
* `ENABLE_AUTH`: If true, JWT Auth0 is required; if false, API is open (local only) (default: false)
* `LOG_LEVEL`: Log verbosity level (e.g., `debug`)

### P2P Node Variables

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

If you want to learn about mesh networking, see how [Eth2 uses gossipsub](https://github.com/LeastAuthority/eth2.0-specs/blob/dev/specs/phase0/p2p-interface.md#the-gossip-domain-gossipsub).

### One-Command Setup (Alternative)

You can also generate the identity with a single command:

```bash
curl -sSL https://raw.githubusercontent.com/getoptimum/optimum-dev-setup-guide/main/script/generate-identity.sh | bash
```

This downloads and runs the same identity generation script, creating the bootstrap peer identity and setting the environment variable.

## Use Cases

You can use this setup to:

* Test local applications with OptimumP2P
* Learn publish/subscribe mechanics via REST or WebSocket
* Simulate client/proxy/node interactions
* Experiment with clustering, sharding, and thresholds

## Two Ways to Connect

1. **Via Proxy** (recommended): Connect to proxies for managed access with authentication and rate limiting
2. **Direct P2P**: Connect directly to P2P nodes for low-level integration

---

## Architecture Overview

The OptimumP2P development setup provides a complete messaging infrastructure with:

### Core Components

- **P2P Node**: Form the RLNC-enhanced mesh network
- **Proxy**: Provide client-facing APIs (REST, gRPC, WebSocket)
- **Static Network Overlay**: Ensures deterministic internal addressing

### Communication Flow

1. **Clients** connect to Proxies via REST/gRPC/WebSocket
2. **Proxies** handle authentication, rate limiting, and message routing
3. **Proxies** connect to P2P nodes via gRPC sidecar interfaces
4. **P2P nodes** form a resilient mesh using RLNC coding for message propagation

### Key Features

- ✅ **JWT Authentication**: Auth0 integration with custom claims
- ✅ **Rate Limiting**: Per-client publish rate and quota enforcement
- ✅ **Telemetry**: Comprehensive Prometheus metrics
- ✅ **Message Deduplication**: Prevents duplicate message delivery
- ✅ **Threshold-based Delivery**: Configurable redundancy requirements
- ✅ **Multiple Protocols**: REST, gRPC, and WebSocket support

---

## Setup and Installation

### 1. Bootstrap Identity Generation

Generate the P2P bootstrap identity for node discovery:

```sh
./script/generate-identity.sh
```

**Output:**
```text
[SUCCESS] Generated P2P identity successfully!
[SUCCESS] Identity saved to: ./identity/p2p.key
[SUCCESS] Peer ID: 12D3KooWLsSmLLoE2T7JJ3ZyPqoXEusnBhsBA1ynJETsziCKGsBw
```

### 2. Service Startup

```sh
# Start all services in detached mode
docker-compose up --build -d

# Check service status
docker-compose ps
```

**Expected Services:**
- `proxy-1`: HTTP :8081, gRPC :50051
- `proxy-2`: HTTP :8082, gRPC :50052  
- `p2pnode-1`: API :9091, Sidecar :33221
- `p2pnode-2`: API :9092, Sidecar :33222
- `p2pnode-3`: API :9093, Sidecar :33223
- `p2pnode-4`: API :9094, Sidecar :33224

### 3. Verification

Run the comprehensive test suite:

```sh
./test_suite.sh
```
---

## Troubleshooting

If you encounter issues during setup, here are common problems and solutions:

**"node not found" errors:**
- Ensure all P2P nodes have access to the identity file (volume mounts are configured correctly)
- Verify the `.env` file contains the correct `BOOTSTRAP_PEER_ID`
- Check that the identity file was generated using the correct script

**"checksum mismatch" errors:**
- Delete the `identity/` directory and regenerate using `./script/generate-identity.sh`
- The identity file must have the proper checksum format expected by OptimumP2P nodes

**Nodes not connecting to bootstrap:**
- Verify all nodes have unique `CLUSTER_ID` values
- Check that the bootstrap peer ID in `BOOTSTRAP_PEERS` matches the generated identity
- Ensure the network topology allows proper communication between nodes

**Proxy connection issues:**
- Verify all P2P nodes are healthy and running
- Check that the proxy can reach all P2P node sidecar ports (33212)
- Ensure the `P2P_NODES` environment variable contains correct node addresses

## API Reference

### Proxy API

Proxies provide the user-facing interface to OptimumP2P.

#### Subscribe to Topic

```sh
curl -X POST http://localhost:8081/api/v1/subscribe \
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

#### WebSocket Connection

```sh
wscat -c "ws://localhost:8081/api/v1/ws?client_id=your-client-id"
```

This is how clients receive messages published to their subscribed topics.

> **Important:** WebSocket has limitations, and you may experience unreliable delivery when publishing message bursts. A gRPC connection will be provided in a future update.

#### Publish Message

```sh
curl -X POST http://localhost:8081/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "your-client-id",
    "topic": "example-topic",
    "message": "Hello, world!"
  }'
```

> **Important:** The `client_id` field is required for all publish requests. This should be the same ID used when subscribing to topics. If you're using WebSocket connections, use the same `client_id` for consistency.

### Proxy gRPC Streaming

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

#### Example

```sh
sh ./script/proxy_client.sh subscribe mytopic 0.7
sh ./script/proxy_client.sh publish mytopic 0.6 10
```

### Proxy REST API

All proxy endpoints support JWT authentication via `Authorization: Bearer <token>` header.

#### Health Check

```sh
curl http://localhost:8081/api/v1/health
```

**Response:**
```json
{
  "status": "ok",
  "memory_used": "8.71",
  "cpu_used": "0.71", 
  "disk_used": "51.10"
}
```

#### Version Information

```sh
curl http://localhost:8081/api/v1/version
```

**Response:**
```json
{
  "version": "v0.0.1-rc3",
  "commit_hash": "245207d"
}
```

#### Subscribe to Topic

```sh
curl -X POST http://localhost:8081/api/v1/subscribe \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "your-client-id", 
    "topic": "example-topic",
    "threshold": 0.7
  }'
```

**Response:**
```json
{
  "status": "subscribed",
  "client": "your-client-id"
}
```

**Parameters:**
- `client_id`: Client identifier (optional with JWT auth - extracted from claims)
- `topic`: Topic name to subscribe to
- `threshold`: Delivery threshold (0.1-1.0, default: 0.1)

#### Publish Message

```sh
curl -X POST http://localhost:8081/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "your-client-id",
    "topic": "example-topic", 
    "message": "Hello, world!"
  }'
```

**Response:**
```json
{
  "status": "published",
  "topic": "example-topic",
  "message_id": "630bae9baff93fd17a4e71611b2ca7da950860c4f90bcd420f7528571615e7df"
}
```

#### WebSocket Connection

```sh
wscat -c "ws://localhost:8081/api/v1/ws?client_id=your-client-id"
```

**Authentication:** Pass JWT in `Authorization` header during connection.

### P2P Node API

#### Node Health

```sh
curl http://localhost:9091/api/v1/health
```

**Response:**
```json
{
  "cpu_used": "0.29",
  "disk_used": "84.05", 
  "memory_used": "6.70",
  "mode": "optimum",
  "status": "ok"
}
```

#### Node State

```sh
curl http://localhost:9091/api/v1/node-state
```

**Response:**
```json
{
  "pub_key": "12D3KooWMwzQYKhRvLRsiKignZ2vAtkr1nPYiDWA7yeZvRqC9ST9",
  "peers": [
    "12D3KooWDLm7bSFnoqP4mhoJminiCixbR2Lwyqu9sU5EDKVvXM5j",
    "12D3KooWJrPmTdXj9hirigHs88BHe6DApLpdXiKrwF1V8tNq9KP7"
  ],
  "addresses": ["/ip4/172.28.0.12/tcp/7070"],
  "topics": ["demo", "example-topic"]
}
```

### gRPC API

The gRPC API provides high-performance streaming capabilities.

#### Service Definition

```protobuf
service ProxyStream {
  rpc ClientStream (stream ProxyMessage) returns (stream ProxyMessage);
  rpc Publish (PublishRequest) returns (PublishResponse);
  rpc Subscribe (SubscribeRequest) returns (SubscribeResponse);
}
```

#### Authentication

Add JWT token to gRPC metadata:

```go
ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer YOUR_JWT_TOKEN")
```

---

## Client Tools

### gRPC Proxy Client Implementation

> **Note:** The provided client code in `grpc_proxy_client/proxy_client.go` is a SAMPLE implementation intended for demonstration and testing purposes only. It is **not production-ready** and should not be used as-is in production environments. Please review, adapt, and harden the code according to your security, reliability, and operational requirements before any production use.

A new Go-based gRPC client implementation is available in `grpc_proxy_client/proxy_client.go` that provides:

#### Features

* **Bidirectional gRPC Streaming**: Establishes persistent connection with the proxy
* **REST API Integration**: Uses REST for subscription and publishing
* **Automatic Client ID Generation**: Generates unique client identifiers
* **Optimized gRPC Connection**: Efficient bidirectional streaming
* **Message Publishing Loop**: Automated message publishing with configurable delays
* **Signal Handling**: Graceful shutdown on interrupt

#### Usage

```sh
# Build the client
cd grpc_proxy_client
go build -o proxy_client proxy_client.go

# Subscribe only (receive messages)
./proxy_client -subscribeOnly -topic=test -threshold=0.7

# Subscribe and publish messages
./proxy_client -topic=test -threshold=0.7 -count=10 -delay=2s

# Custom connection settings
./proxy_client -topic=test -threshold=0.7 -count=10
```

#### Command Line Flags

* `-topic`: Topic name to subscribe/publish (default: "demo")
* `-threshold`: Delivery threshold 0.0 to 1.0 (default: 0.1)
* `-subscribeOnly`: Only subscribe and receive messages
* `-count`: Number of messages to publish (default: 5)
* `-delay`: Delay between message publishing (default: 2s)
* `-proxy`: Proxy server address (default: "localhost:33211")
* `-rest`: REST API base URL (default: "http://localhost:8081")

#### Protocol Flow

1. **Subscription**: Client subscribes to topic via REST API
2. **gRPC Connection**: Establishes bidirectional stream with proxy
3. **Client ID Registration**: Sends client_id as first message
4. **Message Reception**: Receives messages on subscribed topics
5. **Message Publishing**: Publishes messages via REST API (optional)

#### Generated Protobuf Files

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

### Using P2P Nodes Directly (Optional – No Proxy)

If you prefer to interact directly with the P2P mesh, bypassing the proxy, you can use a minimal client script to publish and subscribe directly over the gRPC sidecar interface of the nodes.

This is useful for:

* Low-level integration
* Bypassing HTTP/WebSocket stack
* Simulating internal services or embedded clients

#### Example (sample implementation)

##### Subscribe to a Topic

```sh
./grpc_p2p_client/p2p-client -mode=subscribe -topic=mytopic --addr=localhost:33221
```

> **Note:** Here, `localhost:33221` is the mapped port for `p2pnode-1` (33221:33212) in docker-compose.

response

```sh
Subscribed to topic "mytopic", waiting for messages…
Received message: "random"
Received message: "random1"
Received message: "random2"
```

##### Publish to a Topic

```sh
./grpc_p2p_client/p2p-client -mode=publish -topic=mytopic --addr=localhost:33222
```

> **Note:** Here, `localhost:33222` is the mapped port for `p2pnode-2` (33222:33212) in docker-compose.

response

```sh
Published "random" to "mytopic"
```

* --addr refers to the sidecar gRPC port exposed by the P2P node (e.g., 33221, 33222, etc.)
* Messages published here will still follow RLNC encoding, mesh forwarding, and threshold policies
* Proxy(s) will pick these up only if enough nodes receive the shards (threshold logic)

### Publishing Options

The P2P client supports various publishing options for testing:

#### Basic Publishing

```sh
# Publish a single message
./grpc_p2p_client/p2p-client -mode=publish -topic=my-topic -msg="Hello World" --addr=127.0.0.1:33221

# Publish multiple messages with delay
./grpc_p2p_client/p2p-client -mode=publish -topic=my-topic --addr=127.0.0.1:33221 -count=10 -sleep=1s
```

#### Available Flags

* `-count`: Number of messages to publish (default: 1)
* `-sleep`: Delay between publishes (e.g., 100ms, 1s)

### Inspecting P2P Nodes

#### Get Node Health

```sh
curl http://localhost:9091/api/v1/health
```

response:

```json
{"cpu_used":"0.29","disk_used":"84.05","memory_used":"6.70","mode":"optimum","status":"ok"}
```

#### Get Node State

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

#### Get Node Version

```sh
curl http://localhost:9091/api/v1/version
```

response:

```sh
{"commit_hash":"6d3d086","version":""}
```

### Collecting Trace Data for Experiments

The gRPC P2P client includes built-in trace collection functionality to help you monitor and analyze message delivery performance during experiments. This is particularly useful for hackathon participants and developers who want to understand how OptimumP2P handles message routing and delivery.

#### How Trace Collection Works

When you subscribe to a topic using the P2P client, you'll automatically receive trace events that show:

- **GossipSub traces**: Traditional pubsub delivery events
- **OptimumP2P traces**: RLNC-enhanced shard delivery events

These traces contain valuable metrics like delivery latency, bandwidth usage, and shard redundancy data.

#### Usage Example

```bash
# Subscribe to a topic and collect trace data
./grpc_p2p_client/p2p-client -mode=subscribe -topic=your-topic --addr=127.0.0.1:33221
```

You'll see trace logs in real-time like:
```
Subscribed to topic "your-topic", waiting for messages…
[TRACE] GossipSub trace received: [binary trace data]
[TRACE] OptimumP2P trace received: [binary trace data]
[1] Received message: "Hello World"
```

**Note:** The trace data appears as protobuf binary format (marshaled TraceEvent structures) for performance optimization. This contains delivery latency, bandwidth usage, and shard redundancy metrics in an efficient binary format, aligned with the optimum-proxy implementation.

#### Understanding Trace Data

**GossipSub Traces**: Show traditional message delivery metrics
- Contains delivery latency and bandwidth metrics
- Binary format for performance optimization

**OptimumP2P Traces**: Show RLNC-enhanced delivery events  
- Contains shard redundancy and RLNC performance data
- Binary format for efficient transmission

#### Experimental Use Cases

This trace collection is perfect for:

1. **Performance Benchmarking**: Compare delivery latency between GossipSub and OptimumP2P
2. **Network Analysis**: Understand shard distribution and redundancy levels
3. **Threshold Optimization**: Analyze how different threshold settings affect delivery
4. **Bandwidth Studies**: Monitor data usage patterns across different protocols

#### Reference Implementation

The trace logging is implemented in `grpc_p2p_client/p2p_client.go`:

```go
case protobuf.ResponseType_MessageTraceGossipSub:
    fmt.Printf("[TRACE] GossipSub trace received: %s\n", string(resp.GetData()))
case protobuf.ResponseType_MessageTraceOptimumP2P:
    fmt.Printf("[TRACE] OptimumP2P trace received: %s\n", string(resp.GetData()))
```

#### Advanced Trace Data Parsing

For developers who want to parse the trace data attributes, you can define structs to handle the binary data:

```go
// Example struct for parsing trace data (when available in JSON format)
type TraceData struct {
    Event       string    `json:"event"`
    Timestamp   time.Time `json:"timestamp"`
    LatencyMs   int       `json:"latency_ms,omitempty"`
    BandwidthBytes int    `json:"bandwidth_bytes,omitempty"`
    ShardID     string    `json:"shard_id,omitempty"`
    Redundancy  float64   `json:"redundancy,omitempty"`
}

// Usage example (when trace data is in JSON format)
case protobuf.ResponseType_MessageTraceOptimumP2P:
    var traceData TraceData
    if err := json.Unmarshal(resp.GetData(), &traceData); err != nil {
        log.Printf("Error parsing trace data: %v", err)
    } else {
        fmt.Printf("[TRACE] OptimumP2P %s: latency=%dms, redundancy=%.2f\n", 
            traceData.Event, traceData.LatencyMs, traceData.Redundancy)
    }
```

This provides both simple logging and structured parsing options for trace data analysis.



---

## Advanced Configuration

### Authentication Setup (Optional)

For development, authentication is disabled by default. Enable Auth0 JWT authentication by setting environment variables:

```yaml
# docker-compose.yml
environment:
  ENABLE_AUTH: "false"
```

### Rate Limiting

Configure per-client rate limits via JWT claims:

```json
{
  "max_publish_per_hour": 1000,
  "max_publish_per_sec": 8, 
  "max_message_size": 4194304,
  "daily_quota": 5368709120
}
```

### P2P Node Configuration

Key environment variables for P2P nodes:

```yaml
environment:
  NODE_MODE: "optimum"              # Protocol mode
  OPTIMUM_SHARD_FACTOR: "4"         # Shards per message  
  OPTIMUM_THRESHOLD: "0.75"         # Shard threshold (75%)
  OPTIMUM_MESH_TARGET: "6"          # Target peer connections
  OPTIMUM_MAX_MSG_SIZE: "1048576"   # Max message size (1MB)
```

### Proxy Configuration

Key environment variables for proxies:

```yaml
environment:
  P2P_NODES: "p2pnode-1:33212,p2pnode-2:33212,p2pnode-3:33212,p2pnode-4:33212"
  SUBSCRIBER_THRESHOLD: "0.1"       # Default threshold
  LOG_LEVEL: "info"                 # Logging level
```

---

## Monitoring and Telemetry

### Prometheus Metrics

Access metrics at `/metrics` endpoint:

```sh
curl http://localhost:8081/metrics
```

#### Key Metrics

**Publish Metrics:**
- `published_messages_by_client_total`: Messages published per client/topic
- `published_message_size_bytes`: Message size histogram  
- `publish_error_total`: Publish errors by type

**Connection Metrics:**
- `total_p2pnodes_connections`: Active P2P connections
- `active_ws_clients`: WebSocket client count

**Delivery Metrics:**
- `message_fallback_deliveries_total`: Messages delivered below threshold
- `node_received_messages_total`: Messages per P2P node

#### Example Queries

Monitor publish rate:
```promql
rate(published_messages_by_client_total[5m])
```

Track message sizes:
```promql
histogram_quantile(0.95, rate(published_message_size_bytes_bucket[5m]))
```

### Trace Collection

The P2P client includes built-in trace collection for performance analysis:

```sh
./grpc_p2p_client/p2p-client -mode=subscribe -topic=your-topic --addr=127.0.0.1:33221
```

**Output includes:**
```text
[TRACE] GossipSub trace received: [binary data]
[TRACE] OptimumP2P trace received: [binary data]
[1] Received message: "Hello World"
```

---

## Troubleshooting

### Common Issues

#### Services Not Starting

**Problem:** `docker-compose up` fails with identity errors

**Solution:**
```sh
# Regenerate identity
rm -rf identity/
./script/generate-identity.sh

# Set environment variable
export BOOTSTRAP_PEER_ID=<generated-peer-id>
docker-compose up --build -d
```

#### API Endpoints Not Responding

**Problem:** `/api/v1/subscribe` returns "Cannot POST"

**Solution:**
```sh
# Check if services are using latest images
docker-compose down
docker system prune -f
docker-compose up --build -d
```

#### P2P Nodes Not Connecting

**Problem:** Nodes show empty peer lists

**Solution:**
```sh
# Verify bootstrap configuration
curl http://localhost:9091/api/v1/node-state

# Check logs
docker logs optimum-dev-setup-guide-p2pnode-1-1
```

#### Authentication Failures

**Problem:** JWT token rejection

**Solution:**
```sh
# Verify Auth0 configuration
# Check token format and claims
# Ensure Auth0 domain/audience match configuration
```

### Performance Optimization

#### High Message Throughput

- Use gRPC clients instead of REST
- Increase shard factor for better redundancy
- Tune mesh parameters for network size

#### Low Latency Requirements

- Reduce threshold values (0.1-0.3)
- Use direct P2P connections
- Optimize network topology

### Log Analysis

Check service logs for detailed debugging:

```sh
# Proxy logs
docker logs optimum-dev-setup-guide-proxy-1-1

# P2P node logs  
docker logs optimum-dev-setup-guide-p2pnode-1-1

# All services
docker-compose logs -f
```

---

## Production Considerations

### Security

- ✅ Enable JWT authentication in production
- ✅ Use proper Auth0 configuration with rate limits
- ✅ Implement proper firewall rules
- ✅ Use TLS for all external communications

### Scalability

- ✅ Proxies are stateless and horizontally scalable
- ✅ Add more P2P nodes for increased mesh resilience
- ✅ Configure load balancing for proxy endpoints
- ✅ Monitor metrics and adjust thresholds based on network conditions

### Monitoring

- ✅ Set up Prometheus monitoring for all metrics
- ✅ Configure alerting for service health
- ✅ Track message delivery rates and latencies
- ✅ Monitor P2P mesh connectivity

---

## Developer Tools

### CLI Integration

For production-ready CLI interaction, see:
- [mump2p-cli](https://github.com/getoptimum/mump2p-cli) - Full-featured CLI client
- Supports JWT authentication, rate limiting, and advanced features

### API Clients

Example client implementations available in:
- `grpc_proxy_client/` - Go gRPC client
- `grpc_p2p_client/` - Go P2P direct client  
- `scripts/` - Shell script wrappers

---

*This development setup provides a complete OptimumP2P environment for testing, integration, and development. For production deployment, review the security and scalability considerations section.*
