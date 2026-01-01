#!/bin/bash
# Stop lnd containers
#
# Usage:
#   docker-stop.sh                 # Stop all lnd containers
#   docker-stop.sh --clean         # Stop and remove volumes
#   docker-stop.sh --shared        # Stop shared mode containers
#   docker-stop.sh --btcd          # Stop btcd mode containers
#   docker-stop.sh --neutrino      # Stop neutrino mode containers

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_DIR="$SCRIPT_DIR/../templates"
MODE="standalone"
CLEAN=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --clean|-v)
            CLEAN=true
            shift
            ;;
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
        -h|--help)
            echo "Usage: docker-stop.sh [options]"
            echo ""
            echo "Options:"
            echo "  --clean, -v       Remove volumes (clean state)"
            echo "  --shared          Stop shared mode containers"
            echo "  --btcd            Stop btcd mode containers"
            echo "  --neutrino        Stop neutrino mode containers"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
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
esac

echo "Stopping lnd ($MODE mode)..."

if [ "$CLEAN" = true ]; then
    docker-compose -f "$COMPOSE_FILE" down -v
    echo "Stopped and removed volumes"
else
    docker-compose -f "$COMPOSE_FILE" down
    echo "Stopped (volumes preserved)"
fi
