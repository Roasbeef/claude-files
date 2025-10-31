#!/bin/bash
# Wrapper script for generate_mutations.go
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec go run "$SCRIPT_DIR/generate_mutations.go" "$@"
