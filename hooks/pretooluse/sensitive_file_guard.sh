#!/bin/bash
# PreToolUse hook to protect sensitive files
# Blocks writes to credential files, warns about others

set -euo pipefail

input=$(cat)

tool=$(echo "$input" | jq -r '.tool // empty')
file_path=$(echo "$input" | jq -r '.parameters.file_path // empty')

# Only run for Write/Edit tools
if [[ "$tool" != "Edit" && "$tool" != "Write" ]]; then
    exit 0
fi

# Check for sensitive file patterns
basename_file=$(basename "$file_path")

# BLOCK these files (exit 1 to block)
case "$basename_file" in
    .env|.env.*|credentials.json|*.pem|*.key|id_rsa|id_rsa.pub|*.p12|*.pfx)
        echo "🚫 BLOCKED: Cannot modify sensitive file: $file_path"
        echo "   This file contains credentials or secrets."
        echo "   If you need to modify it, disable this hook temporarily."
        exit 1
        ;;
esac

# WARN about these files (but don't block)
case "$basename_file" in
    config.json|settings.json|*.conf)
        echo "⚠️  Warning: Modifying configuration file: $file_path"
        echo "   Please review changes carefully."
        ;;
esac

exit 0
