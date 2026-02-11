#!/bin/bash
# Subtrate Notification hook - send mail on permission prompts and notifications
#
# This hook runs when Claude Code sends a notification (permission prompt,
# idle prompt, etc.). It sends a Subtrate mail to the User so they can see
# the event in the web UI without watching the terminal.
#
# For idle_prompt, it also outputs JSON with additionalContext to wake the
# agent and prompt it to check mail, preventing indefinite idle states.
#
# Input JSON fields: session_id, message, title, notification_type, cwd
# Output: JSON with hookSpecificOutput.additionalContext (for idle_prompt)

# Read hook input from stdin.
input=$(cat)
session_id=$(echo "$input" | jq -r '.session_id // empty')
message=$(echo "$input" | jq -r '.message // empty')
title=$(echo "$input" | jq -r '.title // "Notification"')
notif_type=$(echo "$input" | jq -r '.notification_type // "unknown"')
cwd=$(echo "$input" | jq -r '.cwd // empty')

# Nothing to send if there's no message.
if [ -z "$message" ]; then
    exit 0
fi

# Build session ID args.
session_args=""
if [ -n "$session_id" ]; then
    session_args="--session-id $session_id"
fi

# Choose subject prefix based on notification type.
case "$notif_type" in
    permission_prompt)
        subject="[Permission] $title"
        ;;
    idle_prompt)
        subject="[Idle] $title"
        ;;
    *)
        subject="[Notification] $title"
        ;;
esac

# Build body with context.
body="$message"
if [ -n "$cwd" ]; then
    body="$body

Working directory: $cwd"
fi
body="$body
Type: $notif_type"

# Send mail in background so the hook returns immediately.
{
    substrate send $session_args \
        --to User \
        --subject "$subject" \
        --body "$body" \
        >/dev/null 2>/dev/null || true
} &

# For idle_prompt, output JSON with additionalContext to wake the agent.
# Without this, the hook silently consumes the idle event and the agent
# stays idle forever instead of re-entering the Stop hook polling loop.
if [ "$notif_type" = "idle_prompt" ]; then
    # Quick check for unread mail to include in the context.
    mail_count=$(substrate status $session_args --format json 2>/dev/null \
        | jq -r '.unread // 0' 2>/dev/null || echo "0")

    if [ "$mail_count" -gt 0 ] && [ "$mail_count" != "0" ]; then
        context="You have $mail_count unread message(s). Check your inbox with: substrate inbox $session_args"
    else
        context="You've been idle. Check for new messages from other agents with: substrate inbox $session_args"
    fi

    # Use proper JSON output format per Claude Code hooks spec.
    jq -n --arg ctx "$context" '{
        hookSpecificOutput: {
            hookEventName: "Notification",
            additionalContext: $ctx
        }
    }'
fi

exit 0
