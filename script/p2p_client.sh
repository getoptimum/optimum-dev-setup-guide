#!/usr/bin/env bash
set -e

P2P_CLIENT_DIR="./p2p_client"

cd "$P2P_CLIENT_DIR"

go build -o p2p-client ../p2p_client/p2p_client.go

if [ -z "${1:-}" ]; then
  echo "Usage: $0 <addr> (subscribe <topic> [keepalive-options])|(publish <topic> <message> [keepalive-options])" >&2
  echo "" >&2
  echo "Keepalive options:" >&2
  echo "  -keepalive-time=<duration>    gRPC keepalive ping interval (default: 2m0s)" >&2
  echo "  -keepalive-timeout=<duration> gRPC keepalive ping timeout (default: 20s)" >&2
  echo "" >&2
  echo "Examples:" >&2
  echo "  $0 127.0.0.1:33221 subscribe test" >&2
  echo "  $0 127.0.0.1:33221 subscribe test -keepalive-time=5m" >&2
  echo "  $0 127.0.0.1:33221 publish test \"Hello World\" -keepalive-timeout=10s" >&2
  exit 1
fi

ADDR="$1"
shift

case "${1:-}" in
  subscribe)
    if [ -z "${2:-}" ]; then
      echo "Usage: $0 <addr> subscribe <topic> [keepalive-options]" >&2
      exit 1
    fi
    TOPIC="$2"
    shift 2
    # Pass remaining arguments as keepalive options
    ./p2p-client -mode=subscribe -topic="$TOPIC" --addr="$ADDR" "$@"
    ;;
  publish)
    if [ -z "${2:-}" ] || [ -z "${3:-}" ]; then
      echo "Usage: $0 <addr> publish <topic> <message> [keepalive-options]" >&2
      exit 1
    fi
    TOPIC="$2"
    MESSAGE="$3"
    shift 3
    # Pass remaining arguments as keepalive options
    ./p2p-client -mode=publish -topic="$TOPIC" -msg="$MESSAGE" --addr="$ADDR" "$@"
    ;;
  *)
    echo "Usage: $0 <addr> (subscribe <topic> [keepalive-options])|(publish <topic> <message> [keepalive-options])" >&2
    exit 1
    ;;
esac