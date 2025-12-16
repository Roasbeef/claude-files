#!/bin/bash
# SessionStart hook: Load project and session context
# Displays active session state for immediate context restoration

# Don't use set -e because grep returns 1 when no matches found
set -uo pipefail

echo ""
echo "🚀 Claude Code Session Starting"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Show current directory
current_dir=$(pwd)
echo "📁 Project: $(basename "$current_dir")"

# Git status
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    branch=$(git branch --show-current 2>/dev/null || echo "detached")
    echo "🌿 Branch: $branch"

    changes=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')
    if [ "$changes" -gt 0 ]; then
        echo "   Uncommitted: $changes file(s)"
    fi
fi

echo ""

# Check for active session - PRIMARY CONTEXT SOURCE
if [ -d ".sessions/active" ]; then
    active_session=$(find .sessions/active -name "*.md" -type f 2>/dev/null | head -1)

    if [ -n "$active_session" ] && [ -f "$active_session" ]; then
        session_id=$(grep "^id:" "$active_session" 2>/dev/null | cut -d: -f2- | xargs || echo "")
        shortname=$(grep "^shortname:" "$active_session" 2>/dev/null | cut -d: -f2- | xargs || echo "unknown")
        compactions=$(grep "^compaction_count:" "$active_session" 2>/dev/null | cut -d: -f2 | xargs || echo "0")

        echo "📋 Active Session: $shortname"
        if [ -n "$compactions" ] && [ "$compactions" -gt 0 ] 2>/dev/null; then
            echo "   (after $compactions compaction(s))"
        fi
        echo ""

        # Show TL;DR for quick context
        if grep -q "^## TL;DR" "$active_session" 2>/dev/null; then
            echo "## TL;DR"
            sed -n '/^## TL;DR/,/^## /p' "$active_session" | tail -n +2 | grep -v "^## " | head -6 || true
            echo ""
        fi

        # Show progress checklist
        progress_done=$(grep -c "^\- \[x\]" "$active_session" 2>/dev/null || echo "0")
        progress_total=$(grep -c "^\- \[" "$active_session" 2>/dev/null || echo "0")
        if [ "$progress_total" -gt 0 ] 2>/dev/null; then
            echo "## Progress ($progress_done/$progress_total)"
            grep "^\- \[" "$active_session" 2>/dev/null | head -7 || true
            echo ""
        fi

        # Show blockers if any
        if grep -q "^## Blockers" "$active_session" 2>/dev/null; then
            blockers=$(sed -n '/^## Blockers/,/^## /p' "$active_session" | grep -v "^## " | grep -v "^- None" | grep -v "^$" | head -3 || true)
            if [ -n "$blockers" ]; then
                echo "## Blockers ⚠️"
                echo "$blockers"
                echo ""
            fi
        fi

        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        echo "💡 Commands:"
        echo "   /session-resume    Full context restoration"
        echo "   /session-view      View session details"
        echo "   /session-log       Add progress entries"
        echo ""

    else
        # No active session, check for paused sessions
        if [ -d ".sessions/active" ]; then
            paused=$(find .sessions/active -name "*.md" -exec grep -l "status: paused" {} \; 2>/dev/null | wc -l | tr -d ' ' || echo "0")
            if [ "$paused" -gt 0 ] 2>/dev/null; then
                echo "⏸️  $paused paused session(s) available"
                echo "   Run /session-resume --list to see them"
                echo ""
            fi
        fi

        # Check for task system
        if [ -d ".tasks/active" ]; then
            in_progress=$(find .tasks/active -name "*.md" -exec grep -l "status: in_progress" {} \; 2>/dev/null | wc -l | tr -d ' ' || echo "0")
            ready=$(find .tasks/active -name "*.md" -exec grep -l "status: ready" {} \; 2>/dev/null | wc -l | tr -d ' ' || echo "0")

            echo "📋 Tasks: $in_progress in progress, $ready ready"

            if [ "$in_progress" -gt 0 ] 2>/dev/null; then
                echo "   💡 Start a session: /session-init --task=<id>"
            elif [ "$ready" -gt 0 ] 2>/dev/null; then
                echo "   💡 Pick a task: /task-next"
            fi
            echo ""
        else
            echo "💡 Start a session: /session-init --goal=\"description\""
            echo ""
        fi
    fi
else
    # No session system, show task system if available
    if [ -d ".tasks/active" ]; then
        total=$(find .tasks/active -name "*.md" 2>/dev/null | wc -l | tr -d ' ' || echo "0")
        in_progress=$(find .tasks/active -name "*.md" -exec grep -l "status: in_progress" {} \; 2>/dev/null | wc -l | tr -d ' ' || echo "0")

        echo "📋 Task System: $total tasks ($in_progress in progress)"
        echo "   💡 Start a session for execution continuity:"
        echo "      /session-init --task=<id>"
        echo ""
    fi
fi

# Project type detection (compact)
if [ -f "go.mod" ]; then
    echo "🔵 Go project"
elif [ -f "package.json" ]; then
    echo "🟢 Node.js project"
elif [ -f "Cargo.toml" ]; then
    echo "🦀 Rust project"
elif [ -f "pyproject.toml" ] || [ -f "setup.py" ]; then
    echo "🐍 Python project"
fi

echo ""
exit 0
