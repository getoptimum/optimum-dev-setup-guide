# OptimumP2P Development Setup - Complete Guide

## **IMPORTANT: Remote P2P Clusters for Distributed Testing**

> **🚨 CRITICAL:** Use these remote clusters for distributed testing across environments.

### **Connecting to Remote Clusters**

OptimumP2P supports connecting to remote P2P clusters for distributed testing and production use:

```bash
# Connect to remote cluster nodes
./grpc_p2p_client/p2p-client -mode=subscribe -topic=distributed-topic --addr=remote-node-1:33212
./grpc_p2p_client/p2p-client -mode=publish -topic=distributed-topic -msg="Hello World" --addr=remote-node-2:33212
```

**Key Points:**

- Remote nodes use the standard sidecar port `33212`
- Ensure you have the correct `CLUSTER_ID` for the target cluster
- Messages will propagate across the entire distributed mesh network

---

*Complete guide for setting up and using the OptimumP2P development environment.*

## Table of Contents

- [OptimumP2P Development Setup - Complete Guide](#optimump2p-development-setup---complete-guide)
  - [**IMPORTANT: Remote P2P Clusters for Distributed Testing**](#important-remote-p2p-clusters-for-distributed-testing)
    - [**Connecting to Remote Clusters**](#connecting-to-remote-clusters)
  - [Table of Contents](#table-of-contents)
  - [Prerequisites](#prerequisites)
  - [Getting Started](#getting-started)
    - [1. Generate Bootstrap Identity](#1-generate-bootstrap-identity)
    - [2. Configure Environment](#2-configure-environment)
    - [3. Start Services](#3-start-services)
    - [4. Test Everything](#4-test-everything)
  - [Build and Development Commands](#build-and-development-commands)
    - [Makefile Commands](#makefile-commands)
    - [Direct Binary Usage](#direct-binary-usage)
  - [Configuration](#configuration)
    - [Environment Variables (.env)](#environment-variables-env)
    - [P2P Node Variables](#p2p-node-variables)
    - [One-Command Setup (Alternative)](#one-command-setup-alternative)
  - [Two Ways to Connect](#two-ways-to-connect)
  - [Setup and Installation](#setup-and-installation)
    - [1. Bootstrap Identity Generation](#1-bootstrap-identity-generation)
    - [2. Service Startup](#2-service-startup)
    - [3. Verification](#3-verification)
  - [Monitoring](#monitoring)
    - [Geographic Network Analysis](#geographic-network-analysis)
  - [Troubleshooting](#troubleshooting)
  - [API Reference](#api-reference)
    - [Proxy API](#proxy-api)
      - [Subscribe to Topic](#subscribe-to-topic)
      - [WebSocket Connection](#websocket-connection)
      - [Publish Message](#publish-message)
    - [Proxy gRPC Streaming](#proxy-grpc-streaming)
      - [Example](#example)
    - [Proxy REST API](#proxy-rest-api)
      - [Health Check](#health-check)
      - [Version Information](#version-information)
      - [Subscribe to Topic](#subscribe-to-topic-1)
      - [Publish Message](#publish-message-1)
      - [WebSocket Connection](#websocket-connection-1)
      - [Node Countries](#node-countries)
      - [Metrics Endpoint](#metrics-endpoint)
    - [P2P Node API](#p2p-node-api)
      - [Node Health](#node-health)
      - [Node State](#node-state)
      - [Node Config](#node-config)
      - [P2P Snapshot](#p2p-snapshot)
      - [Topics](#topics)
    - [gRPC API](#grpc-api)
      - [Service Definition](#service-definition)
      - [Authentication](#authentication)
  - [Client Tools](#client-tools)
    - [gRPC Proxy Client Implementation](#grpc-proxy-client-implementation)
      - [Features](#features)
      - [Usage](#usage)
      - [Command Line Flags](#command-line-flags)
      - [Protocol Flow](#protocol-flow)
      - [Generated Protobuf Files](#generated-protobuf-files)
    - [Using P2P Nodes Directly (Optional – No Proxy)](#using-p2p-nodes-directly-optional--no-proxy)
      - [Example (sample implementation)](#example-sample-implementation)
        - [Subscribe to a Topic](#subscribe-to-a-topic)
      - [Understanding Message Output Format](#understanding-message-output-format)
        - [Publish to a Topic](#publish-to-a-topic)
    - [Publishing Options](#publishing-options)
      - [Basic Publishing](#basic-publishing)
      - [Bulk Random Message Publishing](#bulk-random-message-publishing)
      - [Available Flags](#available-flags)
    - [Inspecting P2P Nodes](#inspecting-p2p-nodes)
      - [Get Node Health](#get-node-health)
      - [Get Node State](#get-node-state)
      - [Get Node Version](#get-node-version)
      - [Node Countries (via Proxy)](#node-countries-via-proxy)
    - [Collecting Trace Data for Experiments](#collecting-trace-data-for-experiments)
      - [How Trace Collection Works](#how-trace-collection-works)
      - [Usage Example](#usage-example)
      - [OptimumP2P Trace Event Types](#optimump2p-trace-event-types)
      - [Implementation Details](#implementation-details)
  - [Advanced Configuration](#advanced-configuration)
    - [Authentication Setup (Optional)](#authentication-setup-optional)
    - [Rate Limiting](#rate-limiting)
    - [P2P Node Configuration](#p2p-node-configuration)
    - [Proxy Configuration](#proxy-configuration)
  - [Monitoring and Telemetry](#monitoring-and-telemetry)
    - [Prometheus Metrics](#prometheus-metrics)
      - [Key Metrics](#key-metrics)
      - [Example Queries](#example-queries)
    - [Trace Collection](#trace-collection)
  - [Troubleshooting](#troubleshooting-1)
    - [Common Issues](#common-issues)
      - [Services Not Starting](#services-not-starting)
      - [API Endpoints Not Responding](#api-endpoints-not-responding)
      - [P2P Nodes Not Connecting](#p2p-nodes-not-connecting)
      - [Authentication Failures](#authentication-failures)
    - [Performance Optimization](#performance-optimization)
      - [High Message Throughput](#high-message-throughput)
      - [Low Latency Requirements](#low-latency-requirements)
    - [Log Analysis](#log-analysis)
    - [Debugging Message Delivery](#debugging-message-delivery)
- [Proxy logs](#proxy-logs)
- [P2P node logs](#p2p-node-logs)
- [All services](#all-services)
  - [Production Considerations](#production-considerations)
    - [Security](#security)
    - [Scalability](#scalability)
    - [Monitoring](#monitoring-1)
  - [Developer Tools](#developer-tools)
    - [CLI Integration](#cli-integration)
    - [API Clients](#api-clients)


## Prerequisites

Install these tools before starting:

- Docker and Docker Compose
- Node.js and npm (for WebSocket testing)
- Golang (required for key generation script)
- wscat (WebSocket client for testing)

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

### 2. Configure Environment

Create `.env` file with your assigned credentials:

```sh
BOOTSTRAP_PEER_ID=<your-generated-peer-id>
CLUSTER_ID=<your-assigned-cluster-id>
```

> **Note**: Each participant will generate their own unique bootstrap identity and receive their assigned cluster ID. No need to copy from examples - use your specific values.

### 3. Start Services

```sh
docker-compose -f docker-compose-optimum.yml up --build
```

### 4. Test Everything

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

# Subscribe to a topic
make subscribe 127.0.0.1:33221 testtopic

# Publish random messages
make publish 127.0.0.1:33221 testtopic random
make publish 127.0.0.1:33221 testtopic random 10 1s

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
./grpc_p2p_client/p2p-client -mode=publish -topic=testtopic -msg="Random Message" --addr=127.0.0.1:33222 -count=5 -sleep=1s
```

**Example Output:**

```
# Subscribe output:
Connecting to node at: 127.0.0.1:33221…
Trying to subscribe to topic testtopic…
Subscribed to topic "testtopic", waiting for messages…
Recv message: [1] [1757588485854443000 75] [1757588485852133000 50] HelloWorld

# Publish output:
Connecting to node at: 127.0.0.1:33222…
Published "[1757588485852133000 50] HelloWorld" to "testtopic" (took 840.875µs)
```

## Configuration

Default values are provided, but it's important to understand what each variable does.

### Environment Variables (.env)

Copy the example environment file:

```bash
cp .env.example .env
```

**Important:** After copying, you need to replace the `BOOTSTRAP_PEER_ID` in your `.env` file with the peer ID generated by `make generate-identity`.

**Workflow:**

1. Run `make generate-identity` - this creates a unique peer ID
2. Copy the generated peer ID from the output
3. Edit your `.env` file and replace the example `BOOTSTRAP_PEER_ID` with your generated one

The `.env.example` file contains:

```bash
BOOTSTRAP_PEER_ID=12D3KooWD5RtEPmMR9Yb2ku5VuxqK7Yj1Y5Gv8DmffJ6Ei8maU44
CLUSTER_ID=docker-dev-cluster
PROXY_VERSION=v0.0.1-rc13
P2P_NODE_VERSION=v0.0.1-rc13
```

**Variables explained:**

- `BOOTSTRAP_PEER_ID`: P2P node identity for network discovery
- `CLUSTER_ID`: Logical cluster identifier
- `PROXY_VERSION`: Docker image version for proxy services
- `P2P_NODE_VERSION`: Docker image version for P2P node services

The docker-compose files use these version variables:

- `${PROXY_VERSION-latest}` - uses `PROXY_VERSION` if set, otherwise `latest`
- `${P2P_NODE_VERSION-latest}` - uses `P2P_NODE_VERSION` if set, otherwise `latest`

### P2P Node Variables

- `CLUSTER_ID`: Logical ID of the node
- `NODE_MODE`: `optimum` or `gossipsub` mode (should be `optimum`)
- `SIDECAR_PORT`: gRPC bidirectional port to which proxies connect (default: `33212`)
- `API_PORT`: HTTP port exposing internal node APIs (default: `9090`)
- `IDENTITY_DIR`: Directory containing node identity (p2p.key) (needed only for bootstrap node; each node generates its own on start)
- `BOOTSTRAP_PEERS`: Comma-separated list of peer multiaddrs with /p2p/ ID for initial connection
- `OPTIMUM_PORT`: Port used by the OptimumP2P (default: 7070)
- `OPTIMUM_MAX_MSG_SIZE`: Max message size in bytes (default: `1048576` → 1MB)
- `OPTIMUM_MESH_MIN`: Min number of mesh-connected peers (default: `3`)
- `OPTIMUM_MESH_MAX`: Max number of mesh-connected peers (default: `12`)
- `OPTIMUM_MESH_TARGET`: Target number of peers to connect to (default: `6`)
- `OPTIMUM_SHARD_FACTOR`: Number of shards per message (default: 4)
- `OPTIMUM_SHARD_MULT`: Shard size multiplier (default: 1.5)
- `OPTIMUM_THRESHOLD`: Minimum % of shard redundancy before forwarding message (e.g., 0.75 = 75%)

If you want to learn about mesh networking, see how [Eth2 uses gossipsub](https://github.com/LeastAuthority/eth2.0-specs/blob/dev/specs/phase0/p2p-interface.md#the-gossip-domain-gossipsub).

### One-Command Setup (Alternative)

You can also generate the identity with a single command:

```bash
curl -sSL https://raw.githubusercontent.com/getoptimum/optimum-dev-setup-guide/main/script/generate-identity.sh | bash
```

This downloads and runs the same identity generation script, creating the bootstrap peer identity and setting the environment variable.

## Two Ways to Connect

1. **Via Proxy** (recommended): Connect to proxies for managed access with authentication and rate limiting
2. **Direct P2P**: Connect directly to P2P nodes for low-level integration

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
docker-compose -f docker-compose-optimum.yml up --build -d

# Check service status
docker-compose -f docker-compose-optimum.yml ps
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

## Monitoring

The setup includes Grafana dashboards for visualizing P2P node and proxy metrics.

**Access Grafana:**

- URL: `http://localhost:3000`
- Credentials: `admin` / `admin`

**Available Dashboards:**

- **Optimum Proxy Dashboard**: Proxy uptime, cluster health, CPU/memory usage, goroutines
- **OptimumP2P Nodes Dashboard**: P2P node system state, CPU usage, RAM utilization

**Prometheus:**

- URL: `http://localhost:9090`
- Scrapes metrics from all P2P nodes (port 9090) and proxies (port 8080)

### Geographic Network Analysis

Use the geolocation-aware endpoints to understand how your mesh is distributed:

**Proxy view of node geolocation:**

```sh
curl http://localhost:8081/api/v1/node-countries | jq
```

**Health and country info of the node:**

```sh
curl http://localhost:9091/api/v1/health | jq
```

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

### Debugging Message Delivery

Follow this workflow when subscriptions are not receiving messages:

1. **Confirm topic subscription**

   ```sh
   curl "http://localhost:9091/api/v1/topics?topic=<topic>&nodeinfo=true"
   ```

   Ensure the expected peers appear with `nodeinfo=true`.

2. **Inspect mesh propagation**

   ```sh
   curl http://localhost:9091/api/v1/p2p-snapshot | jq
   ```

   Check that the message hash appears under `seen_message_hashes` (and eventually `fully_decoded_message_hashes`) and that `connected_peers` shows the nodes forwarding shards. If the arrays remain empty, publish test traffic (`make publish 127.0.0.1:33221 <topic> random 5`) and re-check.

3. **Validate proxy routing**

   ```sh
   curl http://localhost:8081/api/v1/node-countries | jq
   ```

   Ensure the proxy lists every node you expect it to route through. Missing entries indicate connectivity or configuration issues.

4. **Review client**

   Confirm the subscriber process is still connected (WebSocket or gRPC) and using the same `client_id` that was supplied when calling `/api/v1/subscribe`.

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

- `client_id`: Used to identify the client across WebSocket sessions. Auth0 user_id recommended if JWT is used.
- `threshold`: Forward message to this client only if X% of nodes have received it.

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

- Bidirectional streaming.
- First message must include only client_id.
- All subsequent messages are sent by Proxy on subscribed topics.

#### Example

```sh
sh ./script/proxy_client.sh subscribe mytopic 0.7
sh ./script/proxy_client.sh publish mytopic 0.6 10
```

### Proxy REST API

The Compose file runs two proxy instances on the host at `http://localhost:8081` and `http://localhost:8082`. Examples below use `proxy-1` (`localhost:8081`). If authentication is enabled, add the header `Authorization: Bearer <token>` to every request.

#### Health Check

```sh
curl http://localhost:8081/api/v1/health
```

**Example Response:**

```json
{
  "country": "India",
  "country_iso": "IN",
  "cpu_used": "2.15",
  "disk_used": "75.09",
  "memory_used": "74.56",
  "status": "ok"
}
```

Returns proxy health along with geolocation for its connected P2P nodes.

#### Version Information

```sh
curl http://localhost:8081/api/v1/version
```

**Response:**

```json
{
  "version": "v0.0.1-rc13",
  "commit_hash": "8f3057d"
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

- `client_id`: Required when auth is disabled; with JWT auth the proxy uses the token subject.
- `topic`: Topic name to subscribe to.
- `threshold`: Delivery threshold from `0.1` to `1.0` (default `0.1`).

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

Use the same `client_id` used during subscription so the proxy can correlate publish and receive flows.

#### WebSocket Connection

```sh
wscat -c "ws://localhost:8081/api/v1/ws?client_id=your-client-id" \
  -H "Authorization: Bearer <token-if-auth-enabled>"
```

The WebSocket endpoint streams deliveries for all active subscriptions belonging to the supplied `client_id`.

#### Node Countries

```sh
curl http://localhost:8081/api/v1/node-countries
```

Returns geolocation information (country and ISO code) for every connected P2P node. Use this to verify regional coverage or feed dashboards that show geographic distribution.

**Example Response:**

```json
{
  "count": 4,
  "countries": {
    "p2pnode-1:33212": "India",
    "p2pnode-2:33212": "India",
    "p2pnode-3:33212": "India",
    "p2pnode-4:33212": "India"
  },
  "country_isos": {
    "p2pnode-1:33212": "IN",
    "p2pnode-2:33212": "IN",
    "p2pnode-3:33212": "IN",
    "p2pnode-4:33212": "IN"
  }
}
```

#### Metrics Endpoint

```sh
curl http://localhost:8081/metrics
```

Prometheus-formatted metrics are enabled by default.

### P2P Node API

Each node exposes an HTTP API on the host ports `9091-9094` (mapping to container port `9090`). These endpoints surface health, topology, and protocol diagnostics.

#### Node Health

```sh
curl http://localhost:9091/api/v1/health
```

**Response:**

```json
{
  "country": "India",
  "country_iso": "IN",
  "cpu_used": "2.78",
  "disk_used": "75.09",
  "memory_used": "74.72",
  "mode": "optimum",
  "status": "ok"
}
```

Provides node health, resource utilisation, and geolocation metadata.

#### Node State

```sh
curl http://localhost:9091/api/v1/node-state
```

**Response:**

```json
{
  "pub_key": "12D3KooWMwzQYKhRvLRsiKignZ2vAtkr1nPYiDWA7yeZvRqC9ST9",
  "country": "United States",
  "country_iso": "US",
  "addresses": ["/ip4/172.28.0.12/tcp/7070"],
  "peers": [
    "12D3KooWDLm7bSFnoqP4mhoJminiCixbR2Lwyqu9sU5EDKVvXM5j",
    "12D3KooWJrPmTdXj9hirigHs88BHe6DApLpdXiKrwF1V8tNq9KP7"
  ],
  "topics": ["demo", "example-topic"]
}
```

Useful for confirming peer connectivity, advertised addresses, and current topic assignments.

#### Node Config

```sh
curl http://localhost:9091/api/v1/config
```

Returns the active configuration for the node (cluster ID, protocol mode, mesh parameters, etc.) with sensitive values removed. Use this to verify that environment variables are applied as expected.

#### P2P Snapshot

```sh
curl http://localhost:9091/api/v1/p2p-snapshot
```

Produces a detailed snapshot that includes message hashes (seen vs decoded), shard statistics, peer scores, and per-connection metadata.

**Example:**

```json
{
  "seen_message_hashes": [],
  "fully_decoded_message_hashes": [],
  "undecoded_shard_buffer_sizes": {},
  "total_peer_count": 0,
  "full_message_node_count": 0,
  "partial_message_node_count": 0,
  "connected_peers": []
}
```

Run a publish workflow (for example `make publish 127.0.0.1:33221 testtopic random 5`) and re-query to see hashes move from `seen_message_hashes` to `fully_decoded_message_hashes`, plus per-peer shard stats in `connected_peers`.

#### Topics

**List all topics with peer counts:**

```sh
curl http://localhost:9091/api/v1/topics
```

**Inspect a single topic:**

```sh
curl "http://localhost:9091/api/v1/topics?topic=example-topic"
```

**Include detailed peer information:**

```sh
curl "http://localhost:9091/api/v1/topics?topic=example-topic&nodeinfo=true"
```

The topics endpoint lets you audit subscription state across the mesh. Set `nodeinfo=true` to obtain peer IDs, addresses, and connection status for every subscriber on the requested topic.

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

- **Bidirectional gRPC Streaming**: Establishes persistent connection with the proxy
- **REST API Integration**: Uses REST for subscription and publishing
- **Automatic Client ID Generation**: Generates unique client identifiers
- **Optimized gRPC Connection**: Efficient bidirectional streaming
- **Message Publishing Loop**: Automated message publishing with configurable delays
- **Signal Handling**: Graceful shutdown on interrupt

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

- `-topic`: Topic name to subscribe/publish (default: "demo")
- `-threshold`: Delivery threshold 0.0 to 1.0 (default: 0.1)
- `-subscribeOnly`: Only subscribe and receive messages
- `-count`: Number of messages to publish (default: 5)
- `-delay`: Delay between message publishing (default: 2s)
- `-proxy`: Proxy server address (default: "localhost:33211")
- `-rest`: REST API base URL (default: "<http://localhost:8081>")

#### Protocol Flow

1. **Subscription**: Client subscribes to topic via REST API
2. **gRPC Connection**: Establishes bidirectional stream with proxy
3. **Client ID Registration**: Sends client_id as first message
4. **Message Reception**: Receives messages on subscribed topics
5. **Message Publishing**: Publishes messages via REST API (optional)

#### Generated Protobuf Files

The gRPC clients use auto-generated protobuf files:

**Proxy Client:**
- `grpc_proxy_client/grpc/proxy_stream.pb.go`: Message type definitions
- `grpc_proxy_client/grpc/proxy_stream_grpc.pb.go`: gRPC service definitions

**P2P Client:**
- `grpc_p2p_client/grpc/p2p_stream.pb.go`: Message type definitions
- `grpc_p2p_client/grpc/p2p_stream_grpc.pb.go`: gRPC service definitions

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

- Low-level integration
- Bypassing HTTP/WebSocket stack
- Simulating internal services or embedded clients

#### Example (sample implementation)

##### Subscribe to a Topic

**Local Docker Development:**

```sh
./grpc_p2p_client/p2p-client -mode=subscribe -topic=mytopic --addr=localhost:33221
```

> **Note:** Here, `localhost:33221` is the mapped port for `p2pnode-1` (33221:33212) in docker-compose.

**External/Remote P2P Nodes:**

```sh
./grpc_p2p_client/p2p-client -mode=subscribe -topic=mytopic --addr=34.124.246.10:33212
```

> **Note:** External nodes use the standard sidecar port `33212` directly.

response

```sh
Subscribed to topic "mytopic", waiting for messages…
Received message: "random"
Received message: "random1"
Received message: "random2"
```

#### Understanding Message Output Format

When subscribing to topics, you'll see detailed message information in this format:

```sh
Recv message: [1] [1757579641382484000 126] [1757579641203739000 100] bqhn4Yhab4KorTqcHmViooGF3gPmjSwAZon8kjMUGJY8aRoH/ogmuTZ+IHS/xwa1
meOKYWvJ37ossi5bbMGAg5TgsB0aP61x/Oi
```

**Message Format Breakdown:**

1. **`[1]`** - **Message Sequence Number**
   - Incremental counter of received messages
   - Shows this is the 1st, 2nd, 3rd... message received

2. **`[1757579641382484000 126]`** - **Receive Timestamp & Size**
   - `1757579641382484000` = **Unix timestamp in nanoseconds** when message was received
   - `126` = **Total message size** in bytes (including prefix)

3. **`[1757579641203739000 100]`** - **Original Publish Timestamp & Content Size**
   - `1757579641203739000` = **Unix timestamp in nanoseconds** when message was originally published
   - `100` = **Original content size** in bytes (before prefix was added)

4. **`bqhn4Yhab4KorTqcHmViooGF3gPmjSwAZon8kjMUGJY8aRoH/ogmuTZ+IHS/xwa1...`** - **Message Content**
   - The actual message data (base64 encoded random content in this example)
   - This is the original message content that was published

**Key Insights:**

- **Network Latency**: ~18ms (difference between publish and receive timestamps)
- **Message Integrity**: Content size matches original (100 bytes)
- **Real-time Delivery**: Messages arrive with precise timing information

##### Publish to a Topic

**Local Docker Development:**

```sh
./grpc_p2p_client/p2p-client -mode=publish -topic=mytopic -msg="Hello World" --addr=localhost:33222
```

> **Note:** Here, `localhost:33222` is the mapped port for `p2pnode-2` (33222:33212) in docker-compose.

**External/Remote P2P Nodes:**

```sh
./grpc_p2p_client/p2p-client -mode=publish -topic=mytopic -msg="Hello World" --addr=35.197.161.77:33212
```

> **Note:** External nodes use the standard sidecar port `33212` directly.

response

```sh
Published "[1757588485852133000 26] random" to "mytopic" (took 72.042µs)
```

- --addr refers to the sidecar gRPC port exposed by the P2P node (e.g., 33221, 33222, etc.)
- Messages published here will still follow RLNC encoding, mesh forwarding, and threshold policies
- Proxy(s) will pick these up only if enough nodes receive the shards (threshold logic)

### Publishing Options

The P2P client supports various publishing options for testing:

#### Basic Publishing

```sh
# Publish a single message
./grpc_p2p_client/p2p-client -mode=publish -topic=my-topic -msg="Hello World" --addr=127.0.0.1:33221

# Publish multiple messages with delay
./grpc_p2p_client/p2p-client -mode=publish -topic=my-topic -msg="Random Message" --addr=127.0.0.1:33221 -count=10 -sleep=1s
```

#### Bulk Random Message Publishing

For high-volume testing with random messages:

```sh
# Publish 50 random messages with 2-second delays
for i in `seq 1 50`; do 
  string=$(openssl rand -base64 768 | head -c 100)
  echo "Publishing message $i: $string"
  ./grpc_p2p_client/p2p-client -mode=publish -topic=mytopic --addr=34.40.4.192:33212 -msg="$string"
  sleep 2
done
```

**Features:**

- **Random Content**: Each message contains 100 random characters
- **High Volume**: Publishes 50 messages in sequence
- **Real-time Feedback**: Shows message number and content being published
- **Configurable Delay**: 2-second intervals between messages
- **Remote Testing**: Uses remote P2P node for distributed testing

#### Available Flags

- `-count`: Number of messages to publish (default: 1)
- `-sleep`: Delay between publishes (e.g., 100ms, 1s)

### Inspecting P2P Nodes

#### Get Node Health

```sh
curl http://localhost:9091/api/v1/health
```

response:

```json
{
  "status": "ok",
  "mode": "optimum",
  "cpu_used": "0.29",
  "memory_used": "6.70",
  "disk_used": "84.05",
  "country": "United States",
  "country_iso": "US"
}
```

#### Get Node State

```sh
curl http://localhost:9091/api/v1/node-state
```

response:

```json
{
  "pub_key": "12D3KooWMwzQYKhRvLRsiKignZ2vAtkr1nPYiDWA7yeZvRqC9ST9",
  "country": "United States",
  "country_iso": "US",
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

```json
{
  "version": "v0.0.1-rc13",
  "commit_hash": "8f3057d"
}
```

#### Node Countries (via Proxy)

```sh
curl http://localhost:8081/api/v1/node-countries
```

Returns the same geolocation metadata but aggregated across all nodes that the proxy currently manages.

### Collecting Trace Data for Experiments

The gRPC P2P client includes built-in trace collection functionality that automatically parses and displays trace events from both GossipSub and OptimumP2P protocols. This helps monitor message delivery performance and understand RLNC-enhanced shard behavior.

#### How Trace Collection Works

When you subscribe to a topic, the client automatically receives and parses trace events:

- **GossipSub traces**: Traditional pubsub delivery events with structured JSON output
- **OptimumP2P traces**: RLNC-enhanced shard delivery events with detailed shard information

#### Usage Example

```bash
# Subscribe to a topic and collect trace data
./grpc_p2p_client/p2p-client -mode=subscribe -topic=your-topic --addr=127.0.0.1:33221
```

You'll see structured trace output like:

```
Subscribed to topic "your-topic", waiting for messages…
[TRACE] OptimumP2P type=JOIN ts=2025-09-11T15:58:04.746971127+05:30 size=66B
[TRACE] OptimumP2P JSON (136B): {"type":9,"peerID":"ACQIARIgJUuLFt9bycr0mdXiMdJ1bQ8Nuxs2Y8NtQwPrXEVCuKM=","timestamp":1757586484746971127,"join":{"topic":"trace-test"}}
[TRACE] OptimumP2P type=SEND_RPC ts=2025-09-11T15:58:04.73762546+05:30 size=114B
[TRACE] OptimumP2P JSON (260B): {"type":7,"peerID":"ACQIARIgJUuLFt9bycr0mdXiMdJ1bQ8Nuxs2Y8NtQwPrXEVCuKM=","timestamp":1757586484746035127,"sendRPC":{"sendTo":"ACQIARIg46ViPpa30cOyFCgRdiW+TS/qpMkuXQsKK0w+5svzqk8=","meta":{"subscription":[{"subscribe":true,"topic":"trace-test"}]},"length":16}}
[TRACE] OptimumP2P type=GRAFT ts=2025-09-11T15:58:28.517443638+05:30 size=106B
[TRACE] OptimumP2P JSON (202B): {"type":11,"peerID":"ACQIARIg46ViPpa30cOyFCgRdiW+TS/qpMkuXQsKK0w+5svzqk8=","timestamp":1757586508517443638,"graft":{"peerID":"ACQIARIgJUuLFt9bycr0mdXiMdJ1bQ8Nuxs2Y8NtQwPrXEVCuKM=","topic":"trace-test"}}
[1] Received message: "Hello World"
```

**Note:** Trace events are primarily available when connecting to local Docker P2P nodes. Initial connection generates JOIN, SEND_RPC, and GRAFT events. During message flow, you'll see rich RLNC shard events (NEW_SHARD, RECV_RPC, UNNECESSARY_SHARD) that show the protocol's coding behavior. Remote nodes may not generate trace events.

#### OptimumP2P Trace Event Types

The client recognizes these OptimumP2P trace events (observed in practice):

**Common Events:**

- **JOIN**: Node joins a topic (type=9)
- **SEND_RPC**: Sends RPC messages to peers (type=7)
- **GRAFT**: Establishes mesh connections for topic (type=11)

**Shard Events** (when RLNC is active):

- **NEW_SHARD**: New RLNC shard created with message ID and coefficients (type=16)
- **DUPLICATE_SHARD**: Duplicate shard detected (type=13)
- **UNHELPFUL_SHARD**: Shard that doesn't help decode (type=14)
- **UNNECESSARY_SHARD**: Shard that's not needed for decoding (type=15)

**Other Events:**

- **PUBLISH_MESSAGE**: Message published to topic (type=0)
- **DELIVER_MESSAGE**: Message delivered to subscriber (type=3)
- **ADD_PEER/REMOVE_PEER**: Peer connection events (type=4/5)
- **RECV_RPC**: Receives RPC messages from peers (type=6)
- **LEAVE**: Node leaves a topic (type=10)
- **PRUNE**: Removes mesh connections (type=12)

#### Implementation Details

The trace parsing is implemented in `grpc_p2p_client/p2p_client.go`:

```go
func handleGossipSubTrace(data []byte) {
    evt := &pubsubpb.TraceEvent{}
    if err := proto.Unmarshal(data, evt); err != nil {
        fmt.Printf("[TRACE] GossipSub decode error: %v\n", err)
        return
    }
    ts := time.Unix(0, evt.GetTimestamp()).Format(time.RFC3339Nano)
    fmt.Printf("[TRACE] GossipSub type=%s ts=%s size=%dB\n", evt.GetType().String(), ts, len(data))
    jb, _ := json.Marshal(evt)
    fmt.Printf("[TRACE] GossipSub JSON (%dB): %s\n", len(jb), string(jb))
}

func handleOptimumP2PTrace(data []byte) {
    evt := &optsub.TraceEvent{}
    if err := proto.Unmarshal(data, evt); err != nil {
        fmt.Printf("[TRACE] OptimumP2P decode error: %v\n", err)
        return
    }
    ts := time.Unix(0, evt.GetTimestamp()).Format(time.RFC3339Nano)
    typeStr := optsub.TraceEvent_Type_name[int32(evt.GetType())]
    fmt.Printf("[TRACE] OptimumP2P type=%s ts=%s size=%dB\n", typeStr, ts, len(data))
    
    // Display shard-specific details
    switch evt.GetType() {
    case optsub.TraceEvent_NEW_SHARD:
        fmt.Printf("  NEW_SHARD id=%x coeff=%x\n", evt.GetNewShard().GetMessageID(), evt.GetNewShard().GetCoefficients())
    case optsub.TraceEvent_DUPLICATE_SHARD:
        fmt.Printf("  DUPLICATE_SHARD id=%x\n", evt.GetDuplicateShard().GetMessageID())
    // ... other shard types
    }
    
    jb, _ := json.Marshal(evt)
    fmt.Printf("[TRACE] OptimumP2P JSON (%dB): %s\n", len(jb), string(jb))
}
```

This provides both human-readable summaries and complete JSON data for detailed analysis.

## Advanced Configuration

### Authentication Setup (Optional)

For development, authentication is disabled by default. Enable Auth0 JWT authentication by setting environment variables:

```yaml
# docker-compose-optimum.yml
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
[TRACE] OptimumP2P type=JOIN ts=2025-09-11T15:58:04.746971127+05:30 size=66B
[TRACE] OptimumP2P type=SEND_RPC ts=2025-09-11T15:58:04.73762546+05:30 size=114B
[TRACE] OptimumP2P type=GRAFT ts=2025-09-11T15:58:28.517443638+05:30 size=106B
Recv message: [1] [1757579641382484000 126] [1757579641203739000 100] Hello World
```

**Note:** Trace events appear during initial connection setup (JOIN, SEND_RPC, GRAFT) and continue during message flow with rich RLNC shard events (NEW_SHARD, RECV_RPC, UNNECESSARY_SHARD).

---

## Troubleshooting

### Common Issues

#### Services Not Starting

**Problem:** `docker-compose -f docker-compose-optimum.yml up` fails with identity errors

**Solution:**

```sh
# Regenerate identity
rm -rf identity/
./script/generate-identity.sh

# Set environment variable
export BOOTSTRAP_PEER_ID=<generated-peer-id>
docker-compose -f docker-compose-optimum.yml up --build -d
```

#### API Endpoints Not Responding

**Problem:** `/api/v1/subscribe` returns "Cannot POST"

**Solution:**

```sh
# Check if services are using latest images
docker-compose -f docker-compose-optimum.yml down
docker system prune -f
docker-compose -f docker-compose-optimum.yml up --build -d
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
docker-compose -f docker-compose-optimum.yml logs -f
```---

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


