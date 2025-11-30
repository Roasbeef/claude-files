---
description: Pause the current session with automatic checkpoint
argument-hint: [--note="reason"] [--switch-to=<session-id>]
---

Pause the current active session with an automatic checkpoint for later resumption.

Arguments: $ARGUMENTS

## When to Use:
- Switching to a different task/session
- Taking a break from work
- Need to context switch to urgent work
- End of work day

## Steps:
1. **Find Active Session:**
   - Look in `.sessions/active/` for session file
   - If no active session: Error with suggestion

2. **Create Checkpoint:**
   - Run checkpoint logic (same as `/session-checkpoint`)
   - Save all current progress and context

3. **Update Session Status:**
   - Set `status: paused` in frontmatter
   - Add `paused_at: {timestamp}` to frontmatter
   - If `--note` provided: Add `pause_reason: {note}`

4. **Update Progress File:**
   - Add pause indicator to TL;DR
   - Note when session was paused

5. **Handle Task Status:**
   - Update linked task status to `ready` (release back to pool)
   - Or keep as `in_progress` if `--keep-task-status` specified

6. **Switch to Another Session (if --switch-to):**
   - Run `/session-resume {session-id}` logic

## Output Format:
```
⏸️  Session Paused: {shortname}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Progress: {N}/{M} ({pct}%)
Current step: {current step}
{Reason: {note} (if provided)}

Checkpoint saved. Resume anytime with:
  /session-resume {shortname}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## With --switch-to:
```
⏸️  Session Paused: {shortname}
Progress: {N}/{M} ({pct}%)

📋 Switching to: {other-shortname}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
{session-resume output for new session}
```

## Session File Updates:
```yaml
---
id: {uuid}
shortname: {shortname}
status: paused                          # Changed from active
paused_at: 2025-01-15T16:00:00Z         # Added
pause_reason: "Switching to urgent bug" # Added if --note provided
# ... rest of frontmatter
---
```

## Progress File Updates:
Add pause indicator:
```markdown
# Quick Resume: {shortname}

## TL;DR
{summary}. **PAUSED** at step {N}/{M}. Reason: {note or "No reason given"}.

## Status
⏸️ Paused at: {timestamp}
Reason: {note or "Not specified"}
```

## Linked Task Handling:
By default, pausing a session releases linked tasks back to `ready` status:
```
Linked task {task-shortname}: in_progress → ready
```

Use `--keep-task-status` to prevent this:
```
/session-pause --keep-task-status
```

## No Active Session Error:
```
❌ No active session to pause.

To view available sessions:
  /session-view list

To resume a session:
  /session-resume
```
