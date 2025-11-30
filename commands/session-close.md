---
description: Close a session (complete or abandon) and archive it
argument-hint: --complete|--abandon [--reason="..."] [--force]
---

Close the current session, either marking it as complete or abandoned, and archive it.

Arguments: $ARGUMENTS

## Close Modes:

### --complete
Mark session as successfully completed and archive:
- Verifies acceptance criteria are met (if linked to task)
- Updates linked task status to `completed`
- Moves session file to `.sessions/archive/`
- Generates completion summary

### --abandon [--reason="..."]
Abandon session without completing:
- Requires `--reason` explanation (or prompts interactively)
- Updates linked task status to `ready` (releases back to pool)
- Moves session file to `.sessions/archive/`
- Records abandonment reason

## Steps:
1. **Find Active Session:**
   - Look in `.sessions/active/` for session file
   - If no active session: Error with suggestion

2. **Validate Close Mode:**
   - Require `--complete` or `--abandon` flag
   - If neither: Ask user which mode

3. **For --complete:**
   - Check progress: Are all steps marked complete?
   - Check linked tasks: Are acceptance criteria met?
   - If not all complete and no `--force`: Warn and ask for confirmation
   - Generate completion summary

4. **For --abandon:**
   - Require or prompt for reason
   - Record abandonment in session file

5. **Update Session File:**
   - Set `status: completed` or `status: abandoned`
   - Add `closed_at: {timestamp}`
   - Add `close_reason: {reason}` for abandon

6. **Handle Linked Tasks:**
   - For --complete: Mark linked tasks as `completed`, move to archive
   - For --abandon: Set linked tasks to `ready`

7. **Archive Session:**
   - Move from `.sessions/active/` to `.sessions/archive/`
   - Keep journal directory for reference

8. **Display Summary**

## Output Format (--complete):
```
✅ Session Completed: {shortname}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Duration: {start} → {end} ({duration})
Compactions: {N}
Progress: {completed}/{total} steps

## Summary
{Generated summary of what was accomplished}

## Decisions Made
{Count} decisions recorded

## Discoveries
{Count} discoveries logged

## Files Modified
{List of files with line counts}

## Linked Tasks
- {task-shortname}: completed ✅

Session archived to: .sessions/archive/{filename}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Output Format (--abandon):
```
🚫 Session Abandoned: {shortname}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Duration: {start} → {end} ({duration})
Progress: {completed}/{total} steps ({pct}%)
Reason: {reason}

## Work Completed Before Abandonment
{Completed progress items}

## Remaining Work
{Remaining progress items}

## Linked Tasks
- {task-shortname}: released to ready (was in_progress)

Session archived to: .sessions/archive/{filename}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Incomplete Warning (--complete without all steps done):
```
⚠️  Session Not Fully Complete
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Progress: {completed}/{total} steps ({pct}%)

## Incomplete Steps:
- [ ] {incomplete step 1}
- [ ] {incomplete step 2}

Options:
1. Complete anyway (use --force)
2. Continue working
3. Abandon session instead

What would you like to do? [1/2/3]:
```

## Session File Updates (Complete):
```yaml
---
id: {uuid}
shortname: {shortname}
status: completed                        # Changed
closed_at: 2025-01-15T18:00:00Z          # Added
duration_hours: 8.5                       # Added (calculated)
# ... rest of frontmatter
---
```

## Session File Updates (Abandon):
```yaml
---
id: {uuid}
shortname: {shortname}
status: abandoned                        # Changed
closed_at: 2025-01-15T18:00:00Z          # Added
close_reason: "Requirements changed, new approach needed"  # Added
# ... rest of frontmatter
---
```

## No Mode Specified:
```
How would you like to close this session?

Session: {shortname}
Progress: {N}/{M} ({pct}%)

Options:
1. Complete - Mark as successfully finished
2. Abandon - Close without completing (will prompt for reason)
3. Cancel - Keep session active

Choice [1/2/3]:
```

## No Active Session Error:
```
❌ No active session to close.

To view sessions:
  /session-view list

To start a new session:
  /session-init
```
