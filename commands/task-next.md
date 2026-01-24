---
name: task-next
description: Pick and start working on the next highest priority task
argument-hint: [task-id] [--priority=P0-P3] [--size=XS-XL] [--tag=tag] [--force]
match: always
---

Select and begin work on the next appropriate task using Claude Code's built-in task system.

Arguments: $ARGUMENTS

## Steps

1. **Use TaskList tool** to get all tasks

2. **Check WIP limits**:
   - Count tasks with `status=in_progress`
   - If count >= 2 and `--force` not present:
     - Warn: "You have N tasks in progress. Consider completing current work first."
     - List current in_progress tasks
     - STOP execution

3. **Filter for ready tasks**:
   - `status=pending`
   - `blockedBy` is empty OR all tasks in `blockedBy` have `status=completed`
   - `metadata.blocked_reason` is not set

4. **If specific task-id provided**:
   - Find by numeric ID or `metadata.shortname`
   - Check if ready (not blocked)
   - Skip to step 6

5. **Selection algorithm** (if no task-id):
   a. Prefer tasks already `in_progress` (resuming work)
   b. Then by priority: P0 > P1 > P2 > P3
   c. Within same priority, prefer smaller sizes (XS > S > M > L > XL)
   d. Prefer tasks that have items in `blocks` list (unblock others faster)
   e. Break ties by `metadata.created_at` (FIFO - oldest first)

6. **Use TaskUpdate tool** to start task:
   ```json
   {
     "taskId": "<id>",
     "status": "in_progress",
     "owner": "claude-code",
     "activeForm": "<present continuous form>",
     "metadata": {
       "updated_at": "<now>"
     }
   }
   ```

7. **Display task details and begin work**

## Parameters
- `[task-id]`: Specific task ID or shortname to start
- `--current`: Resume current in_progress task (skip selection)
- `--priority=<P0-P3>`: Filter candidates to specific priority
- `--size=<XS-XL>`: Filter candidates to specific size
- `--tag=<tag>`: Filter candidates by tag in `metadata.tags`
- `--force`: Skip WIP limit warning and confirmation

## Selection Algorithm Priority Order
1. Exclude blocked tasks (unresolved `blockedBy` or `blocked_reason` set)
2. Exclude completed tasks
3. Prefer resuming existing `in_progress` tasks
4. Sort by priority: P0 (critical) > P1 (high) > P2 (medium) > P3 (low)
5. Within priority, prefer smaller sizes (quick wins)
6. Prefer tasks that unblock others (have entries in `blocks`)
7. Break ties by creation date (FIFO)

## Dependency Checking

A task is ready when:
- `status=pending`
- `blockedBy` array is empty OR all task IDs in `blockedBy` have `status=completed`
- `metadata.blocked_reason` is null/empty

Use TaskGet for each blocking task ID to check its status.

## Output Format
```
Selected Task: #3 (fix-auth-bug)
---
Subject: Fix critical authentication bug
Priority: P0 | Size: M
Status: pending -> in_progress
Owner: claude-code

Description:
Users are unable to login when their session expires. The JWT refresh
logic appears to have a race condition.

Acceptance Criteria:
[ ] Bug is fixed
[ ] Tests added
[ ] Documentation updated

Quick Actions:
- Block task: /task-status 3 blocked --reason="..."
- Complete task: /task-complete 3
- View dependencies: /task-deps check 3
```

## Current Task Handling

If `--current` flag:
1. Use TaskList to find tasks with `status=in_progress`
2. If none: "No tasks currently in progress. Use `/task-list` to see available tasks."
3. If multiple: List all and ask which to resume, or pick most recently updated
4. Display task details

## No Ready Tasks

If no tasks pass the ready filter:
```
No Ready Tasks
---
Summary: 5 tasks total
  In Progress: 2
  Blocked: 2
  Completed: 1

Blocked Tasks:
  #4 deploy-feature - Blocked by: #3 fix-auth-bug (in_progress)
  #5 update-docs - Reason: Waiting for API documentation

Suggestion: Complete current tasks or resolve blockers first.
```

## Error Handling
- No tasks found: "No tasks found. Create your first task with `/task-add`"
- Task not found (specific ID): "Task not found. Use `/task-list` to see available tasks."
- Task already completed: "Task #N is already completed."
- Task is blocked: Show blocking tasks/reason, ask if user wants to proceed anyway

## Important
- **ALWAYS provide feedback** about what happened or why no task was selected
- **NEVER exit silently**
- After starting a task, begin work on it immediately
