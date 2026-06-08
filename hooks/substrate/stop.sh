#!/bin/bash
# Subtrate Stop hook - Arm-Once Watcher Pattern
#
# The persistence model is the background watcher (`substrate watch`),
# not the Stop hook. The watcher runs as a Claude Code background task;
# when mail arrives it exits, and the harness re-invokes the agent with
# the digest. The Stop hook's only job is to make sure a watcher is
# armed before the agent goes idle.
#
# Priority order:
# 1. Quick mail check - if mail exists, block with the digest (the agent
#    will read mail via tool calls, which resets Claude Code's
#    consecutive-block counter).
# 2. If a watcher is armed, allow exit - the agent will be woken by the
#    watcher's exit notification when there is work.
# 3. If stop_hook_active is true, allow exit - we already nudged once
#    this stop cycle. This respects the Claude Code 2.1.143+ cap on
#    consecutive Stop hook blocks (default 8). This check runs AFTER
#    the armed check so a mail block earlier in the turn (which sets
#    stop_hook_active for the rest of the turn) cannot suppress the
#    arming nudge and strand the agent without a watcher.
# 4. Otherwise block ONCE with arming instructions (folding in a
#    reminder about incomplete tasks, if any).
#
# Output format: JSON for Stop hook decision

# Read hook input from stdin
input=$(cat)
session_id=$(echo "$input" | jq -r '.session_id // empty')
stop_hook_active=$(echo "$input" | jq -r '.stop_hook_active // false')

# Build session ID args if available (critical for agent identity
# resolution). Use a bash array so a session_id containing whitespace
# or glob metacharacters cannot word-split or expand into extra args.
session_args=()
if [ -n "$session_id" ]; then
    session_args=(--session-id "$session_id")
fi

# ============================================================================
# Step 1: Quick mail check
# ============================================================================

# Record heartbeat (best effort)
substrate heartbeat "${session_args[@]}" --format context 2>/dev/null || true

# Quick (non-blocking) check for mail
quick_result=$(substrate poll "${session_args[@]}" --format hook --quiet 2>/dev/null || echo '{}')
quick_decision=$(echo "$quick_result" | jq -r '.decision // empty')

if [ "$quick_decision" = "block" ]; then
    # Mail exists - output the result immediately
    echo "$quick_result"
    exit 0
fi

# ============================================================================
# Step 2: If a watcher is armed, exit cleanly
# ============================================================================

# Note: allow is an empty object — newer Claude Code rejects
# {"decision": null}.
if substrate watch --check "${session_args[@]}" >/dev/null 2>&1; then
    echo '{}'
    exit 0
fi

# ============================================================================
# Step 3: Respect stop_hook_active - never block twice in a row
# ============================================================================

# stop_hook_active=true means Claude is stopping after a previous block
# this turn. No watcher is armed at this point (step 2 fell through),
# but we already had our chance to nudge; allow exit rather than
# accumulate consecutive blocks. The next user prompt or session start
# re-nudges.
if [ "$stop_hook_active" = "true" ]; then
    echo '{}'
    exit 0
fi

# ============================================================================
# Step 4: Block once with arming instructions
# ============================================================================

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
            output="${output}#${id} [${status}], "
        fi
    done

    echo "$output"
}

# Fold an incomplete-task reminder into the single arming block, rather
# than blocking separately for it (repeated task nags are what tripped
# the consecutive-block cap; /goal handles task gating natively now).
task_note=""
if [ -n "$session_id" ]; then
    task_list=$(list_incomplete_tasks "$session_id")
    if [ -n "$task_list" ]; then
        task_note="Note: incomplete task(s) remain: ${task_list}finish or update them first if appropriate. "
    fi
fi

reason="No mail watcher is armed. ${task_note}Arm the watcher now: run \`substrate watch --session-id ${session_id:-\$CLAUDE_SESSION_ID}\` via the Bash tool with run_in_background set to true, then end your turn. The watcher exits when mail arrives, which wakes you automatically with a digest; re-arm it after handling each wake."

jq -cn --arg reason "$reason" '{"decision": "block", "reason": $reason}'
