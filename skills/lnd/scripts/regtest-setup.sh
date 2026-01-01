#!/bin/bash
# Initialize regtest environment with funded lnd wallet
#
# Usage:
#   regtest-setup.sh           # Full setup
#   regtest-setup.sh --quick   # Just fund wallet (skip mining)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
QUICK=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --quick)
            QUICK=true
            shift
            ;;
        -h|--help)
            echo "Usage: regtest-setup.sh [--quick]"
            echo ""
            echo "Initialize regtest environment with funded lnd wallet."
            echo ""
            echo "Options:"
            echo "  --quick    Skip initial mining (assumes coinbase already mature)"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "=== LND Regtest Setup ==="
echo ""

# Check if lnd container is running
if ! docker ps --format '{{.Names}}' | grep -qE '^lnd(-shared|-btcd)?$'; then
    echo "Error: lnd container not found. Start with:"
    echo "  ~/.claude/skills/lnd/scripts/docker-start.sh"
    exit 1
fi

# Detect which container is running
if docker ps --format '{{.Names}}' | grep -q '^lnd$'; then
    LND_CONTAINER="lnd"
    BITCOIND_CONTAINER="lnd-bitcoind"
elif docker ps --format '{{.Names}}' | grep -q '^lnd-shared$'; then
    LND_CONTAINER="lnd-shared"
    BITCOIND_CONTAINER="bitcoind"
elif docker ps --format '{{.Names}}' | grep -q '^lnd-btcd$'; then
    LND_CONTAINER="lnd-btcd"
    echo "btcd mode detected - setup works differently"
    exit 1
fi

echo "LND container: $LND_CONTAINER"
echo "Bitcoind container: $BITCOIND_CONTAINER"
echo ""

# Wait for lnd to be ready
echo "Waiting for lnd to be ready..."
for i in {1..30}; do
    if docker exec "$LND_CONTAINER" lncli --network=regtest getinfo &>/dev/null; then
        break
    fi
    echo "  Waiting... ($i/30)"
    sleep 2
done

# Check lnd is ready
if ! docker exec "$LND_CONTAINER" lncli --network=regtest getinfo &>/dev/null; then
    echo "Error: lnd not responding after 60 seconds"
    exit 1
fi

echo "lnd is ready!"
echo ""

# Get RPC credentials based on container
if [ "$BITCOIND_CONTAINER" == "bitcoind" ]; then
    RPCUSER="bitcoin"
    RPCPASS="bitcoin"
else
    RPCUSER="devuser"
    RPCPASS="devpass"
fi

if [ "$QUICK" = false ]; then
    # Mine initial blocks for coinbase maturity
    echo "Mining 101 blocks for coinbase maturity..."
    ADDR=$(docker exec "$BITCOIND_CONTAINER" bitcoin-cli -regtest -rpcuser="$RPCUSER" -rpcpassword="$RPCPASS" getnewaddress)
    docker exec "$BITCOIND_CONTAINER" bitcoin-cli -regtest -rpcuser="$RPCUSER" -rpcpassword="$RPCPASS" generatetoaddress 101 "$ADDR" >/dev/null
    echo "Done!"
    echo ""
fi

# Get lnd address
echo "Getting lnd wallet address..."
LND_ADDR=$(docker exec "$LND_CONTAINER" lncli --network=regtest newaddress p2wkh | jq -r '.address')
echo "LND address: $LND_ADDR"
echo ""

# Send funds to lnd
echo "Sending 10 BTC to lnd wallet..."
docker exec "$BITCOIND_CONTAINER" bitcoin-cli -regtest -rpcuser="$RPCUSER" -rpcpassword="$RPCPASS" sendtoaddress "$LND_ADDR" 10
echo "Done!"
echo ""

# Mine block to confirm
echo "Mining 1 block to confirm..."
ADDR=$(docker exec "$BITCOIND_CONTAINER" bitcoin-cli -regtest -rpcuser="$RPCUSER" -rpcpassword="$RPCPASS" getnewaddress)
docker exec "$BITCOIND_CONTAINER" bitcoin-cli -regtest -rpcuser="$RPCUSER" -rpcpassword="$RPCPASS" generatetoaddress 1 "$ADDR" >/dev/null
echo "Done!"
echo ""

# Show wallet balance
echo "=== LND Wallet Status ==="
docker exec "$LND_CONTAINER" lncli --network=regtest walletbalance

echo ""
echo "=== LND Node Info ==="
docker exec "$LND_CONTAINER" lncli --network=regtest getinfo | jq '{identity_pubkey, alias, num_active_channels, num_peers, synced_to_chain, block_height}'

echo ""
echo "Setup complete! LND is funded and ready."
echo ""
echo "Next steps:"
echo "  # Connect to a peer"
echo "  lncli.sh connect <pubkey>@<host>:9735"
echo ""
echo "  # Open a channel"
echo "  lncli.sh openchannel --node_key=<pubkey> --local_amt=1000000"
