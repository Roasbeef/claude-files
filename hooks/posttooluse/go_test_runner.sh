#!/bin/bash
# PostToolUse hook to run Go tests after code changes
# Only runs if explicitly enabled via CLAUDE_AUTO_TEST=1

set -euo pipefail

# Check if auto-test is enabled
if [ "${CLAUDE_AUTO_TEST:-0}" != "1" ]; then
    exit 0
fi

input=$(cat)

tool=$(echo "$input" | jq -r '.tool // empty')
file_path=$(echo "$input" | jq -r '.parameters.file_path // empty')

# Only run for Edit/Write on Go files
if [[ "$tool" != "Edit" && "$tool" != "Write" ]]; then
    exit 0
fi

if [[ ! "$file_path" =~ \.go$ ]]; then
    exit 0
fi

# Skip test files themselves
if [[ "$file_path" =~ _test\.go$ ]]; then
    exit 0
fi

# Get the directory of the modified file
dir=$(dirname "$file_path")

echo "🧪 Running tests for $dir..."

# Run tests with short timeout
if go test -timeout 30s "$dir" 2>&1; then
    echo "✅ Tests passed for $dir"
else
    echo "❌ Tests failed for $dir"
    echo "   Review the output above and fix any failing tests."
fi

exit 0
