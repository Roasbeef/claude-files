#!/usr/bin/env bash
# Install gremlins mutation testing tool.
# Idempotent: skips if already installed at the pinned version.
set -euo pipefail

# Pin to a known-good gremlins version. Bump after testing.
PINNED_VERSION="${GREMLINS_VERSION:-v0.5.0}"

if command -v gremlins >/dev/null 2>&1; then
    have_version="$(gremlins version 2>/dev/null | awk '{print $NF}' || true)"
    if [[ "$have_version" == "$PINNED_VERSION" ]]; then
        echo "gremlins $PINNED_VERSION already installed."
        exit 0
    fi
    echo "gremlins present at $have_version; reinstalling $PINNED_VERSION."
fi

if ! command -v go >/dev/null 2>&1; then
    echo "error: go toolchain not found in PATH." >&2
    exit 1
fi

echo "Installing gremlins $PINNED_VERSION via 'go install'..."
GOFLAGS="" go install "github.com/go-gremlins/gremlins/cmd/gremlins@${PINNED_VERSION}"

# Verify GOBIN/GOPATH/bin is on PATH.
GOBIN="$(go env GOBIN)"
if [[ -z "$GOBIN" ]]; then
    GOBIN="$(go env GOPATH)/bin"
fi

if ! command -v gremlins >/dev/null 2>&1; then
    echo "error: gremlins installed to $GOBIN but not on PATH. Add it to PATH and retry." >&2
    exit 1
fi

echo "Installed: $(gremlins version)"
