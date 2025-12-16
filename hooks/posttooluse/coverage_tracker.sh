#!/bin/bash
# PostToolUse hook to track test coverage after changes
# Only runs if explicitly enabled via CLAUDE_TRACK_COVERAGE=1

set -euo pipefail

# Check if coverage tracking is enabled
if [ "${CLAUDE_TRACK_COVERAGE:-0}" != "1" ]; then
    exit 0
fi

input=$(cat)

tool=$(echo "$input" | jq -r '.tool // empty')
file_path=$(echo "$input" | jq -r '.parameters.file_path // empty')

# Only run for Edit/Write on Go files (not test files)
if [[ "$tool" != "Edit" && "$tool" != "Write" ]]; then
    exit 0
fi

if [[ ! "$file_path" =~ \.go$ ]] || [[ "$file_path" =~ _test\.go$ ]]; then
    exit 0
fi

# Get the package of the modified file
dir=$(dirname "$file_path")

echo "📈 Checking coverage for $dir..."

# Run coverage check with short timeout
if coverage_output=$(go test -cover -timeout 30s "$dir" 2>&1); then
    # Extract coverage percentage
    coverage=$(echo "$coverage_output" | grep -o "coverage: [0-9.]*%" | head -1)
    if [ -n "$coverage" ]; then
        echo "   $coverage"
    fi
else
    echo "   Coverage check timed out or failed"
fi

exit 0
