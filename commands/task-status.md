---
name: task-status
description: Update the status of a task
argument-hint: <task-id> <status> [--reason=...] [--assignee=...]
match: always
---

Update the status of a task (ready, in_progress, blocked, completed). Task ID: $1, New status: $2

Arguments: $ARGUMENTS

## Steps:
1. Find task by ID (partial match) or shortname
2. Show current status and metadata
3. Update to new status
4. Handle status-specific actions
5. Update `updated_at` timestamp

## Parameters:
- `<task-id>`: Task ID or shortname (partial match supported)
- `<status>`: New status (ready|in_progress|blocked|completed)
- `--reason`: Reason for status change (required for blocked)
- `--assignee`: Update assignee (for in_progress)

## Status Transitions:
```
ready → in_progress: Assign to user/agent
ready → blocked: Requires reason
in_progress → blocked: Requires reason
in_progress → completed: Check acceptance criteria
in_progress → ready: Unassign
blocked → ready: Clear blocker reason
blocked → in_progress: Assign and clear blocker
completed → ready: Reopen task (rare)
```

## Output:
```
📝 Status Updated: fix-auth-bug
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Previous: in_progress (assigned: agent-1)
New: blocked
Reason: Waiting for API documentation

Task: Fix authentication bug in login flow
Updated: 2025-01-01T14:30:00Z
```

## Special Handling:
- `blocked`: Add blocker reason to task description
- `in_progress`: Set assignee, warn if already assigned
- `completed`: Suggest using task-complete instead
- Multiple `in_progress`: Warn about WIP limits