#!/usr/bin/env bash
set -e

GATEWAY_CLIENT_DIR="./grpc_gateway_client"

cd "$GATEWAY_CLIENT_DIR"

go build -o gateway-client ./gateway_client.go

if [ -z "${1:-}" ]; then
  echo "Usage: $0 (subscribe <topic> <threshold> [keepalive-options])|(publish <topic> <threshold> <message-count> [keepalive-options])" >&2
  echo "" >&2
  echo "Keepalive options:" >&2
  echo "  -keepalive-interval=<duration>  gRPC keepalive ping interval (default: 2m0s)" >&2
  echo "  -keepalive-timeout=<duration>   gRPC keepalive ping timeout (default: 20s)" >&2
  echo "" >&2
  echo "Examples:" >&2
  echo "  $0 subscribe demo 0.6" >&2
  echo "  $0 subscribe demo 0.7 -keepalive-interval=5m" >&2
  echo "  $0 publish demo 0.5 10 -keepalive-timeout=10s" >&2
  exit 1
fi

MODE="$1"
shift

case "$MODE" in
  subscribe)
    if [ -z "${1:-}" ] || [ -z "${2:-}" ]; then
      echo "Usage: $0 subscribe <topic> <threshold> [keepalive-options]" >&2
      exit 1
    fi
    TOPIC="$1"
    THRESHOLD="$2"
    shift 2
    ./gateway-client -topic="$TOPIC" -threshold="$THRESHOLD" -subscribeOnly "$@"
    ;;
  publish)
    if [ -z "${1:-}" ] || [ -z "${2:-}" ] || [ -z "${3:-}" ]; then
      echo "Usage: $0 publish <topic> <threshold> <message-count> [keepalive-options]" >&2
      exit 1
    fi
    TOPIC="$1"
    THRESHOLD="$2"
    COUNT="$3"
    shift 3
    ./gateway-client -topic="$TOPIC" -threshold="$THRESHOLD" -count="$COUNT" "$@"
    ;;
  *)
    echo "Invalid mode: $MODE" >&2
    echo "Usage: $0 (subscribe <topic> <threshold> [keepalive-options])|(publish <topic> <threshold> <message-count> [keepalive-options])" >&2
    exit 1
    ;;
esac
