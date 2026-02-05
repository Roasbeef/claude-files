#!/bin/bash
# Subtrate SessionStart hook - heartbeat + check for mail
#
# This hook runs when a Claude Code session starts. It:
# 1. Exports CLAUDE_SESSION_ID to the environment (via CLAUDE_ENV_FILE)
# 2. Sends a heartbeat to mark the agent as active
# 3. Checks for any pending messages to inject as context
#
# Output format: plain text for context injection

# Read hook input from stdin to get session_id.
input=$(cat)
session_id=$(echo "$input" | jq -r '.session_id // empty')

# Export CLAUDE_SESSION_ID via CLAUDE_ENV_FILE if available.
# This makes the session ID available to the agent during the session.
if [ -n "$session_id" ] && [ -n "$CLAUDE_ENV_FILE" ]; then
    echo "CLAUDE_SESSION_ID=$session_id" >> "$CLAUDE_ENV_FILE"
fi

# Send heartbeat to mark session start.
if [ -n "$session_id" ]; then
    substrate heartbeat --session-start --session-id "$session_id" --format context 2>/dev/null || true
else
    substrate heartbeat --session-start --format context 2>/dev/null || true
fi

# Poll for new messages (non-blocking).
# Output is injected as context at session start.
if [ -n "$session_id" ]; then
    result=$(substrate poll --session-id "$session_id" --format context --quiet 2>/dev/null || echo "")
else
    result=$(substrate poll --format context --quiet 2>/dev/null || echo "")
fi

if [ -n "$result" ]; then
    echo "$result"
fi
