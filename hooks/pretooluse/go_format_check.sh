#!/bin/bash
# PreToolUse hook for Go file formatting
# Checks if Go files need formatting before edits

set -euo pipefail

# Read the tool use data from stdin
input=$(cat)

# Extract tool name and file path
tool=$(echo "$input" | jq -r '.tool // empty')
file_path=$(echo "$input" | jq -r '.parameters.file_path // empty')

# Only run for Edit/Write tools on Go files
if [[ "$tool" != "Edit" && "$tool" != "Write" ]]; then
    exit 0
fi

if [[ ! "$file_path" =~ \.go$ ]]; then
    exit 0
fi

# If file exists and is a Go file, check formatting
if [ -f "$file_path" ]; then
    # Check if gofmt would make changes
    if ! gofmt -l "$file_path" | grep -q "^$"; then
        # File needs formatting - provide feedback but don't block
        echo "⚠️  Note: $file_path needs Go formatting (gofmt)"
        echo "   This won't block your edit, but consider running: gofmt -w $file_path"
    fi
fi

exit 0
