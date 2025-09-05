#!/usr/bin/env bash
set -e

P2P_CLIENT_DIR="./grpc_p2p_client"

cd "$P2P_CLIENT_DIR"

go build -o p2p-client ./p2p_client.go

if [ -z "${1:-}" ]; then
  echo "Usage: $0 <addr> (subscribe <topic>)|(publish <topic> <message|\"random\"> [count=N] [sleep=Xs])" >&2
  echo "" >&2
  echo "Options for publish:" >&2
  echo "  -count=<N>                    Number of messages to send (default: 1)" >&2
  echo "  -sleep=<duration>             Delay between messages (e.g., 500ms, 2s)" >&2
  echo "" >&2
  echo "Examples:" >&2
  echo "  $0 127.0.0.1:33221 subscribe test" >&2
  echo "  $0 127.0.0.1:33221 publish test \"Hello World\"" >&2
  echo "  $0 127.0.0.1:33221 publish test \"random\" -count=100 -sleep=200ms" >&2
  exit 1
fi

ADDR="$1"
shift

case "${1:-}" in
  subscribe)
    if [ -z "${2:-}" ]; then
      echo "Usage: $0 <addr> subscribe <topic>" >&2
      exit 1
    fi
    TOPIC="$2"
    shift 2
    ./p2p-client -mode=subscribe -topic="$TOPIC" --addr="$ADDR" "$@"
    ;;
  publish)
    if [ -z "${2:-}" ] || [ -z "${3:-}" ]; then
      echo "Usage: $0 <addr> publish <topic> <message|\"random\"> [count=N] [sleep=Xs]" >&2
      exit 1
    fi
    TOPIC="$2"
    MESSAGE="$3"
    shift 3

    if [[ "$MESSAGE" == "random" ]]; then
      ./p2p-client -mode=publish -topic="$TOPIC" --addr="$ADDR" "$@" # msg is generated internally
    else
      ./p2p-client -mode=publish -topic="$TOPIC" -msg="$MESSAGE" --addr="$ADDR" "$@"
    fi
    ;;
  *)
    echo "Usage: $0 <addr> (subscribe <topic>)|(publish <topic> <message> [options])" >&2
    exit 1
    ;;
esac
