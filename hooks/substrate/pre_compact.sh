#!/bin/bash
# Subtrate PreCompact hook - save identity and send status mail
#
# This hook runs before context compaction. It:
# 1. Saves the agent's identity and consumer offsets
# 2. Records a heartbeat
# 3. Sends a status update mail to User (summarizing session)
# 4. Outputs status for context injection
#
# Output format: plain text for context injection

# Read hook input from stdin to get session_id.
input=$(cat)
session_id=$(echo "$input" | jq -r '.session_id // empty')

# Build session args if available.
session_args=""
if [ -n "$session_id" ]; then
    session_args="--session-id $session_id"
fi

# Save identity state before compaction.
substrate identity save $session_args 2>/dev/null || true

# Record heartbeat.
substrate heartbeat $session_args --format context 2>/dev/null || true

# Get agent name for the status mail.
# Format is "Current agent: AgentName (ID: N)"
agent_name=$(substrate identity current $session_args --format text 2>/dev/null | sed -n 's/Current agent: \([^ ]*\).*/\1/p' || echo "Unknown")

# Get project info.
project_dir="${CLAUDE_PROJECT_DIR:-$(pwd)}"
project_name=$(basename "$project_dir")
git_branch=$(git -C "$project_dir" branch --show-current 2>/dev/null || echo "unknown")

# Send status update mail to User (fire and forget, don't block compaction).
# NOTE: claude -p is DISABLED - it causes recursive hook loops.
{
    # Try to get summary from sessions-index.json instead.
    project_hash=$(echo "$project_dir" | tr '/.' '-')
    sessions_index="$HOME/.claude/projects/$project_hash/sessions-index.json"
    summary=""

    if [ -f "$sessions_index" ]; then
        summary=$(jq -r '.entries[-1].summary // empty' "$sessions_index" 2>/dev/null)
    fi

    # Fallback: generic status message.
    if [ -z "$summary" ]; then
        summary="Session compacting. Agent will resume shortly."
    fi

    # Build the status message
    status_body="[Context: Working on $project_name, branch: $git_branch]

$summary

---
(Automated status before context compaction)"

    # Send to User agent
    substrate send $session_args \
        --to User \
        --subject "[Status] $agent_name - Compacting" \
        --body "$status_body" \
        2>/dev/null || true
} &

# Output status summary for context after compaction.
substrate status $session_args --format context 2>/dev/null || true
