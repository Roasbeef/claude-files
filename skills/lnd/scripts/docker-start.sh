#!/bin/bash
# Start lnd containers
#
# Usage:
#   docker-start.sh                              # Start bitcoind + lnd stack
#   docker-start.sh --profile taproot           # Use taproot profile
#   docker-start.sh --args "--accept-keysend"   # Custom args
#   docker-start.sh --shared                    # Use external bitcoind
#   docker-start.sh --multi                     # Multi-node (alice + bob)
#
# Profiles: default, taproot, wumbo, debug, interop, experimental

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_DIR="$SCRIPT_DIR/../templates"
PROFILE_DIR="$SCRIPT_DIR/../profiles"
MODE="standalone"
BUILD=false
DETACH=true
PROFILE=""
CUSTOM_ARGS=""
COMPOSE_EXTRA_ARGS=""

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
        --profile|-p)
            PROFILE="$2"
            shift 2
            ;;
        --args|-a)
            CUSTOM_ARGS="$2"
            shift 2
            ;;
        --list-profiles)
            echo "Available profiles:"
            echo ""
            for f in "$PROFILE_DIR"/*.env; do
                name=$(basename "$f" .env)
                desc=$(head -1 "$f" | sed 's/^# //')
                printf "  %-15s %s\n" "$name" "$desc"
            done
            echo ""
            echo "Usage: docker-start.sh --profile <name>"
            exit 0
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
            echo "Configuration:"
            echo "  --profile, -p     Load profile (taproot, wumbo, debug, interop, experimental)"
            echo "  --args, -a        Custom lnd args (quoted string)"
            echo "  --list-profiles   Show available profiles"
            echo ""
            echo "Options:"
            echo "  --build           Rebuild images before starting"
            echo "  --foreground, -f  Run in foreground (show logs)"
            echo ""
            echo "Examples:"
            echo "  docker-start.sh --profile taproot"
            echo "  docker-start.sh --profile wumbo --build"
            echo "  docker-start.sh --args '--accept-keysend --protocol.wumbo-channels'"
            echo "  docker-start.sh --profile interop --args '--maxpendingchannels=10'"
            exit 0
            ;;
        *)
            COMPOSE_EXTRA_ARGS="$COMPOSE_EXTRA_ARGS $1"
            shift
            ;;
    esac
done

cd "$TEMPLATE_DIR"

# Load profile if specified
if [ -n "$PROFILE" ]; then
    PROFILE_FILE="$PROFILE_DIR/$PROFILE.env"
    if [ -f "$PROFILE_FILE" ]; then
        echo "Loading profile: $PROFILE"
        source "$PROFILE_FILE"
        export LND_DEBUG
        export LND_EXTRA_ARGS
    else
        echo "Error: Profile '$PROFILE' not found"
        echo "Available profiles:"
        ls -1 "$PROFILE_DIR"/*.env 2>/dev/null | xargs -n1 basename | sed 's/.env$//'
        exit 1
    fi
fi

# Append custom args if specified
if [ -n "$CUSTOM_ARGS" ]; then
    if [ -n "$LND_EXTRA_ARGS" ]; then
        export LND_EXTRA_ARGS="$LND_EXTRA_ARGS $CUSTOM_ARGS"
    else
        export LND_EXTRA_ARGS="$CUSTOM_ARGS"
    fi
fi

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
if [ -n "$PROFILE" ]; then
    echo "  Profile: $PROFILE"
fi
if [ -n "$LND_EXTRA_ARGS" ]; then
    echo "  Extra args: $LND_EXTRA_ARGS"
fi
echo ""

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

CMD="$CMD $COMPOSE_EXTRA_ARGS"

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
