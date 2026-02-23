#!/bin/bash
# Subtrate Stop hook - Mail-First Persistent Agent Pattern
#
# Priority order:
# 1. Quick mail check - if mail exists, block immediately
# 2. Check Tasks - if incomplete tasks exist, block immediately
# 3. Long poll - keep agent alive, continuously checking for work
#
# Key behaviors:
# - Checks mail before tasks (mail is more actionable)
# - Long-polls for 9m30s (under 10m hook timeout)
# - Always outputs {"decision": "block"} to stay alive
# - User can force exit with Ctrl+C (bypasses hooks)
# - No automated status/diff messages (see issue #78)
#
# Output format: JSON for Stop hook decision

# Read hook input from stdin
input=$(cat)
session_id=$(echo "$input" | jq -r '.session_id // empty')
stop_hook_active=$(echo "$input" | jq -r '.stop_hook_active // false')

# Build session ID args if available (critical for agent identity resolution).
session_args=""
if [ -n "$session_id" ]; then
    session_args="--session-id $session_id"
fi

# ============================================================================
# Step 1: Quick mail check
# ============================================================================

# Record heartbeat (best effort)
substrate heartbeat $session_args --format context 2>/dev/null || true

# Quick (non-blocking) check for mail
quick_result=$(substrate poll $session_args --format hook --quiet 2>/dev/null || echo '{"decision": null}')
quick_decision=$(echo "$quick_result" | jq -r '.decision // empty')

if [ "$quick_decision" = "block" ]; then
    # Mail exists - output the result immediately
    echo "$quick_result"
    exit 0
fi

# ============================================================================
# Step 2: Check for incomplete tasks
# ============================================================================

# Function to count incomplete tasks for a session.
count_incomplete_tasks() {
    local task_dir="$HOME/.claude/tasks/$1"

    if [ ! -d "$task_dir" ]; then
        echo "0"
        return
    fi

    local count=0
    for task_file in "$task_dir"/*.json; do
        [ -f "$task_file" ] || continue

        # Check if status is not "completed"
        local status=$(jq -r '.status // "pending"' "$task_file" 2>/dev/null)
        if [ "$status" != "completed" ]; then
            count=$((count + 1))
        fi
    done

    echo "$count"
}

# Function to list incomplete tasks for a session.
list_incomplete_tasks() {
    local task_dir="$HOME/.claude/tasks/$1"
    local output=""

    if [ ! -d "$task_dir" ]; then
        return
    fi

    for task_file in "$task_dir"/*.json; do
        [ -f "$task_file" ] || continue

        local status=$(jq -r '.status // "pending"' "$task_file" 2>/dev/null)
        if [ "$status" != "completed" ]; then
            local id=$(jq -r '.id' "$task_file" 2>/dev/null)
            local subject=$(jq -r '.subject' "$task_file" 2>/dev/null | head -c 50)
            output="${output}#${id} [${status}], "
        fi
    done

    echo "$output"
}

if [ -n "$session_id" ]; then
    incomplete_count=$(count_incomplete_tasks "$session_id")

    if [ "$incomplete_count" -gt 0 ]; then
        task_list=$(list_incomplete_tasks "$session_id")

        cat <<EOF
{"decision": "block", "reason": "${incomplete_count} incomplete task(s): ${task_list}Complete ALL tasks before stopping."}
EOF
        exit 0
    fi
fi

# ============================================================================
# Steps 2.5 & 3 removed: automated diff and status messages were noisy and
# flooded the inbox with near-identical content (see issue #78). Diffs are
# now sent explicitly via `substrate send-diff` with content-based
# idempotency, and heartbeats handle liveness for the UI.
# ============================================================================

# ============================================================================
# Step 4: Long poll to keep agent alive
# ============================================================================

# No mail, no tasks - do a longer poll to keep agent alive.
# --always-block ensures we output block decision even with no messages.
# This keeps the agent alive indefinitely, continuously checking for work.
#
# When the server is unreachable (poll fails immediately), we fall back to
# a local sleep-based retry loop to avoid churning the hook in a tight loop.

# Debug log for poll.
debug_log="$HOME/.subtrate/stop_hook_trace.log"
mkdir -p "$(dirname "$debug_log")"
echo "=== Stop Hook Step 4: $(date) ===" >> "$debug_log"
echo "session_args: [$session_args]" >> "$debug_log"

# Record start time so we can fill the full 9m30s window on failure.
start_time=$(date +%s)
max_duration=570  # 9m30s

poll_output=$(substrate poll $session_args --wait=${max_duration}s --format hook --always-block 2>&1)
poll_exit=$?
echo "poll_exit: $poll_exit, output: $poll_output" >> "$debug_log"

if [ $poll_exit -eq 0 ]; then
    # Output poll result directly. The poll command already includes a
    # suitable reason; no need to inject "Standing by" text.
    echo "$poll_output"
else
    # Poll failed (server likely down). Sleep-retry to fill the remaining
    # time window instead of returning immediately, which would cause the
    # hook to churn in a tight loop.
    elapsed=$(( $(date +%s) - start_time ))
    remaining=$(( max_duration - elapsed ))
    echo "Poll failed, entering sleep-retry loop (${remaining}s remaining)" >> "$debug_log"

    retry_interval=30
    while [ "$remaining" -gt 0 ]; do
        sleep_time=$retry_interval
        if [ "$sleep_time" -gt "$remaining" ]; then
            sleep_time=$remaining
        fi
        sleep "$sleep_time"
        remaining=$(( max_duration - ($(date +%s) - start_time) ))

        # Try polling again in case the server came back up.
        retry_output=$(substrate poll $session_args --wait=5s --format hook --always-block 2>/dev/null)
        if [ $? -eq 0 ]; then
            has_mail=$(echo "$retry_output" | jq -r '.reason // ""' | grep -c "unread")
            if [ "$has_mail" -gt 0 ]; then
                echo "Server recovered, mail found" >> "$debug_log"
                echo "$retry_output"
                exit 0
            fi
        fi

        remaining=$(( max_duration - ($(date +%s) - start_time) ))
        echo "Retry at $(date), ${remaining}s remaining" >> "$debug_log"
    done

    echo '{"decision": "block", "reason": "Server unreachable, retrying."}'
fi
