#!/usr/bin/env bash
set -e

P2P_CLIENT_DIR="./grpc_p2p_client"

cd "$P2P_CLIENT_DIR"

go build -o p2p-client ./p2p_client.go

if [ -z "${1:-}" ]; then
  echo "Usage: $0 <addr> (subscribe <topic> [flow-control-options])|(publish <topic> <message> [count] [flow-control-options])" >&2
  echo "" >&2
  echo "Flow control options:" >&2
  echo "  -buffer-size=<size>           Message processing buffer size (default: 1000)" >&2
  echo "  -workers=<num>                Number of concurrent workers (default: 4)" >&2
  echo "  -keepalive-interval=<duration> gRPC keepalive ping interval (default: 2m0s)" >&2
  echo "  -keepalive-timeout=<duration> gRPC keepalive ping timeout (default: 20s)" >&2
  echo "" >&2
  echo "Examples:" >&2
  echo "  $0 127.0.0.1:33221 subscribe test" >&2
  echo "  $0 127.0.0.1:33221 subscribe test -buffer-size=2000 -workers=8" >&2
  echo "  $0 127.0.0.1:33221 publish test \"Hello World\" 100" >&2
  echo "  $0 127.0.0.1:33221 publish test \"Hello World\" 1000 -buffer-size=5000" >&2
  exit 1
fi

ADDR="$1"
shift

case "${1:-}" in
  subscribe)
    if [ -z "${2:-}" ]; then
      echo "Usage: $0 <addr> subscribe <topic> [flow-control-options]" >&2
      exit 1
    fi
    TOPIC="$2"
    shift 2
    # Pass remaining arguments as flow control options
    ./p2p-client -mode=subscribe -topic="$TOPIC" --addr="$ADDR" "$@"
    ;;
  publish)
    if [ -z "${2:-}" ] || [ -z "${3:-}" ]; then
      echo "Usage: $0 <addr> publish <topic> <message> [count] [flow-control-options]" >&2
      exit 1
    fi
    TOPIC="$2"
    MESSAGE="$3"
    COUNT="${4:-1}"  # Default to 1 if not specified
    shift 4
    # Pass remaining arguments as flow control options
    ./p2p-client -mode=publish -topic="$TOPIC" -msg="$MESSAGE" -count="$COUNT" --addr="$ADDR" "$@"
    ;;
  *)
    echo "Usage: $0 <addr> (subscribe <topic> [flow-control-options])|(publish <topic> <message> [count] [flow-control-options])" >&2
    exit 1
    ;;
esac