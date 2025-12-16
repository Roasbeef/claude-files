#!/bin/bash
# PreCompact hook: Auto-checkpoint active session before context compaction
# This ensures execution continuity across context window resets

# Don't use set -e because grep returns 1 when no matches found
set -uo pipefail

current_dir=$(pwd)
project_name=$(basename "$current_dir")

echo ""
echo "💾 Pre-Compaction Checkpoint"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check for active session
if [ -d ".sessions/active" ]; then
    active_session=$(find .sessions/active -name "*.md" -type f 2>/dev/null | head -1)

    if [ -n "$active_session" ] && [ -f "$active_session" ]; then
        # Extract session metadata
        session_id=$(grep "^id:" "$active_session" 2>/dev/null | cut -d: -f2- | xargs || echo "")
        shortname=$(grep "^shortname:" "$active_session" 2>/dev/null | cut -d: -f2- | xargs || echo "unknown")
        current_count=$(grep "^compaction_count:" "$active_session" 2>/dev/null | cut -d: -f2 | xargs || echo "0")
        new_count=$((current_count + 1))

        echo "Session: $shortname"
        echo "Compaction: #$new_count"
        echo ""

        # Update session file: increment compaction count and timestamp
        sed -i '' "s/^compaction_count: .*/compaction_count: $new_count/" "$active_session" 2>/dev/null || true
        sed -i '' "s/^updated_at: .*/updated_at: $(date -u +%Y-%m-%dT%H:%M:%SZ)/" "$active_session" 2>/dev/null || true

        # Update progress file if it exists
        if [ -n "$session_id" ]; then
            progress_file=".sessions/journal/${session_id}/progress.md"
            if [ -f "$progress_file" ]; then
                sed -i '' "s/^compaction_count: .*/compaction_count: $new_count/" "$progress_file" 2>/dev/null || true
                sed -i '' "s/^last_updated: .*/last_updated: $(date -u +%Y-%m-%dT%H:%M:%SZ)/" "$progress_file" 2>/dev/null || true
            fi
        fi

        # Output key context for the compaction summary
        echo "## Session Context (for next context window)"
        echo ""

        # TL;DR - most important for quick resume
        if grep -q "^## TL;DR" "$active_session" 2>/dev/null; then
            echo "### TL;DR"
            sed -n '/^## TL;DR/,/^## /p' "$active_session" | tail -n +2 | grep -v "^## " | head -8 || true
            echo ""
        fi

        # Current progress checklist
        echo "### Progress"
        grep "^\- \[" "$active_session" 2>/dev/null | head -10 || echo "(no checklist items)"
        echo ""

        # Key context section
        if grep -q "^\*\*Key Context\*\*" "$active_session" 2>/dev/null; then
            echo "### Key Context"
            sed -n '/^\*\*Key Context\*\*/,/^## /p' "$active_session" | head -10 || true
            echo ""
        fi

        # Next steps
        if grep -q "^## Next Steps" "$active_session" 2>/dev/null; then
            echo "### Next Steps"
            sed -n '/^## Next Steps/,/^## /p' "$active_session" | tail -n +2 | grep -v "^## " | head -5 || true
            echo ""
        fi

        # Current blockers
        if grep -q "^## Blockers" "$active_session" 2>/dev/null; then
            blockers=$(sed -n '/^## Blockers/,/^## /p' "$active_session" | grep -v "^## " | grep -v "^- None" | head -3 || true)
            if [ -n "$blockers" ]; then
                echo "### Blockers"
                echo "$blockers"
                echo ""
            fi
        fi

        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        echo "💡 After compaction, run /session-resume for full context"
        echo ""

    else
        echo "No active session found."
        echo ""
        echo "💡 Start a session with /session-init for execution continuity"
        echo ""
    fi
else
    echo "No session system detected in this project."
    echo ""
    echo "💡 Initialize sessions with /session-init for execution continuity"
    echo ""
fi

# Also save git state for reference
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    branch=$(git branch --show-current 2>/dev/null || echo "detached")
    changes=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')

    echo "Git: $branch"
    if [ "$changes" -gt 0 ] 2>/dev/null; then
        echo "     $changes uncommitted file(s)"
    fi
    echo ""
fi

exit 0
