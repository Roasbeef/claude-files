#!/bin/bash
# Wrapper script for run_all_mutations.go
# Runs all mutations sequentially (safe for in-place file modification)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec go run "$SCRIPT_DIR/run_all_mutations.go" "$@"
