#!/bin/bash
# Subtrate SubagentStop hook - One-shot Pattern
#
# This hook is for subagents (Task tool spawned agents). Unlike main agents,
# subagents should complete their work and exit, not stay alive indefinitely.
#
# Behavior:
# - First stop: Check for messages, block if any exist
# - After processing: Allow exit (check stop_hook_active flag)
#
# Output format: JSON for SubagentStop hook decision

# Read hook input from stdin to check stop_hook_active and session_id.
input=$(cat)
stop_hook_active=$(echo "$input" | jq -r '.stop_hook_active // false')
session_id=$(echo "$input" | jq -r '.session_id // empty')

# Build session ID args if available.
session_args=""
if [ -n "$session_id" ]; then
    session_args="--session-id $session_id"
fi

# If we already blocked once and Claude processed messages, allow exit.
# stop_hook_active=true means Claude is trying to stop after our previous block.
if [ "$stop_hook_active" = "true" ]; then
    echo '{"decision": null}'
    exit 0
fi

# First stop - quick check for messages (no long-polling).
# Block only if there are pending messages.
substrate poll $session_args --format hook 2>/dev/null || echo '{"decision": null}'
