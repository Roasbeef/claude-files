#!/bin/bash

# Task Management Aliases
# Source this file in your shell profile to get quick access to task commands
# Add to ~/.bashrc or ~/.zshrc:
#   source ~/.claude/scripts/task-aliases.sh

# Quick task listing
alias tasks='~/.claude/scripts/list-tasks.sh'

# List all projects with tasks
alias all-tasks='~/.claude/scripts/list-all-tasks.sh'

# Quick navigation to task directory
task-cd() {
    if [ -d ".tasks/active" ]; then
        cd .tasks/active
    else
        echo "No .tasks/active directory in current location"
    fi
}

# View a specific task by shortname
task-cat() {
    if [ -z "$1" ]; then
        echo "Usage: task-cat <shortname>"
        return 1
    fi

    local file=$(find .tasks/active -name "$1*.md" -type f 2>/dev/null | head -1)
    if [ -f "$file" ]; then
        cat "$file"
    else
        echo "Task not found: $1"
    fi
}

# Quick task status check
task-status() {
    if [ ! -d ".tasks/active" ]; then
        echo "No tasks in current directory"
        return 1
    fi

    local total=$(find .tasks/active -name "*.md" -type f | wc -l | tr -d ' ')
    local ready=$(grep -l "^status: ready$" .tasks/active/*.md 2>/dev/null | wc -l | tr -d ' ')
    local blocked=$(grep -l "^blocked_by:" .tasks/active/*.md 2>/dev/null | grep -v "blocked_by: \[\]" | wc -l | tr -d ' ')
    local in_progress=$(grep -l "^status: in_progress$" .tasks/active/*.md 2>/dev/null | wc -l | tr -d ' ')

    echo "📊 Task Status:"
    echo "  Total: $total"
    echo "  Ready: $ready ✅"
    echo "  In Progress: $in_progress 🔄"
    echo "  Blocked: $blocked 🔒"
}

echo "Task aliases loaded! Available commands:"
echo "  tasks         - List tasks in current project"
echo "  all-tasks     - List tasks in all projects"
echo "  task-cd       - Change to .tasks/active directory"
echo "  task-cat <id> - View a specific task"
echo "  task-status   - Quick status summary"