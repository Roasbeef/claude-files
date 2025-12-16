#!/bin/bash
# PreToolUse hook to encourage task management workflow
# Reminds Claude when editing without an active task

set -euo pipefail

input=$(cat)

tool=$(echo "$input" | jq -r '.tool // empty')

# Only check for file modification tools
if [[ "$tool" != "Edit" && "$tool" != "Write" && "$tool" != "MultiEdit" ]]; then
    exit 0
fi

# Check if we're in a project with a .tasks directory
if [ ! -d ".tasks" ]; then
    exit 0
fi

# Check if there's an in_progress task
if [ -d ".tasks/active" ]; then
    in_progress_count=$(find .tasks/active -name "*.md" -exec grep -l "status: in_progress" {} \; 2>/dev/null | wc -l | tr -d ' ')

    if [ "$in_progress_count" -eq 0 ]; then
        echo "💡 Reminder: No task currently in_progress. Consider using /task-next to select a task."
        # Don't block - just provide feedback
    fi
fi

exit 0
