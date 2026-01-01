#!/bin/bash
# Build lnd Docker image from source
#
# Usage:
#   docker-build.sh [--source /path/to/lnd] [--tag tagname]
#   docker-build.sh --branch <branch>       # Build from a specific branch
#   docker-build.sh --pr <number>           # Build from a GitHub PR
#   docker-build.sh --commit <sha>          # Build from a specific commit
#
# Options:
#   --source PATH    Path to lnd source directory (default: attempts to find)
#   --tag NAME       Docker image tag (default: lnd:local)
#   --branch NAME    Checkout and build from a specific branch
#   --commit SHA     Checkout and build from a specific commit
#   --pr NUMBER      Checkout and build from a GitHub PR (uses gh CLI)
#   --remote NAME    Git remote to fetch from (default: origin)
#   --dev            Use dev.Dockerfile (default for local builds)
#   --prod           Use production Dockerfile
#   --tags "list"    Build tags to use (default: signrpc walletrpc chainrpc invoicesrpc peersrpc kvdb_sqlite)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LND_SOURCE=""
IMAGE_TAG="lnd:local"
BRANCH=""
COMMIT=""
PR_NUMBER=""
REMOTE="origin"
USE_DEV=true
BUILD_TAGS=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --source)
            LND_SOURCE="$2"
            shift 2
            ;;
        --tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        --branch)
            BRANCH="$2"
            shift 2
            ;;
        --commit)
            COMMIT="$2"
            shift 2
            ;;
        --pr)
            PR_NUMBER="$2"
            shift 2
            ;;
        --remote)
            REMOTE="$2"
            shift 2
            ;;
        --dev)
            USE_DEV=true
            shift
            ;;
        --prod)
            USE_DEV=false
            shift
            ;;
        --tags)
            BUILD_TAGS="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: docker-build.sh [options]"
            echo ""
            echo "Build lnd Docker image from source with optional branch/commit/PR checkout."
            echo ""
            echo "Options:"
            echo "  --source PATH    Path to lnd source directory"
            echo "  --tag NAME       Docker image tag (default: lnd:local)"
            echo "  --dev            Use dev.Dockerfile (default)"
            echo "  --prod           Use production Dockerfile"
            echo "  --tags \"list\"    Custom build tags"
            echo ""
            echo "Git checkout options (mutually exclusive):"
            echo "  --branch NAME    Checkout and build from a specific branch"
            echo "  --commit SHA     Checkout and build from a specific commit"
            echo "  --pr NUMBER      Checkout and build from a GitHub PR (requires gh CLI)"
            echo "  --remote NAME    Git remote to fetch from (default: origin)"
            echo ""
            echo "Examples:"
            echo "  # Build from current source state"
            echo "  docker-build.sh"
            echo ""
            echo "  # Build from a specific branch"
            echo "  docker-build.sh --branch simple-taproot-chans"
            echo ""
            echo "  # Build from a GitHub PR"
            echo "  docker-build.sh --pr 1234"
            echo ""
            echo "  # Build with custom tags"
            echo "  docker-build.sh --tags \"signrpc walletrpc routerrpc\""
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Try to find lnd source
if [ -z "$LND_SOURCE" ]; then
    # Check common locations
    if [ -d "$GOPATH/src/github.com/lightningnetwork/lnd" ]; then
        LND_SOURCE="$GOPATH/src/github.com/lightningnetwork/lnd"
    elif [ -d "$HOME/gocode/src/github.com/lightningnetwork/lnd" ]; then
        LND_SOURCE="$HOME/gocode/src/github.com/lightningnetwork/lnd"
    elif [ -d "/users/roasbeef/gocode/src/github.com/lightningnetwork/lnd" ]; then
        LND_SOURCE="/users/roasbeef/gocode/src/github.com/lightningnetwork/lnd"
    elif [ -d "$HOME/src/lnd" ]; then
        LND_SOURCE="$HOME/src/lnd"
    else
        echo "Error: Could not find lnd source directory."
        echo "Please specify with --source /path/to/lnd"
        exit 1
    fi
fi

# Determine which Dockerfile to use
if [ "$USE_DEV" = true ]; then
    DOCKERFILE="dev.Dockerfile"
else
    DOCKERFILE="Dockerfile"
fi

if [ ! -f "$LND_SOURCE/$DOCKERFILE" ]; then
    echo "Error: $DOCKERFILE not found in $LND_SOURCE"
    exit 1
fi

cd "$LND_SOURCE"

# Check for mutually exclusive git options
GIT_OPTIONS=0
[ -n "$BRANCH" ] && GIT_OPTIONS=$((GIT_OPTIONS + 1))
[ -n "$COMMIT" ] && GIT_OPTIONS=$((GIT_OPTIONS + 1))
[ -n "$PR_NUMBER" ] && GIT_OPTIONS=$((GIT_OPTIONS + 1))

if [ $GIT_OPTIONS -gt 1 ]; then
    echo "Error: --branch, --commit, and --pr are mutually exclusive"
    exit 1
fi

# Handle git checkout if requested
if [ $GIT_OPTIONS -gt 0 ]; then
    # Check for uncommitted changes
    if ! git diff --quiet || ! git diff --cached --quiet; then
        echo "Error: Uncommitted changes in $LND_SOURCE"
        echo "Please commit or stash your changes before switching branches."
        git status --short
        exit 1
    fi

    if [ -n "$PR_NUMBER" ]; then
        # Check if gh CLI is available
        if ! command -v gh &> /dev/null; then
            echo "Error: gh CLI is required for --pr option"
            echo "Install with: brew install gh"
            exit 1
        fi
        echo "Checking out PR #$PR_NUMBER..."
        gh pr checkout "$PR_NUMBER" --repo lightningnetwork/lnd
    elif [ -n "$BRANCH" ]; then
        echo "Fetching from $REMOTE and checking out branch: $BRANCH"
        git fetch "$REMOTE"
        # Try local branch first, then remote
        if git show-ref --verify --quiet "refs/heads/$BRANCH"; then
            git checkout "$BRANCH"
            git pull "$REMOTE" "$BRANCH" 2>/dev/null || true
        elif git show-ref --verify --quiet "refs/remotes/$REMOTE/$BRANCH"; then
            git checkout -B "$BRANCH" "$REMOTE/$BRANCH"
        else
            echo "Error: Branch '$BRANCH' not found locally or on $REMOTE"
            exit 1
        fi
    elif [ -n "$COMMIT" ]; then
        echo "Checking out commit: $COMMIT"
        git fetch "$REMOTE"
        git checkout "$COMMIT"
    fi
fi

# Show current git state
echo ""
echo "Building lnd Docker image"
echo "  Source:     $LND_SOURCE"
echo "  Dockerfile: $DOCKERFILE"
echo "  Branch:     $(git branch --show-current 2>/dev/null || echo 'detached HEAD')"
echo "  Commit:     $(git rev-parse --short HEAD) - $(git log -1 --format='%s')"
echo "  Tag:        $IMAGE_TAG"
if [ -n "$BUILD_TAGS" ]; then
    echo "  Build Tags: $BUILD_TAGS"
fi
echo ""

# Build the image
if [ -n "$BUILD_TAGS" ]; then
    # Build with custom tags (requires modifying build args)
    docker build \
        -f "$DOCKERFILE" \
        --build-arg tags="$BUILD_TAGS" \
        -t "$IMAGE_TAG" \
        .
else
    docker build -f "$DOCKERFILE" -t "$IMAGE_TAG" .
fi

echo ""
echo "Build complete! Image available as: $IMAGE_TAG"
echo ""
echo "To use this image with docker-compose:"
echo "  LND_IMAGE=$IMAGE_TAG docker-compose up -d"
