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
    # In CI environment, always use existing identity
    if [ "$CI" = "true" ] || [ "$GITHUB_ACTIONS" = "true" ]; then
        print_status "CI environment detected - using existing identity"
        # Extract and export existing Peer ID
        PEER_ID=$(cat "$KEY_FILE" | grep -o '"ID":"[^"]*"' | cut -d'"' -f4)
        export BOOTSTRAP_PEER_ID="$PEER_ID"
        print_success "BOOTSTRAP_PEER_ID=$BOOTSTRAP_PEER_ID"
        echo "export BOOTSTRAP_PEER_ID=$BOOTSTRAP_PEER_ID"
        exit 0
    else
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
fi

# Create identity directory
print_status "Creating identity directory..."
mkdir -p "$IDENTITY_DIR"

# Use the existing keygen script
print_status "Using existing keygen script..."
cd keygen

print_status "Generating P2P keypair..."
PEER_OUTPUT=$(go run generate_p2p_key.go)

cd ..

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
