---
name: task-complete
description: Mark a task as completed and archive it
match: always
---

Mark a task as completed and optionally archive it.

## Steps:
1. Find task by ID (partial match supported) or current in_progress task
2. Verify all acceptance criteria are checked
3. Update status to `completed`
4. Update `updated_at` timestamp
5. Optionally move to `.tasks/archive/`
6. Create completion summary

## Parameters:
- `<task-id>`: Task ID (partial match supported)
- `--current`: Complete current in_progress task
- `--no-archive`: Keep in active directory
- `--force`: Skip acceptance criteria check

## Acceptance Criteria Verification:
Parse the markdown for checkboxes:
- `[x]` or `[X]` = completed
- `[ ]` = not completed

Warn if any criteria unchecked (unless --force).

## Output:
```
✅ Task Completed: 01234567-89ab-7cde-f012-456789abcdef
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Title: Fix authentication bug
Duration: 4 hours (started 2025-01-01 10:00)
All acceptance criteria met: ✓

Archived to: .tasks/archive/01234567-89ab-7cde-f012-456789abcdef.md
```

## Post-completion:
1. Check for dependent tasks that are now unblocked
2. Suggest next task to work on
3. Update any project metrics/burndown (if tracked)