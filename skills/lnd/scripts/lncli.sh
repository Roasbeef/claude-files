#!/bin/bash
# Wrapper for lncli commands
#
# Usage:
#   lncli.sh getinfo
#   lncli.sh walletbalance
#   lncli.sh openchannel --node_key=<pubkey> --local_amt=1000000
#
# Options:
#   --node NAME         Node name for multi-node: alice, bob, charlie (default: auto-detect)
#   --container NAME    Docker container name (overrides --node)
#   --network NAME      Network: regtest, testnet, signet, mainnet (default: regtest)
#   --local             Use local lncli instead of docker exec
#   All other args are passed to lncli

set -e

CONTAINER=""
NODE=""
NETWORK="regtest"
USE_LOCAL=false
LNCLI_ARGS=()

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --node)
            NODE="$2"
            shift 2
            ;;
        --container)
            CONTAINER="$2"
            shift 2
            ;;
        --network)
            NETWORK="$2"
            shift 2
            ;;
        --local)
            USE_LOCAL=true
            shift
            ;;
        -h|--help)
            echo "Usage: lncli.sh [options] <command> [args]"
            echo ""
            echo "Options:"
            echo "  --node NAME         Node name: alice, bob, charlie (multi-node mode)"
            echo "  --container NAME    Docker container name (overrides --node)"
            echo "  --network NAME      Network: regtest, testnet, signet, mainnet"
            echo "  --local             Use local lncli"
            echo ""
            echo "Examples:"
            echo "  # Single node mode"
            echo "  lncli.sh getinfo"
            echo "  lncli.sh walletbalance"
            echo ""
            echo "  # Multi-node mode"
            echo "  lncli.sh --node alice getinfo"
            echo "  lncli.sh --node bob walletbalance"
            echo "  lncli.sh --node alice openchannel --node_key=<bob_pubkey> --local_amt=1000000"
            exit 0
            ;;
        *)
            LNCLI_ARGS+=("$1")
            shift
            ;;
    esac
done

if [ ${#LNCLI_ARGS[@]} -eq 0 ]; then
    echo "Error: No lncli command specified"
    echo "Usage: lncli.sh <command> [args]"
    exit 1
fi

# Resolve node name to container
if [ -n "$NODE" ] && [ -z "$CONTAINER" ]; then
    CONTAINER="lnd-$NODE"
fi

# Auto-detect container if not specified
if [ -z "$CONTAINER" ]; then
    # Try single-node containers first
    if docker ps --format '{{.Names}}' | grep -q '^lnd$'; then
        CONTAINER="lnd"
    elif docker ps --format '{{.Names}}' | grep -q '^lnd-shared$'; then
        CONTAINER="lnd-shared"
    elif docker ps --format '{{.Names}}' | grep -q '^lnd-btcd$'; then
        CONTAINER="lnd-btcd"
    elif docker ps --format '{{.Names}}' | grep -q '^lnd-neutrino$'; then
        CONTAINER="lnd-neutrino"
    # Try multi-node containers (default to alice)
    elif docker ps --format '{{.Names}}' | grep -q '^lnd-alice$'; then
        CONTAINER="lnd-alice"
        echo "Note: Using alice node. Specify --node bob or --node charlie for other nodes." >&2
    else
        echo "Error: No lnd container found. Is lnd running?"
        echo "Start with: docker-compose up -d"
        exit 1
    fi
fi

# Verify container exists
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER}$"; then
    echo "Error: Container '$CONTAINER' not found or not running"
    echo "Running lnd containers:"
    docker ps --format '{{.Names}}' | grep -E '^lnd' || echo "  (none)"
    exit 1
fi

if [ "$USE_LOCAL" = true ]; then
    lncli --network="$NETWORK" "${LNCLI_ARGS[@]}"
else
    docker exec "$CONTAINER" lncli --network="$NETWORK" "${LNCLI_ARGS[@]}"
fi
