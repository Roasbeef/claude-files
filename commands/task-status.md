---
name: task-status
description: Update the status of a task
argument-hint: <task-id> <status> [--reason=...] [--owner=...]
match: always
---

Update the status of a task using Claude Code's built-in task system.

Task ID: $1, New status: $2
Arguments: $ARGUMENTS

## Steps

1. **Find the task**:
   - If numeric ID: Use TaskGet directly
   - If shortname: Use TaskList, find task where `metadata.shortname` matches

2. **Show current status** before updating

3. **Use TaskUpdate tool** with appropriate fields based on status

4. **Display change summary**

## Parameters
- `<task-id>`: Task ID (numeric) or shortname
- `<status>`: New status (ready|in_progress|blocked|completed)
- `--reason=<text>`: Reason for status change (required for blocked)
- `--owner=<name>`: Set owner (for in_progress)

## Status Mapping to Built-in System

| User Status | Built-in Status | Additional Actions |
|-------------|-----------------|-------------------|
| `ready` | `pending` | Clear blockedBy, clear metadata.blocked_reason |
| `in_progress` | `in_progress` | Set owner field |
| `blocked` | `pending` | Set metadata.blocked_reason (requires --reason) |
| `completed` | `completed` | Suggest using /task-complete instead |

## TaskUpdate Fields by Status

### ready
```json
{
  "taskId": "<id>",
  "status": "pending",
  "metadata": {
    "blocked_reason": null,
    "updated_at": "<now>"
  }
}
```
Note: Cannot clear blockedBy directly - those are task dependencies.

### in_progress
```json
{
  "taskId": "<id>",
  "status": "in_progress",
  "owner": "<owner-name>",
  "metadata": {
    "blocked_reason": null,
    "updated_at": "<now>"
  }
}
```

### blocked
```json
{
  "taskId": "<id>",
  "status": "pending",
  "metadata": {
    "blocked_reason": "<reason from --reason flag>",
    "updated_at": "<now>"
  }
}
```

### completed
Suggest using `/task-complete <id>` instead for proper acceptance criteria checking.
If user insists:
```json
{
  "taskId": "<id>",
  "status": "completed",
  "metadata": {
    "updated_at": "<now>"
  }
}
```

## Status Transitions
```
pending (ready) -> in_progress: Set owner
pending (ready) -> pending (blocked): Set blocked_reason
in_progress -> pending (blocked): Set blocked_reason
in_progress -> completed: Check acceptance criteria first
in_progress -> pending (ready): Clear owner
pending (blocked) -> pending (ready): Clear blocked_reason
pending (blocked) -> in_progress: Clear blocked_reason, set owner
completed -> pending: Reopen task (rare)
```

## Output Format
```
Status Updated: #3 (fix-auth-bug)
---
Previous: in_progress (@agent-1)
New: blocked
Reason: Waiting for API documentation

Task: Fix authentication bug in login flow
Updated: 2025-01-23T14:30:00Z
```

## Special Handling
- `blocked`: Requires --reason flag; store in metadata.blocked_reason
- `in_progress`: Set owner; warn if task already has different owner
- `completed`: Recommend /task-complete for proper workflow
- Multiple `in_progress`: Warn about WIP limits (>= 2 tasks)

## Error Handling
- Task not found: "Task not found. Use `/task-list` to see available tasks."
- Missing --reason for blocked: "The `blocked` status requires a reason. Use `--reason=\"...\"`"
