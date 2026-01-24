---
name: task-complete
description: Mark a task as completed
argument-hint: [task-id] [--current] [--force]
match: always
---

Mark a task as completed using Claude Code's built-in task system.

Task ID (if provided): $1
Arguments: $ARGUMENTS

## Steps

1. **Find target task**:
   - If `--current` flag: Use TaskList to find task with `status=in_progress`
   - If task-id provided: Use TaskGet with numeric ID, or TaskList to find by `metadata.shortname`

2. **Verify acceptance criteria** (unless `--force` flag):
   - Check `metadata.acceptance_criteria` array
   - Count completed vs total items
   - If any incomplete: Display list and ask for confirmation

3. **Use TaskUpdate tool**:
   ```json
   {
     "taskId": "<id>",
     "status": "completed",
     "metadata": {
       "updated_at": "<now>"
     }
   }
   ```

4. **Find newly unblocked tasks**:
   - Use TaskList to get all tasks
   - Find tasks where `blockedBy` contains this task's ID
   - For each: Check if ALL their blockers are now completed
   - Report which tasks are now unblocked/ready

5. **Display completion summary**

## Parameters
- `[task-id]`: Task ID (numeric) or shortname
- `--current`: Complete current in_progress task
- `--force`: Skip acceptance criteria check

## Acceptance Criteria Verification

If `metadata.acceptance_criteria` exists:
```json
[
  {"text": "Bug is fixed", "completed": true},
  {"text": "Tests added", "completed": false}
]
```

Display:
```
Acceptance Criteria: 1/2 completed
[x] Bug is fixed
[ ] Tests added

Not all criteria are met. Complete anyway? (y/n)
```

If `--force` is present, skip this check.

## Output Format
```
Task Completed: #3 (fix-auth-bug)
---
Subject: Fix authentication bug in login flow
Status: in_progress -> completed
Acceptance Criteria: 5/5 completed

Unblocked Tasks:
  #4 implement-2fa (now ready)
  #5 update-docs (now ready)

Suggested: Run `/task-next` to pick up next task
```

## Finding Unblocked Tasks

After completion:
1. Use TaskList to get all tasks
2. For each task with this task's ID in `blockedBy`:
   - Check if task status is `pending`
   - Check if ALL tasks in `blockedBy` are now `completed`
   - If yes: Task is now ready (unblocked)
3. Display list of newly unblocked tasks

## Note on Archival

The built-in task system does not archive completed tasks. Completed tasks remain in the task list with `status=completed`. Use `/task-list --status=pending` or `/task-list --status=in_progress` to filter out completed tasks.

## Error Handling
- Task not found: "Task not found. Use `/task-list` to see available tasks."
- No in_progress tasks (with --current): "No tasks currently in progress."
- Task already completed: "Task #N is already completed."

## Important
- **ALWAYS provide feedback** about what happened
- **NEVER exit silently**
- If TaskUpdate fails, report the error
