#!/bin/bash
set -e

KEYGEN_DIR="./keygen"
IDENTITY_DIR="./identity"
KEY_FILE="$IDENTITY_DIR/p2p.key"

if [ -f "$KEY_FILE" ]; then
  echo "p2p.key already exists at $KEY_FILE — deleting identity folder"
  rm -rf "$IDENTITY_DIR"
fi

echo "🔧 Generating new p2p.key"

mkdir -p "$IDENTITY_DIR"
cd "$KEYGEN_DIR"

# Initialize Go module only once
if [ ! -f "go.mod" ]; then
  echo "Initializing Go module"
  go mod init p2p-keygen
  go get github.com/libp2p/go-libp2p@latest
fi

go run generate_p2p_key.go
