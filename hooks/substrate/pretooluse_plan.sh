#!/bin/bash
# pretooluse_plan.sh - PermissionRequest hook for intercepting ExitPlanMode.

set -euo pipefail

DEBUG_LOG="/tmp/substrate-plan-hook.log"
echo "$(date): Hook invoked" >> "$DEBUG_LOG"

HOOK_INPUT=$(cat)

if ! command -v jq &>/dev/null; then
    exit 0
fi

TOOL=$(echo "$HOOK_INPUT" | jq -r '.tool_name // empty')
SESSION_ID=$(echo "$HOOK_INPUT" | jq -r '.session_id // empty')
CWD=$(echo "$HOOK_INPUT" | jq -r '.cwd // empty')

echo "$(date): TOOL=$TOOL SESSION=$SESSION_ID" >> "$DEBUG_LOG"

if [[ "$TOOL" != "ExitPlanMode" ]]; then
    exit 0
fi

export CLAUDE_SESSION_ID="$SESSION_ID"

SUBSTRATE="${SUBSTRATE_CLI:-substrate}"
if ! command -v "$SUBSTRATE" &>/dev/null; then
    if [[ -x "$HOME/go/bin/substrate" ]]; then
        SUBSTRATE="$HOME/go/bin/substrate"
    elif [[ -x "$HOME/gocode/bin/substrate" ]]; then
        SUBSTRATE="$HOME/gocode/bin/substrate"
    elif [[ -x "/usr/local/bin/substrate" ]]; then
        SUBSTRATE="/usr/local/bin/substrate"
    else
        exit 0
    fi
fi

# Submit plan WITHOUT AI summarization (--no-ai) to avoid the 30+ second
# delay that causes Claude Code to show its fallback UI. This makes
# submit instant (<1s) matching plannotator's fast-start behavior.
echo "$(date): Submitting plan (no-ai)..." >> "$DEBUG_LOG"
SUBMIT_OUTPUT=$("$SUBSTRATE" plan submit \
    --session-id "$SESSION_ID" \
    --cwd "$CWD" \
    --no-ai \
    --format json 2>>"$DEBUG_LOG") || {
    echo "$(date): plan submit failed" >> "$DEBUG_LOG"
    exit 0
}

echo "$(date): Submit output: $SUBMIT_OUTPUT" >> "$DEBUG_LOG"

PLAN_REVIEW_ID=$(echo "$SUBMIT_OUTPUT" | jq -r '.plan_review_id // empty')

if [[ -z "$PLAN_REVIEW_ID" ]]; then
    echo "$(date): No plan_review_id" >> "$DEBUG_LOG"
    exit 0
fi

echo "$(date): Waiting for review $PLAN_REVIEW_ID" >> "$DEBUG_LOG"

"$SUBSTRATE" plan wait \
    --plan-review-id "$PLAN_REVIEW_ID" \
    --timeout 96h \
    --format hook 2>>"$DEBUG_LOG" || {
    echo "$(date): plan wait failed" >> "$DEBUG_LOG"
    echo '{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","message":"Plan review check failed. Retry ExitPlanMode."}}}'
    exit 0
}
