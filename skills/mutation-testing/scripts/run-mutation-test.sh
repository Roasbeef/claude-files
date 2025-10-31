#!/bin/bash
# Wrapper script for run_mutation.go
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec go run "$SCRIPT_DIR/run_mutation.go" "$@"
