#!/bin/bash
# Wrapper script for apply_mutation.go
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec go run "$SCRIPT_DIR/apply_mutation.go" "$@"
