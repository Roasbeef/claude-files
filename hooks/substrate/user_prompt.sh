#!/bin/bash
# Subtrate UserPromptSubmit hook - silent heartbeat + check mail
#
# This hook runs each time the user submits a prompt. It:
# 1. Sends a heartbeat to indicate active use
# 2. Quietly checks for new mail to inject as context
#
# Output format: plain text for context injection (quiet if no messages)

# Read hook input from stdin to get session_id.
input=$(cat)
session_id=$(echo "$input" | jq -r '.session_id // empty')

# Try CLAUDE_SESSION_ID env var as fallback.
if [ -z "$session_id" ]; then
    session_id="$CLAUDE_SESSION_ID"
fi

# Send heartbeat (best effort, silent).
if [ -n "$session_id" ]; then
    substrate heartbeat --session-id "$session_id" --format context 2>/dev/null || true
else
    substrate heartbeat --format context 2>/dev/null || true
fi

# Check for new mail and inject as context if any.
if [ -n "$session_id" ]; then
    substrate poll --session-id "$session_id" --quiet --format context 2>/dev/null || true
else
    substrate poll --quiet --format context 2>/dev/null || true
fi
