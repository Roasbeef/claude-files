#!/bin/bash
# Subtrate Notification hook - send mail on permission prompts and notifications
#
# This hook runs when Claude Code sends a notification (permission prompt,
# idle prompt, etc.). It sends a Subtrate mail to the User so they can see
# the event in the web UI without watching the terminal.
#
# Input JSON fields: session_id, message, title, notification_type, cwd
# Output format: plain text (informational only, cannot block)

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

exit 0
