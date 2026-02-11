#!/bin/bash
# Subtrate Stop hook - Mail-First Persistent Agent Pattern
#
# Priority order:
# 1. Quick mail check - if mail exists, block immediately
# 2. Check Tasks - if incomplete tasks exist, block immediately
# 3. Send status update to User (with deduplication)
# 4. Long poll - keep agent alive, continuously checking for work
#
# Key behaviors:
# - Checks mail before tasks (mail is more actionable)
# - Sends status updates to User with deduplication
# - Long-polls for 9m30s (under 10m hook timeout)
# - Always outputs {"decision": "block"} to stay alive
# - User can force exit with Ctrl+C (bypasses hooks)
#
# Output format: JSON for Stop hook decision

# Prevent recursion when claude -p spawns its own Stop hook.
# The env var is set when we invoke claude -p below for status summaries.
if [ "$SUBSTRATE_SUMMARIZING" = "1" ]; then
    echo '{"decision": "approve"}'
    exit 0
fi

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
# Step 2.5: Send git diff summary (with deduplication)
# ============================================================================

# Send diff summary in background if there are changes since the base branch.
{
    diff_flag="$HOME/.subtrate/diff_sent_${session_id:-default}"
    now_diff=$(date +%s)

    # Only send once per session (or every 30 minutes).
    send_diff=true
    if [ -f "$diff_flag" ]; then
        last_diff=$(cat "$diff_flag" 2>/dev/null || echo "0")
        elapsed_diff=$((now_diff - last_diff))
        if [ "$elapsed_diff" -lt 1800 ]; then
            send_diff=false
        fi
    fi

    if [ "$send_diff" = "true" ]; then
        project_dir="${CLAUDE_PROJECT_DIR:-$(pwd)}"

        # Only send if there are actual changes.
        if git -C "$project_dir" diff --quiet HEAD 2>/dev/null; then
            # No uncommitted changes - check committed changes vs base.
            base_branch="main"
            current_branch=$(git -C "$project_dir" branch --show-current 2>/dev/null)
            if [ -n "$current_branch" ] && [ "$current_branch" != "$base_branch" ]; then
                if ! git -C "$project_dir" diff --quiet "$base_branch...$current_branch" 2>/dev/null; then
                    substrate send-diff $session_args --repo "$project_dir" >/dev/null 2>/dev/null && \
                        echo "$now_diff" > "$diff_flag"
                fi
            fi
        else
            # Has uncommitted changes.
            substrate send-diff $session_args --repo "$project_dir" >/dev/null 2>/dev/null && \
                echo "$now_diff" > "$diff_flag"
        fi
    fi
} &

# ============================================================================
# Step 3: Send status update (with deduplication)
# ============================================================================

# Send status update in background to not block the hook.
# Uses a flag file for deduplication - only sends if no recent status sent.
{
    # Debug log file
    debug_log="$HOME/.subtrate/stop_hook_debug.log"
    mkdir -p "$(dirname "$debug_log")"

    echo "=== Stop Hook Status Update $(date) ===" >> "$debug_log"
    echo "Session ID: ${session_id:-'(empty)'}" >> "$debug_log"
    echo "Session args: ${session_args:-'(empty)'}" >> "$debug_log"

    # Deduplication: check if we sent a status in the last 30 minutes.
    # Heartbeats already handle liveness for the UI, so status updates
    # are purely work-summary messages and don't need to be frequent.
    status_flag="$HOME/.subtrate/status_sent_${session_id:-default}"
    now=$(date +%s)

    if [ -f "$status_flag" ]; then
        last_sent=$(cat "$status_flag" 2>/dev/null || echo "0")
        elapsed=$((now - last_sent))
        echo "Dedup check: flag exists, last_sent=$last_sent, elapsed=${elapsed}s" >> "$debug_log"
        # Skip if sent within last 30 minutes (1800 seconds).
        if [ "$elapsed" -lt 1800 ]; then
            echo "SKIPPED: Within 30-minute dedup window" >> "$debug_log"
            exit 0
        fi
    else
        echo "Dedup check: no flag file exists" >> "$debug_log"
    fi

    # Get agent info
    # Format is "Current agent: AgentName (ID: N)"
    agent_name=$(substrate identity current $session_args --format text 2>/dev/null | sed -n 's/Current agent: \([^ ]*\).*/\1/p' || echo "Unknown")
    echo "Agent name lookup result: '$agent_name'" >> "$debug_log"

    # Skip if no valid agent
    if [ "$agent_name" = "Unknown" ] || [ -z "$agent_name" ]; then
        echo "SKIPPED: No valid agent (name='$agent_name')" >> "$debug_log"
        exit 0
    fi

    # Get project info
    project_dir="${CLAUDE_PROJECT_DIR:-$(pwd)}"
    project_name=$(basename "$project_dir")
    git_branch=$(git -C "$project_dir" branch --show-current 2>/dev/null || echo "unknown")

    # Try to generate summary using multiple strategies.
    summary=""

    # Compute project hash for Claude's projects directory.
    project_hash=$(echo "$project_dir" | tr '/.' '-')
    claude_projects_dir="$HOME/.claude/projects/$project_hash"

    # Strategy 1: Use claude -p with haiku to summarize session log.
    # SUBSTRATE_SUMMARIZING=1 prevents recursion (checked at top of script).
    # Run in subshell with redirected stdio to isolate from parent terminal.
    if command -v claude >/dev/null 2>&1 && [ -n "$session_id" ]; then
        echo "Strategy 1: Running claude -p (file-based)" >> "$debug_log"

        session_log="$claude_projects_dir/$session_id.jsonl"
        if [ -f "$session_log" ]; then
            echo "Found session log: $session_log" >> "$debug_log"

            # Use temp files for input/output to fully isolate from parent process.
            tmp_in=$(mktemp)
            tmp_out=$(mktemp)
            tail -10 "$session_log" 2>/dev/null | head -c 50000 > "$tmp_in"

            # Run in subshell with all stdio redirected to files.
            (
                SUBSTRATE_SUMMARIZING=1 claude -p \
                    "Summarize what this Claude Code agent accomplished in 2-3 brief bullet points. Be very concise." \
                    --model haiku < "$tmp_in" > "$tmp_out" 2>/dev/null
            ) </dev/null >/dev/null 2>/dev/null

            summary=$(cat "$tmp_out" 2>/dev/null || echo "")
            rm -f "$tmp_in" "$tmp_out"

            if [ -n "$summary" ]; then
                echo "Generated summary with claude -p haiku" >> "$debug_log"
            else
                echo "claude -p returned empty summary" >> "$debug_log"
            fi
        else
            echo "Session log not found: $session_log" >> "$debug_log"
        fi
    else
        echo "Strategy 1: SKIPPED (claude not available or no session_id)" >> "$debug_log"
    fi

    # Strategy 2: Check ~/.claude/projects/{project-hash}/sessions-index.json
    if [ -z "$summary" ]; then
        sessions_index="$claude_projects_dir/sessions-index.json"
        echo "Strategy 2: Looking for sessions-index at: $sessions_index" >> "$debug_log"

        if [ -f "$sessions_index" ]; then
            latest_summary=$(jq -r '.entries[-1].summary // empty' "$sessions_index" 2>/dev/null)
            if [ -n "$latest_summary" ]; then
                summary="$latest_summary"
                echo "Got summary from sessions-index.json: $summary" >> "$debug_log"
            fi
        fi
    fi

    # Strategy 3: Fallback to project's .sessions/active/ directory TL;DR.
    if [ -z "$summary" ]; then
        session_dir="$project_dir/.sessions/active"
        echo "Strategy 3: Looking for session files in: $session_dir" >> "$debug_log"

        if [ -d "$session_dir" ]; then
            session_file=$(ls -t "$session_dir"/*.md 2>/dev/null | head -1)
            if [ -n "$session_file" ] && [ -f "$session_file" ]; then
                tldr=$(sed -n '/^## TL;DR/,/^## /p' "$session_file" 2>/dev/null | head -10 | tail -n +2)
                if [ -n "$tldr" ]; then
                    summary="$tldr"
                    echo "Extracted TL;DR from session file" >> "$debug_log"
                fi
            fi
        fi
    fi

    # Fallback summary
    if [ -z "$summary" ]; then
        summary="Agent idle, waiting for next task."
        echo "Using fallback summary" >> "$debug_log"
    fi

    # Build status message
    status_body="[Context: Working on $project_name, branch: $git_branch]

$summary

---
(Automated status update - agent standing by)"

    echo "Attempting to send status update..." >> "$debug_log"
    echo "Subject: [Status] $agent_name - Standing By" >> "$debug_log"

    # Send to User (redirect stdout to prevent corrupting hook JSON output).
    if substrate send $session_args \
        --to User \
        --subject "[Status] $agent_name - Standing By" \
        --body "$status_body" \
        >>"$debug_log" 2>&1; then
            # Update deduplication flag
            mkdir -p "$(dirname "$status_flag")"
            echo "$now" > "$status_flag"
            echo "SUCCESS: Status sent, flag updated at $status_flag" >> "$debug_log"
    else
        echo "FAILED: substrate send returned error" >> "$debug_log"
    fi
} &

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

# Generate unique poll UUID to prevent output deduplication by Claude Code.
poll_uuid=$(uuidgen | tr '[:upper:]' '[:lower:]' | cut -d'-' -f1)
echo "poll_uuid: $poll_uuid" >> "$debug_log"

if [ $poll_exit -eq 0 ]; then
    # Replace the reason with one that includes the UUID for Claude to echo.
    echo "$poll_output" | jq -c --arg uuid "$poll_uuid" \
        '.reason = "No new messages. Say: Standing by [" + $uuid + "]"'
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
                echo "$retry_output" | jq -c --arg uuid "$poll_uuid" \
                    '.reason = .reason + " [" + $uuid + "]"'
                exit 0
            fi
        fi

        remaining=$(( max_duration - ($(date +%s) - start_time) ))
        echo "Retry at $(date), ${remaining}s remaining" >> "$debug_log"
    done

    echo '{"decision": "block", "reason": "Server unreachable. Say: Standing by ['"$poll_uuid"']"}'
fi
