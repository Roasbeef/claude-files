#!/bin/bash

# Task List Viewer for Claude Task Management System
# This script displays tasks from a project's .tasks/active/ directory
# Usage: list-tasks.sh [directory]

# Use provided directory or current directory
TASK_DIR="${1:-.}"

# Check if .tasks directory exists
if [ ! -d "$TASK_DIR/.tasks/active" ]; then
    echo "❌ No .tasks/active/ directory found in $TASK_DIR"
    echo "   Run this from a project root or specify the project directory"
    exit 1
fi

# Get project name (use full path basename if current directory)
if [ "$TASK_DIR" = "." ]; then
    PROJECT_NAME=$(basename "$(pwd)")
else
    PROJECT_NAME=$(basename "$TASK_DIR")
fi

echo "📋 Active Tasks in $PROJECT_NAME"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Arrays to store task data for sorting
declare -a tasks

# Process each task file
for file in "$TASK_DIR"/.tasks/active/*.md; do
    if [ -f "$file" ]; then
        # Extract metadata from YAML frontmatter
        priority=$(awk '/^priority:/ {print $2}' "$file")
        size=$(awk '/^size:/ {print $2}' "$file")
        status=$(awk '/^status:/ {print $2}' "$file")
        title=$(awk '/^title:/ {$1=""; print substr($0,2)}' "$file")
        shortname=$(awk '/^shortname:/ {print $2}' "$file")
        blocked_by=$(awk '/^blocked_by:/ {$1=""; print substr($0,2)}' "$file")
        assignee=$(awk '/^assignee:/ {print $2}' "$file")

        # Determine status icon and actual status
        if [ "$blocked_by" != "[]" ] && [ "$blocked_by" != "" ]; then
            status_icon="🔒"
            display_status="blocked"
        elif [ "$status" = "in_progress" ]; then
            status_icon="🔄"
            display_status="in_progress"
            if [ "$assignee" != "" ] && [ "$assignee" != "null" ]; then
                display_status="$display_status @$assignee"
            fi
        elif [ "$status" = "ready" ]; then
            status_icon="✅"
            display_status="ready"
        else
            status_icon="📝"
            display_status="$status"
        fi

        # Format output line
        printf "%s [%s] %-30s [%s] [%s]\n" "$status_icon" "$priority" "$shortname" "$size" "$display_status"
        echo "   $title"
        if [ "$blocked_by" != "[]" ] && [ "$blocked_by" != "" ]; then
            echo "   ⚠️  Blocked by: $blocked_by"
        fi
        echo ""
    fi
done

# Count tasks by status
total=$(find "$TASK_DIR"/.tasks/active -name "*.md" -type f | wc -l | tr -d ' ')
ready=$(grep -l "^status: ready$" "$TASK_DIR"/.tasks/active/*.md 2>/dev/null | wc -l | tr -d ' ')
in_progress=$(grep -l "^status: in_progress$" "$TASK_DIR"/.tasks/active/*.md 2>/dev/null | wc -l | tr -d ' ')

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Summary: $total active tasks | $ready ready | $in_progress in progress"
echo "Legend: ✅ Ready | 🔄 In Progress | 🔒 Blocked"
echo ""
echo "Commands:"
echo "  /task-next        - Start highest priority task"
echo "  /task-view <id>   - View task details"
echo "  /task-complete    - Complete current task"