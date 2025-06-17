# OptimumP2P – Local Development Setup

This repository provides a full-stack setup for running OptimumP2P, a high-performance RLNC-enhanced pubsub protocol, along with multiple gateways for scalable message routing.
This repository provides a sample Docker Compose setup for deploying the OptimumP2P messaging infrastructure locally.
It demonstrates how partners can configure gateways and P2P nodes, and serves as a reference architecture for integration, testing, and scaling into production.

## Architecture

!['Gateway-P2P Node Communication'](./docs/intro.png)

### How it works

* Clients (like CLI, dApps, or backend services) interact with Gateways using REST/WebSocket.
* Gateways handle user subscriptions, publish requests, and stream delivery.
* Gateways connect to multiple P2P nodes via gRPC sidecar connections.
* P2P nodes form an RLNC-enhanced mesh network, forwarding messages via coded shards.
* Messages propagate based on configurable thresholds and shard redundancy.

**Important:** Gateways are stateless and horizontally scalable. P2P nodes form a resilient gossip + RLNC mesh.

## Purpose

This setup is not production-ready but is designed to:

* Show how to run multiple P2P nodes and gateways
* Demonstrate typical configuration options
* Help partners bootstrap their own network using OptimumP2P

**You are expected to modify this template based on your environment, infrastructure, and security needs.**

## What It Includes

* 4 P2P Nodes running the OptimumP2P
* 2 Gateways for client-facing APIs (HTTP/WebSocket)
* Static IP overlay (optimum-network) for deterministic internal addressing
* .env-based dynamic peer identity setup
* Optional Auth0 support (disabled by default)

## Repository Structure

```
optimum-dev-setup-guide/
├── keygen/                 # Key generation utilities
│   └── generate_p2p_key.go # P2P key generation implementation
├── p2p_client/            # P2P client implementation
│   ├── grpc/              # gRPC implementation
│   │   ├── proto/         # Protocol buffer definitions
│   │   │   └── stream.pb.go    # Generated protobuf code
│   │   ├── stream.pb.go        # Generated protobuf message types
│   │   └── stream_grpc.pb.go   # Generated gRPC service definitions
│   └── p2p_client.go      # Main P2P client implementation
├── script/                # Utility scripts
│   ├── p2p_client.sh     # P2P client setup script
│   └── generate_p2p_key.sh # Key generation script
├── docker-compose.yml     # Docker compose configuration
└── test_suite.sh         # Test suite script
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

##### Gateway Variables

* `CLUSTER_ID`: Gateway instance ID
* `GATEWAY_PORT`: Port on which the gateway serves REST/WebSocket API
* `P2P_NODES`: Comma-separated list of gRPC sidecar endpoints (e.g., `host:port`)
* `ENABLE_AUTH`: If true, JWT Auth0 is required; if false, API is open (local only) (default: false)
* `LOG_LEVEL`: Log verbosity level (e.g., `debug`)

##### P2P Node Variables

* `CLUSTER_ID`: Logical ID of the node
* `NODE_MODE`: `optimum` or `gossipsub` mode (should be `optimum`)
* `SIDECAR_PORT`: gRPC bidirectional port to which gateways connect (default: `33212`)
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
* Simulate client/gateway/node interaction
* Customize clustering, shard behavior, and thresholds

## Ways of Using It

There are two main ways to use this setup:

1. **Gateway:** Connects to all P2P nodes and provides responses based on a threshold (e.g., whether 1-100% of nodes have received the message).
2. **P2P Nodes:** Interact directly with a P2P node.

## Gateway API

Gateways provide the user-facing interface to OptimumP2P.

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

## Using P2P Nodes Directly (Optional – No Gateway)

If you prefer to interact directly with the P2P mesh, bypassing the gateway, you can use a minimal client script to publish and subscribe directly over the gRPC sidecar interface of the nodes.

This is useful for:

* Low-level integration
* Bypassing HTTP/WebSocket stack
* Simulating internal services or embedded clients

### Subscribe to a Topic

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

### Publish to a Topic

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
* Gateway(s) will pick these up only if enough nodes receive the shards (threshold logic)

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

You can use CLI for testing as well that connects to gateway

See CLI guide: [mump2p-cli](https://github.com/getoptimum/mump2p-cli)
