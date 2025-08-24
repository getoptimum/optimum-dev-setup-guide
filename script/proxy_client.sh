#!/usr/bin/env bash
set -e

PROXY_CLIENT_DIR="./grpc_proxy_client"

cd "$PROXY_CLIENT_DIR"

go build -o proxy-client ./proxy_client.go

if [ -z "${1:-}" ]; then
  echo "Usage: $0 (subscribe <topic> <threshold>)|(publish <topic> <threshold> <message-count>)" >&2
  echo "" >&2
  echo "Examples:" >&2
  echo "  $0 subscribe demo 0.6" >&2
  echo "  $0 subscribe demo 0.7" >&2
  echo "  $0 publish demo 0.5 10" >&2
  exit 1
fi

MODE="$1"
shift

case "$MODE" in
  subscribe)
    if [ -z "${1:-}" ] || [ -z "${2:-}" ]; then
      echo "Usage: $0 subscribe <topic> <threshold>" >&2
      exit 1
    fi
    TOPIC="$1"
    THRESHOLD="$2"
    shift 2
    ./proxy-client -topic="$TOPIC" -threshold="$THRESHOLD" -subscribeOnly "$@"
    ;;
  publish)
    if [ -z "${1:-}" ] || [ -z "${2:-}" ] || [ -z "${3:-}" ]; then
      echo "Usage: $0 publish <topic> <threshold> <message-count>" >&2
      exit 1
    fi
    TOPIC="$1"
    THRESHOLD="$2"
    COUNT="$3"
    shift 3
    ./proxy-client -topic="$TOPIC" -threshold="$THRESHOLD" -count="$COUNT" "$@"
    ;;
  *)
    echo "Invalid mode: $MODE" >&2
    echo "Usage: $0 (subscribe <topic> <threshold>)|(publish <topic> <threshold> <message-count>)" >&2
    exit 1
    ;;
esac
