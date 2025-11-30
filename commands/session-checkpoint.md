---
description: Save an explicit checkpoint of current session state
argument-hint: [--note="description"] [--pre-compact]
---

Create an explicit checkpoint of the current session state for continuity.

Arguments: $ARGUMENTS

## When to Use:
- Before taking a break from work
- Before making risky changes
- When you want to ensure state is saved
- Before context compaction (auto-triggered by PreCompact hook)

## Steps:
1. **Find Active Session:**
   - Look in `.sessions/active/` for session file
   - If no active session: Error with suggestion

2. **Gather Current State:**
   - Parse session file for all sections
   - Get git state: branch, last commit, uncommitted files
   - Calculate progress percentage

3. **Update TL;DR Section:**
   - Regenerate summary from current progress and context
   - Format: "{summary}. Progress: {N}/{M}. Current: {step}. Blocker: {blocker or None}"

4. **Update Progress File:**
   - Regenerate `.sessions/journal/{id}/progress.md` with current state
   - Update all sections: TL;DR, Checklist, Key Context, Current Position

5. **Update Session Frontmatter:**
   - Update `updated_at` timestamp
   - Update `git_last_commit` if changed
   - If `--pre-compact`: Increment `compaction_count`

6. **Create Journal Entry (if --note):**
   - Create timestamped entry in `.sessions/journal/{id}/entries/`
   - Include note and current state snapshot

## Output Format:
```
💾 Checkpoint Saved: {shortname}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Progress: {N}/{M} ({pct}%)
Current: {current step}
Files modified: {count}
{Note: {note} (if provided)}

Session can be resumed with:
  /session-resume
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Pre-Compact Mode (--pre-compact):
Used by the PreCompact hook to save state before context compaction:
```
💾 Pre-Compaction Checkpoint: {shortname}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Compaction #{N}
Progress: {M}/{T} ({pct}%)

## Context Summary (for next context window)
{TL;DR section content}

## Current Position
{Current step with details}

## Key Context
{Key Context section - technical details that must survive}

## Next Action
{First item from Next Steps}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Progress File Update:
The progress.md file is completely regenerated with current state:
```markdown
---
session_id: {uuid}
shortname: {shortname}
last_updated: {NOW}
compaction_count: {N}
progress_pct: {calculated}
current_step: {step number}
total_steps: {total}
---

# Quick Resume: {shortname}

## TL;DR
{Current one-sentence summary}. Working on {current step}.
Next: {immediate next action}.

## Checklist
1. [x] Completed step 1
2. [x] Completed step 2
3. [ ] Current step      <- HERE
4. [ ] Remaining step
...

## Key Context
{Extracted from session Key Context section}

## Current Position
- File: {last modified file}
- Function: {current function if known}
- Last action: {last progress entry}

## Open Blockers
{Current blockers or "None"}

## Resume Commands
```bash
{Commands from session file}
```
```

## No Active Session Error:
```
❌ No active session to checkpoint.

To start a new session:
  /session-init --goal="description"

To resume an existing session:
  /session-resume
```
