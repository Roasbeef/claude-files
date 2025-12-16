#!/bin/bash
# PostToolUse hook to show git status after file modifications
# Helps track what's been changed during the session

set -euo pipefail

input=$(cat)

tool=$(echo "$input" | jq -r '.tool // empty')

# Only run for file modification tools
if [[ "$tool" != "Edit" && "$tool" != "Write" && "$tool" != "MultiEdit" ]]; then
    exit 0
fi

# Check if we're in a git repo
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    exit 0
fi

# Get counts of modified files
modified_count=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')

if [ "$modified_count" -gt 0 ]; then
    echo ""
    echo "📊 Git Status: $modified_count file(s) modified"

    # Show summary (but not too verbose)
    if [ "$modified_count" -le 5 ]; then
        git status --short 2>/dev/null
    else
        echo "   Run 'git status' to see all changes"
    fi
fi

exit 0
