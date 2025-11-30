---
description: Resume an existing work session with full context restoration
argument-hint: [session-id] [--list] [--last]
---

Resume a work session with full context restoration for continuity across compactions.

Arguments: $ARGUMENTS

## Steps:
1. **Find Session to Resume:**
   - If `--list`: Display available sessions and let user choose
   - If `--last`: Resume most recently updated session
   - If `session-id` provided: Find by ID or shortname
   - If no args: Resume active session, or show list if none active

2. **Load Context:**
   - Read `.sessions/journal/{id}/progress.md` first (quick context)
   - Read full session file from `.sessions/active/` or `.sessions/archive/`
   - Parse git state for current branch and uncommitted changes

3. **Run Resume Checks:**
   - Execute "Resume Commands" from progress.md (e.g., `git status`, test commands)
   - Check for conflicts between session branch and current branch
   - Verify linked task files still exist

4. **Update Session State:**
   - If session was paused: Set status to `active`
   - Increment `compaction_count` if resuming after compaction
   - Update `updated_at` timestamp
   - Update `git_branch` and `git_last_commit` if changed

5. **Display Context Briefing**

## Output Format:
```
📋 Resuming Session: {shortname}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Status: {paused → active | active (after {N} compaction(s))}
Started: {created_at} ({relative time})

## Where You Left Off
{Extract from TL;DR section}

## Key Context
{Key Context section from session file}

## Progress ({completed}/{total})
{Progress checklist with <- CURRENT marker}

## Open Blockers
{Blockers or "None"}

## Next Steps
{Next Steps section}

## Git State
Branch: {current_branch} {✓ matches | ⚠ differs from session branch}
Last commit: {commit_hash} - {commit_message}
Uncommitted: {N} files

## Resume Checks
{Output of resume commands if any}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Ready to continue. What would you like to work on?
```

## List Mode (--list):
```
## Available Sessions

### Active
📋 {shortname} - {goal summary}
   Progress: {N}/{M} ({pct}%) | Updated: {relative-time}
   Branch: {branch} | Tasks: {task-ids}

### Paused
⏸️  {shortname} - {goal summary}
   Progress: {N}/{M} ({pct}%) | Paused: {relative-time}
   Reason: {pause reason if any}

### Recent (Archived)
✅ {shortname} - completed {date}
✅ {shortname} - completed {date}
...

Enter session ID or shortname to resume, or 'q' to cancel:
```

## Branch Mismatch Warning:
If current git branch differs from session's recorded branch:
```
⚠️  Branch Mismatch
Session branch: {session_branch}
Current branch: {current_branch}

Options:
1. Continue on current branch (session branch will be updated)
2. Switch to session branch: git checkout {session_branch}
3. Cancel resume

Which option? [1/2/3]:
```

## No Sessions Found:
```
No sessions available to resume.

To start a new session:
  /session-init --goal="description"
  /session-init --task=<task-id>
```

## After Compaction Detection:
If this appears to be a resume after context compaction (detected via session startup hook or user indication):
```
📋 Resuming After Compaction
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Session: {shortname}
This is compaction #{N} for this session.

## TL;DR (Your Context)
{TL;DR section - this is what you were doing}

## What Was Completed
{Completed progress items}

## Current Position
{Current step with <- CURRENT marker}

## Key Context (Technical Details)
{Key Context section}

## Immediate Next Action
{First item from Next Steps}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```
