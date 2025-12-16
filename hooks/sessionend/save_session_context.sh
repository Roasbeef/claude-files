#!/bin/bash
# SessionEnd hook to save session context for next time
# Archives what was accomplished and suggests next steps

set -euo pipefail

current_dir=$(pwd)
project_name=$(basename "$current_dir")

# Create session context directory
mkdir -p "$HOME/.claude/session-context"

context_file="$HOME/.claude/session-context/${project_name}_last_session.txt"

# Save session summary
{
    echo "Session ended: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""

    # Git changes summary
    if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        changes=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')
        if [ "$changes" -gt 0 ]; then
            echo "Files modified in session: $changes"
            git status --short 2>/dev/null | head -10
            if [ "$changes" -gt 10 ]; then
                echo "   ... and $((changes - 10)) more files"
            fi
            echo ""
        fi
    fi

    # Task system updates
    if [ -d ".tasks" ]; then
        in_progress=$(find .tasks/active -name "*.md" -exec grep -l "status: in_progress" {} \; 2>/dev/null | wc -l | tr -d ' ')
        if [ "$in_progress" -gt 0 ]; then
            echo "Tasks still in progress: $in_progress"
            echo "Consider using /task-complete or /task-status to update"
            echo ""
        fi
    fi

    echo "Next session: Continue from where you left off"
} > "$context_file"

# Optional: Archive session to history
if [ -d ".claude-sessions" ]; then
    session_id=$(date '+%Y%m%d_%H%M%S')
    cp "$context_file" ".claude-sessions/session_${session_id}.txt"
fi

echo ""
echo "📝 Session context saved for next time"
echo ""

exit 0
