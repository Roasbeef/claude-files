#!/bin/bash

# List tasks from all projects with .tasks directories
# Searches from current directory or specified base directory

BASE_DIR="${1:-$(pwd)}"

echo "🔍 Searching for projects with tasks in: $BASE_DIR"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Find all directories with .tasks/active subdirectory
projects_found=0
for task_dir in $(find "$BASE_DIR" -type d -path "*/.tasks/active" 2>/dev/null | sort); do
    project_dir=$(dirname "$(dirname "$task_dir")")
    project_name=$(basename "$project_dir")

    # Count tasks in this project
    task_count=$(find "$task_dir" -name "*.md" -type f 2>/dev/null | wc -l | tr -d ' ')

    if [ "$task_count" -gt 0 ]; then
        projects_found=$((projects_found + 1))
        echo "📁 Project: $project_name"
        echo "   Path: $project_dir"
        echo "   Tasks: $task_count active"

        # Show high priority tasks
        high_priority=$(grep -l "^priority: P[01]$" "$task_dir"/*.md 2>/dev/null | wc -l | tr -d ' ')
        if [ "$high_priority" -gt 0 ]; then
            echo "   ⚠️  High priority tasks: $high_priority"
        fi

        # Show in-progress tasks
        in_progress=$(grep -l "^status: in_progress$" "$task_dir"/*.md 2>/dev/null | wc -l | tr -d ' ')
        if [ "$in_progress" -gt 0 ]; then
            echo "   🔄 In progress: $in_progress"
        fi

        echo ""
    fi
done

if [ "$projects_found" -eq 0 ]; then
    echo "No projects with active tasks found."
else
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Found $projects_found projects with active tasks"
    echo ""
    echo "To view tasks in a specific project:"
    echo "  cd <project-dir> && ~/.claude/scripts/list-tasks.sh"
    echo "Or:"
    echo "  ~/.claude/scripts/list-tasks.sh <project-dir>"
fi