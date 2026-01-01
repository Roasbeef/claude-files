#!/bin/bash
# Initialize multi-node regtest environment
#
# Sets up alice and bob nodes with funded wallets and an open channel between them.
#
# Usage:
#   multi-node-setup.sh              # Full setup with channel
#   multi-node-setup.sh --no-channel # Setup without opening channel

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OPEN_CHANNEL=true

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-channel)
            OPEN_CHANNEL=false
            shift
            ;;
        -h|--help)
            echo "Usage: multi-node-setup.sh [--no-channel]"
            echo ""
            echo "Initialize multi-node regtest with funded wallets."
            echo ""
            echo "Options:"
            echo "  --no-channel    Skip opening a channel between alice and bob"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "=== LND Multi-Node Setup ==="
echo ""

# Check containers are running
for NODE in alice bob; do
    if ! docker ps --format '{{.Names}}' | grep -q "^lnd-$NODE\$"; then
        echo "Error: lnd-$NODE container not found."
        echo "Start multi-node stack with:"
        echo "  cd ~/.claude/skills/lnd/templates"
        echo "  docker-compose -f docker-compose-multi.yml up -d --build"
        exit 1
    fi
done

BITCOIND_CONTAINER="lnd-multi-bitcoind"
if ! docker ps --format '{{.Names}}' | grep -q "^${BITCOIND_CONTAINER}$"; then
    echo "Error: bitcoind container not found"
    exit 1
fi

echo "Nodes detected: alice, bob"
echo "Bitcoind: $BITCOIND_CONTAINER"
echo ""

# Wait for nodes to be ready
echo "Waiting for nodes to be ready..."
for NODE in alice bob; do
    for i in {1..30}; do
        if docker exec "lnd-$NODE" lncli --network=regtest getinfo &>/dev/null; then
            echo "  $NODE: ready"
            break
        fi
        if [ $i -eq 30 ]; then
            echo "  $NODE: timeout"
            exit 1
        fi
        sleep 2
    done
done
echo ""

# Mine initial blocks
echo "Mining 101 blocks for coinbase maturity..."
ADDR=$(docker exec "$BITCOIND_CONTAINER" bitcoin-cli -regtest -rpcuser=devuser -rpcpassword=devpass getnewaddress)
docker exec "$BITCOIND_CONTAINER" bitcoin-cli -regtest -rpcuser=devuser -rpcpassword=devpass generatetoaddress 101 "$ADDR" >/dev/null
echo "Done!"
echo ""

# Fund both nodes
for NODE in alice bob; do
    echo "Funding $NODE..."
    NODE_ADDR=$(docker exec "lnd-$NODE" lncli --network=regtest newaddress p2wkh | jq -r '.address')
    docker exec "$BITCOIND_CONTAINER" bitcoin-cli -regtest -rpcuser=devuser -rpcpassword=devpass sendtoaddress "$NODE_ADDR" 10
    echo "  Sent 10 BTC to $NODE_ADDR"
done
echo ""

# Confirm transactions
echo "Mining 6 blocks to confirm..."
docker exec "$BITCOIND_CONTAINER" bitcoin-cli -regtest -rpcuser=devuser -rpcpassword=devpass generatetoaddress 6 "$ADDR" >/dev/null
echo "Done!"
echo ""

# Get node pubkeys
ALICE_PUBKEY=$(docker exec lnd-alice lncli --network=regtest getinfo | jq -r '.identity_pubkey')
BOB_PUBKEY=$(docker exec lnd-bob lncli --network=regtest getinfo | jq -r '.identity_pubkey')

echo "=== Node Info ==="
echo "Alice pubkey: $ALICE_PUBKEY"
echo "Bob pubkey:   $BOB_PUBKEY"
echo ""

if [ "$OPEN_CHANNEL" = true ]; then
    # Connect alice to bob
    echo "Connecting alice to bob..."
    docker exec lnd-alice lncli --network=regtest connect "${BOB_PUBKEY}@lnd-bob:9735" 2>/dev/null || true
    echo "Done!"
    echo ""

    # Open channel from alice to bob
    echo "Opening 1M sat channel from alice to bob..."
    docker exec lnd-alice lncli --network=regtest openchannel --node_key="$BOB_PUBKEY" --local_amt=1000000 --push_amt=100000
    echo "Done!"
    echo ""

    # Confirm channel
    echo "Mining 6 blocks to confirm channel..."
    docker exec "$BITCOIND_CONTAINER" bitcoin-cli -regtest -rpcuser=devuser -rpcpassword=devpass generatetoaddress 6 "$ADDR" >/dev/null
    echo "Done!"
    echo ""

    # Show channel status
    echo "=== Channel Status ==="
    echo "Alice channels:"
    docker exec lnd-alice lncli --network=regtest listchannels | jq '.channels[] | {remote_pubkey: .remote_pubkey[:16], capacity, local_balance, remote_balance, active}'
    echo ""
fi

# Show balances
echo "=== Wallet Balances ==="
echo "Alice:"
docker exec lnd-alice lncli --network=regtest walletbalance | jq '{total_balance, confirmed_balance}'
echo ""
echo "Bob:"
docker exec lnd-bob lncli --network=regtest walletbalance | jq '{total_balance, confirmed_balance}'
echo ""

echo "=== Setup Complete! ==="
echo ""
echo "Commands:"
echo "  # Alice"
echo "  lncli.sh --node alice getinfo"
echo "  lncli.sh --node alice channelbalance"
echo ""
echo "  # Bob"
echo "  lncli.sh --node bob getinfo"
echo "  lncli.sh --node bob addinvoice --amt=10000"
echo ""
echo "  # Alice pays Bob"
echo "  INVOICE=\$(lncli.sh --node bob addinvoice --amt=10000 | jq -r '.payment_request')"
echo "  lncli.sh --node alice sendpayment --pay_req=\$INVOICE"
