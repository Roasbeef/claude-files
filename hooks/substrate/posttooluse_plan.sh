#!/bin/bash
# posttooluse_plan.sh - PostToolUse hook for tracking plan file writes.
# This script is called after Write tool executions. It records plan file
# paths to .claude/.substrate-plan-context so the PreToolUse ExitPlanMode
# hook knows which plan files exist.
#
# Claude Code passes hook data via STDIN as JSON with these fields:
#   tool_name     - Name of the tool (Write, Edit, etc.)
#   tool_input    - JSON input parameters passed to the tool
#   session_id    - Current session ID
#   cwd           - Current working directory

set -euo pipefail

# Read all hook data from stdin.
HOOK_INPUT=$(cat)

# Parse fields using jq. Exit silently if jq unavailable.
if ! command -v jq &>/dev/null; then
    exit 0
fi

TOOL=$(echo "$HOOK_INPUT" | jq -r '.tool_name // empty')
SESSION_ID=$(echo "$HOOK_INPUT" | jq -r '.session_id // empty')
CWD=$(echo "$HOOK_INPUT" | jq -r '.cwd // empty')

# Only process Write tool calls.
if [[ "$TOOL" != "Write" ]]; then
    exit 0
fi

# Extract the file path from tool_input.
FILE_PATH=$(echo "$HOOK_INPUT" | jq -r '.tool_input.file_path // empty')

# Only track writes to plan files (under .claude/plans/).
if [[ -z "$FILE_PATH" ]] || [[ "$FILE_PATH" != *"/.claude/plans/"* ]]; then
    exit 0
fi

# Ensure we have a CWD to write the context file.
if [[ -z "$CWD" ]]; then
    exit 0
fi

# Create the context directory if needed.
CONTEXT_DIR="${CWD}/.claude"
mkdir -p "$CONTEXT_DIR" 2>/dev/null || exit 0

# Append a JSON line to the plan context file.
CONTEXT_FILE="${CONTEXT_DIR}/.substrate-plan-context"
TIMESTAMP=$(date +%s)

echo "{\"session_id\":\"${SESSION_ID}\",\"plan_path\":\"${FILE_PATH}\",\"timestamp\":${TIMESTAMP}}" \
    >> "$CONTEXT_FILE" 2>/dev/null || true

# Silent exit — no output.
exit 0
