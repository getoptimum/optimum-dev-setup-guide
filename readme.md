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

## Example: Basic Usage

```sh
# Subscribe to a topic (in one terminal)
sh ./script/p2p_client.sh 127.0.0.1:33221 subscribe my-topic

# Publish messages (in another terminal)
sh ./script/p2p_client.sh 127.0.0.1:33221 publish my-topic "Hello World"
```

---

## Repository Structure

```sh
optimum-dev-setup-guide/
├── docs/                   # Documentation
│   ├── guide.md           # Complete setup guide
│   └── intro.png          # Architecture diagram
├── grpc_p2p_client/       # P2P client implementation
│   ├── grpc/              # Generated gRPC files
│   ├── proto/             # Protocol definitions
│   └── p2p_client.go      # Main P2P client
├── grpc_proxy_client/     # Proxy client implementation
│   ├── grpc/              # Generated gRPC files
│   ├── proto/             # Protocol definitions
│   └── proxy_client.go    # Main proxy client
├── keygen/                # Key generation utilities
│   └── generate_p2p_key.go
├── script/                # Utility scripts
│   ├── generate-identity.sh # Bootstrap identity generation
│   ├── p2p_client.sh       # P2P client wrapper
│   └── proxy_client.sh     # Proxy client wrapper
├── docker-compose.yml     # Service orchestration
├── test_suite.sh          # API validation tests
└── README.md              # This file
```

---

## 📚 Complete Setup Guide

For detailed setup instructions, configuration options, API reference, and troubleshooting, see:

**[Complete Setup Guide](./docs/guide.md)** - Comprehensive documentation covering:
- Prerequisites and detailed setup
- Configuration options for proxies and P2P nodes
- API reference (REST, gRPC, WebSocket)
- Authentication and rate limiting
- Monitoring and telemetry
- Troubleshooting and performance optimization

---

## Quick Start

```sh
# 1. Generate bootstrap identity
./script/generate-identity.sh

# 2. Start all services
docker-compose up --build -d

# 3. Test the setup
./test_suite.sh
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
