#!/bin/bash
# Mine blocks in regtest
#
# Usage:
#   mine.sh        # Mine 1 block
#   mine.sh 6      # Mine 6 blocks (confirm channel)
#   mine.sh 100    # Mine 100 blocks (coinbase maturity)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BLOCKS="${1:-1}"
CONTAINER="${BITCOIND_CONTAINER:-lnd-bitcoind}"

# Check if container exists, fall back to eclair's bitcoind
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER}$"; then
    if docker ps --format '{{.Names}}' | grep -q '^bitcoind$'; then
        CONTAINER="bitcoind"
        RPCUSER="bitcoin"
        RPCPASS="bitcoin"
    else
        echo "Error: bitcoind container not found"
        exit 1
    fi
else
    RPCUSER="devuser"
    RPCPASS="devpass"
fi

# Get new address and mine to it
ADDRESS=$(docker exec "$CONTAINER" bitcoin-cli \
    -regtest \
    -rpcuser="$RPCUSER" \
    -rpcpassword="$RPCPASS" \
    getnewaddress)

echo "Mining $BLOCKS block(s) to $ADDRESS..."

HASHES=$(docker exec "$CONTAINER" bitcoin-cli \
    -regtest \
    -rpcuser="$RPCUSER" \
    -rpcpassword="$RPCPASS" \
    generatetoaddress "$BLOCKS" "$ADDRESS")

echo "Mined $BLOCKS block(s)"
echo "Block hashes: $HASHES"
