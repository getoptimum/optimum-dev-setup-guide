#!/usr/bin/env bash
set -e

P2P_CLIENT_DIR="./p2p_client"

cd "$P2P_CLIENT_DIR"

go build -o p2p-client ../p2p_client/p2p_client.go

if [ -z "${1:-}" ]; then
  echo "Usage: $0 <addr> (subscribe <topic>)|(publish <topic> <message>)" >&2
  exit 1
fi

ADDR="$1"

case "${2:-}" in
  subscribe)
    if [ -z "${3:-}" ]; then
      echo "Usage: $0 <addr> subscribe <topic>" >&2
      exit 1
    fi
    ./p2p-client -mode=subscribe -topic="$3" --addr="$ADDR"
    ;;
  publish)
    if [ -z "${3:-}" ] || [ -z "${4:-}" ]; then
      echo "Usage: $0 <addr> publish <topic> <message>" >&2
      exit 1
    fi
    ./p2p-client -mode=publish -topic="$3" -msg="$4" --addr="$ADDR"
    ;;
  *)
    echo "Usage: $0 <addr> (subscribe <topic>)|(publish <topic> <message>)" >&2
    exit 1
    ;;
esac