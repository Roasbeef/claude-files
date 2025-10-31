#!/bin/bash
# Wrapper script for parse_results.go
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec go run "$SCRIPT_DIR/parse_results.go" "$@"
