#!/bin/bash
# task_sync.sh - PostToolUse hook for syncing Claude Code task operations to Substrate.
# This script is called after TaskCreate, TaskUpdate, TaskList, or TaskGet tools execute.
#
# Claude Code passes hook data via STDIN as JSON with these fields:
#   tool_name     - Name of the tool (TaskCreate, TaskUpdate, etc.)
#   tool_input    - JSON input parameters passed to the tool
#   tool_response - JSON output/result from the tool execution
#   session_id    - Current session ID (also the task list ID)
#   cwd           - Current working directory

set -euo pipefail

# Debug log file location.
DEBUG_LOG="${HOME}/.subtrate/task_sync_debug.log"

# Debug logging enabled by default during development.
DEBUG="${TASK_SYNC_DEBUG:-1}"

# Log a debug message with timestamp.
debug_log() {
    if [[ "$DEBUG" == "1" ]]; then
        mkdir -p "$(dirname "$DEBUG_LOG")" 2>/dev/null || true
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >> "$DEBUG_LOG" 2>/dev/null || true
    fi
}

debug_log "=== task_sync.sh invoked ==="

# Read all hook data from stdin (Claude Code passes JSON via stdin).
HOOK_INPUT=$(cat)

debug_log "HOOK_INPUT=${HOOK_INPUT:-(empty)}"

# Parse fields from stdin JSON using jq.
if ! command -v jq &>/dev/null; then
    debug_log "jq not found, cannot parse hook input."
    exit 0
fi

TOOL=$(echo "$HOOK_INPUT" | jq -r '.tool_name // empty')
TOOL_INPUT_JSON=$(echo "$HOOK_INPUT" | jq -c '.tool_input // empty')
TOOL_RESPONSE_JSON=$(echo "$HOOK_INPUT" | jq -c '.tool_response // empty')

# The session_id IS the task list ID (Claude Code uses session ID as list ID).
LIST_ID=$(echo "$HOOK_INPUT" | jq -r '.session_id // empty')

debug_log "tool_name=$TOOL"
debug_log "tool_input=$TOOL_INPUT_JSON"
debug_log "tool_response=$TOOL_RESPONSE_JSON"
debug_log "list_id=$LIST_ID"

if [[ -z "$TOOL" ]]; then
    debug_log "No tool_name in hook input, exiting."
    exit 0
fi

if [[ -z "$LIST_ID" ]]; then
    debug_log "No session_id/list_id in hook input, exiting."
    exit 0
fi

# Run sync in background to avoid blocking the agent.
(
    # Determine substrate command location.
    SUBSTRATE="${SUBSTRATE_CLI:-substrate}"

    # Check if substrate CLI is available.
    if ! command -v "$SUBSTRATE" &>/dev/null; then
        # Try common locations.
        if [[ -x "$HOME/go/bin/substrate" ]]; then
            SUBSTRATE="$HOME/go/bin/substrate"
        elif [[ -x "$HOME/gocode/bin/substrate" ]]; then
            SUBSTRATE="$HOME/gocode/bin/substrate"
        elif [[ -x "/usr/local/bin/substrate" ]]; then
            SUBSTRATE="/usr/local/bin/substrate"
        else
            debug_log "substrate CLI not found, exiting."
            exit 0
        fi
    fi

    debug_log "Using substrate at: $(command -v "$SUBSTRATE" 2>/dev/null || echo "$SUBSTRATE")"

    # Route based on tool name, piping the tool response/input to hook-sync.
    # Pass --session-id so the CLI can resolve/create agent identity.
    case "$TOOL" in
        TaskCreate)
            # Merge tool_input (full fields: subject, description, activeForm)
            # with tool_response.task (has assigned ID) to get complete data.
            MERGED=$(echo "$HOOK_INPUT" | jq -c '{task: (.tool_input + (.tool_response.task // .tool_response // {}))}')
            debug_log "Syncing TaskCreate merged=$MERGED"
            echo "$MERGED" | "$SUBSTRATE" tasks hook-sync \
                --tool create --list "$LIST_ID" \
                --session-id "$LIST_ID" 2>>"$DEBUG_LOG" || {
                debug_log "hook-sync create failed (exit $?)"
                true
            }
            ;;
        TaskUpdate)
            # tool_input contains the update parameters (taskId, status, etc.).
            debug_log "Syncing TaskUpdate input"
            echo "$TOOL_INPUT_JSON" | "$SUBSTRATE" tasks hook-sync \
                --tool update --list "$LIST_ID" \
                --session-id "$LIST_ID" 2>>"$DEBUG_LOG" || {
                debug_log "hook-sync update failed (exit $?)"
                true
            }
            ;;
        TaskList)
            # tool_response contains the full list of tasks.
            debug_log "Syncing TaskList response"
            echo "$TOOL_RESPONSE_JSON" | "$SUBSTRATE" tasks hook-sync \
                --tool list --list "$LIST_ID" \
                --session-id "$LIST_ID" 2>>"$DEBUG_LOG" || {
                debug_log "hook-sync list failed (exit $?)"
                true
            }
            ;;
        TaskGet)
            # tool_response contains the retrieved task.
            debug_log "Syncing TaskGet response"
            echo "$TOOL_RESPONSE_JSON" | "$SUBSTRATE" tasks hook-sync \
                --tool get --list "$LIST_ID" \
                --session-id "$LIST_ID" 2>>"$DEBUG_LOG" || {
                debug_log "hook-sync get failed (exit $?)"
                true
            }
            ;;
        *)
            debug_log "Unknown tool: $TOOL, ignoring."
            ;;
    esac

    debug_log "=== task_sync.sh complete ==="
) &

# Exit immediately, sync runs in background.
exit 0
