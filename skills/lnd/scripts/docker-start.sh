#!/bin/bash
# Start lnd containers
#
# Usage:
#   docker-start.sh                    # Start bitcoind + lnd stack
#   docker-start.sh --shared           # Start lnd only (use external bitcoind)
#   docker-start.sh --btcd             # Start btcd + lnd stack
#   docker-start.sh --neutrino         # Start lnd in neutrino mode
#   docker-start.sh --multi            # Start multi-node (alice + bob)
#   docker-start.sh --build            # Rebuild before starting

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_DIR="$SCRIPT_DIR/../templates"
MODE="standalone"
BUILD=false
DETACH=true
EXTRA_ARGS=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --shared)
            MODE="shared"
            shift
            ;;
        --btcd)
            MODE="btcd"
            shift
            ;;
        --neutrino)
            MODE="neutrino"
            shift
            ;;
        --multi)
            MODE="multi"
            shift
            ;;
        --build)
            BUILD=true
            shift
            ;;
        --foreground|-f)
            DETACH=false
            shift
            ;;
        -h|--help)
            echo "Usage: docker-start.sh [options]"
            echo ""
            echo "Modes:"
            echo "  (default)         Start bitcoind + lnd stack"
            echo "  --shared          Start lnd only (uses external bitcoind)"
            echo "  --btcd            Start btcd + lnd stack"
            echo "  --neutrino        Start lnd in neutrino mode"
            echo "  --multi           Start multi-node (alice + bob)"
            echo ""
            echo "Options:"
            echo "  --build           Rebuild images before starting"
            echo "  --foreground, -f  Run in foreground (show logs)"
            exit 0
            ;;
        *)
            EXTRA_ARGS="$EXTRA_ARGS $1"
            shift
            ;;
    esac
done

cd "$TEMPLATE_DIR"

# Select compose file based on mode
case "$MODE" in
    standalone)
        COMPOSE_FILE="docker-compose.yml"
        ;;
    shared)
        COMPOSE_FILE="docker-compose-shared.yml"
        ;;
    btcd)
        COMPOSE_FILE="docker-compose-btcd.yml"
        ;;
    neutrino)
        COMPOSE_FILE="docker-compose-neutrino.yml"
        ;;
    multi)
        COMPOSE_FILE="docker-compose-multi.yml"
        ;;
esac

echo "Starting lnd in $MODE mode..."

# Build command
CMD="docker-compose -f $COMPOSE_FILE"

if [ "$BUILD" = true ]; then
    CMD="$CMD up --build"
else
    CMD="$CMD up"
fi

if [ "$DETACH" = true ]; then
    CMD="$CMD -d"
fi

CMD="$CMD $EXTRA_ARGS"

echo "Running: $CMD"
eval "$CMD"

if [ "$DETACH" = true ]; then
    echo ""
    echo "lnd started in background. Check status with:"
    echo "  docker logs -f lnd"
    echo ""
    echo "Run commands with:"
    echo "  ~/.claude/skills/lnd/scripts/lncli.sh getinfo"
fi
