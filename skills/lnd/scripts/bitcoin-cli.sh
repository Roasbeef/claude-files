#!/bin/bash
# Wrapper for bitcoin-cli commands
#
# Usage:
#   bitcoin-cli.sh getblockchaininfo
#   bitcoin-cli.sh getbalance
#   bitcoin-cli.sh sendtoaddress <address> <amount>

set -e

CONTAINER="${BITCOIND_CONTAINER:-lnd-bitcoind}"
NETWORK="regtest"

# Check if container exists
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER}$"; then
    # Try eclair's bitcoind for shared mode
    if docker ps --format '{{.Names}}' | grep -q '^bitcoind$'; then
        CONTAINER="bitcoind"
    else
        echo "Error: bitcoind container not found"
        exit 1
    fi
fi

docker exec "$CONTAINER" bitcoin-cli \
    -regtest \
    -rpcuser=devuser \
    -rpcpassword=devpass \
    "$@"
