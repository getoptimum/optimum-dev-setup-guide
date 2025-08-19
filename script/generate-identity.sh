#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO] $1${NC}"
}

print_success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
}

print_error() {
    echo -e "${RED}[ERROR] $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}[WARNING] $1${NC}"
}

# Main script
print_status "Generating P2P Bootstrap Identity..."

IDENTITY_DIR="./identity"
KEY_FILE="$IDENTITY_DIR/p2p.key"

# Check if identity already exists
if [ -f "$KEY_FILE" ]; then
    print_warning "P2P identity already exists at $KEY_FILE"
    read -p "Delete existing identity and generate new one? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$IDENTITY_DIR"
        print_status "Deleted existing identity"
    else
        print_status "Using existing identity"
        # Extract and export existing Peer ID
        PEER_ID=$(cat "$KEY_FILE" | grep -o '"ID":"[^"]*"' | cut -d'"' -f4)
        export BOOTSTRAP_PEER_ID="$PEER_ID"
        print_success "BOOTSTRAP_PEER_ID=$BOOTSTRAP_PEER_ID"
        echo "export BOOTSTRAP_PEER_ID=$BOOTSTRAP_PEER_ID"
        exit 0
    fi
fi

# Create identity directory
print_status "Creating identity directory..."
mkdir -p "$IDENTITY_DIR"

# Create temporary Go program
print_status "Creating key generator..."
cat > ./temp_generate_key.go << 'EOF'
package main

import (
    "crypto/rand"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "github.com/libp2p/go-libp2p/core/crypto"
    "github.com/libp2p/go-libp2p/core/peer"
)

type IdentityInfo struct {
    Key []byte  `json:"Key"`
    ID  peer.ID `json:"ID"`
}

func main() {
    // Generate Ed25519 keypair
    pk, _, err := crypto.GenerateEd25519Key(rand.Reader)
    if err != nil {
        fmt.Printf("Failed to generate key: %v\n", err)
        os.Exit(1)
    }

    // Get peer ID from private key
    id, err := peer.IDFromPrivateKey(pk)
    if err != nil {
        fmt.Printf("Failed to derive peer ID: %v\n", err)
        os.Exit(1)
    }

    // Marshal private key to bytes
    raw, err := crypto.MarshalPrivateKey(pk)
    if err != nil {
        fmt.Printf("Failed to marshal key: %v\n", err)
        os.Exit(1)
    }

    // Save to identity/p2p.key
    info := IdentityInfo{Key: raw, ID: id}
    data, err := json.Marshal(info)
    if err != nil {
        fmt.Printf("Failed to marshal identity: %v\n", err)
        os.Exit(1)
    }

    keyPath := filepath.Join("identity", "p2p.key")
    if err := os.WriteFile(keyPath, data, 0600); err != nil {
        fmt.Printf("Failed to write key file: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("Peer ID: %s\n", id.String())
}
EOF

# Initialize Go module and generate key
print_status "Initializing Go module..."
go mod init temp-identity-gen 2>/dev/null || true

print_status "Downloading dependencies..."
go get github.com/libp2p/go-libp2p@latest

print_status "Generating P2P keypair..."
PEER_OUTPUT=$(go run temp_generate_key.go)

# Clean up temporary files
rm -f temp_generate_key.go go.mod go.sum

# Extract Peer ID and export it
PEER_ID=$(echo "$PEER_OUTPUT" | grep "Peer ID:" | cut -d' ' -f3)
export BOOTSTRAP_PEER_ID="$PEER_ID"

print_success "Generated P2P identity successfully!"
print_success "Identity saved to: $KEY_FILE"
print_success "Peer ID: $PEER_ID"
echo
print_status "To use in docker-compose:"
echo "export BOOTSTRAP_PEER_ID=$PEER_ID"
echo
print_success "Done! Your OptimumP2P peer ID: $PEER_ID"
