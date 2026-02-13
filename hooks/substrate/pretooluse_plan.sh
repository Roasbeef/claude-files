#!/bin/bash
# pretooluse_plan.sh - PreToolUse hook for intercepting ExitPlanMode.
# This script is called before ExitPlanMode executes. It submits the plan
# for review via the substrate CLI and blocks for up to 9 minutes waiting
# for approval.
#
# Claude Code passes hook data via STDIN as JSON with these fields:
#   tool_name     - Name of the tool (ExitPlanMode)
#   tool_input    - JSON input parameters
#   session_id    - Current session ID
#   cwd           - Current working directory
#
# Output: JSON with hookSpecificOutput.permissionDecision (allow/deny).
# If substrate is unavailable, falls through (exit 0, no output → allow).

set -euo pipefail

# Read all hook data from stdin.
HOOK_INPUT=$(cat)

# Parse fields using jq. Fall through if jq unavailable.
if ! command -v jq &>/dev/null; then
    exit 0
fi

TOOL=$(echo "$HOOK_INPUT" | jq -r '.tool_name // empty')
SESSION_ID=$(echo "$HOOK_INPUT" | jq -r '.session_id // empty')
CWD=$(echo "$HOOK_INPUT" | jq -r '.cwd // empty')

# Only process ExitPlanMode tool calls.
if [[ "$TOOL" != "ExitPlanMode" ]]; then
    exit 0
fi

# Determine substrate command location.
SUBSTRATE="${SUBSTRATE_CLI:-substrate}"
if ! command -v "$SUBSTRATE" &>/dev/null; then
    if [[ -x "$HOME/go/bin/substrate" ]]; then
        SUBSTRATE="$HOME/go/bin/substrate"
    elif [[ -x "$HOME/gocode/bin/substrate" ]]; then
        SUBSTRATE="$HOME/gocode/bin/substrate"
    elif [[ -x "/usr/local/bin/substrate" ]]; then
        SUBSTRATE="/usr/local/bin/substrate"
    else
        # Substrate not available — fall through, allow ExitPlanMode.
        exit 0
    fi
fi

# Submit the plan for review.
SUBMIT_OUTPUT=$("$SUBSTRATE" plan submit \
    --session-id "$SESSION_ID" \
    --cwd "$CWD" \
    --format json 2>/dev/null) || {
    # Submit failed — fall through, allow ExitPlanMode.
    exit 0
}

# Extract plan_review_id from the submit response.
PLAN_REVIEW_ID=$(echo "$SUBMIT_OUTPUT" | jq -r '.plan_review_id // empty')

if [[ -z "$PLAN_REVIEW_ID" ]]; then
    # Could not get plan review ID — fall through.
    exit 0
fi

# Wait for the review decision (up to 9 minutes).
# The wait command returns hook-format JSON with permissionDecision.
"$SUBSTRATE" plan wait \
    --plan-review-id "$PLAN_REVIEW_ID" \
    --timeout 9m \
    --format hook 2>/dev/null || {
    # Wait failed — deny to be safe.
    echo '{"hookSpecificOutput":{"permissionDecision":"deny","permissionDecisionReason":"Plan review check failed. Retry ExitPlanMode."}}'
    exit 0
}
