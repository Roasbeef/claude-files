---
name: task-list
description: List all tasks with optional filters
argument-hint: [--status=...] [--priority=P0-P3] [--owner=...] [--size=XS-XL] [--tag=...]
match: always
---

Display all tasks using Claude Code's built-in task system.

Arguments: $ARGUMENTS

## Steps

1. **Use TaskList tool** to retrieve all tasks

2. **Apply filters** from arguments:
   - `--status=<pending|in_progress|completed>`: Filter by status
   - `--priority=<P0-P3>`: Filter by metadata.priority
   - `--owner=<name>`: Filter by owner field
   - `--size=<XS-XL>`: Filter by metadata.size
   - `--tag=<tag>`: Filter by metadata.tags array

3. **Derive display status** for each task:
   - `status=completed` -> "completed"
   - `status=in_progress` -> "in_progress"
   - `status=pending` + non-empty `blockedBy` with incomplete tasks -> "blocked"
   - `status=pending` + `metadata.blocked_reason` set -> "blocked"
   - `status=pending` + empty/resolved `blockedBy` + no `blocked_reason` -> "ready"

4. **Sort tasks**:
   - Status: in_progress first, then ready, then blocked, then completed
   - Within same status: by priority (P0 > P1 > P2 > P3)
   - Within same priority: by size (smaller first: XS > S > M > L > XL)

5. **Display formatted output**

## Display Format
```
Active Tasks
---
[P0] #3 fix-auth-bug               [M] [in_progress] @claude-code
     Fix authentication bug in login flow

[P1] #5 add-tests                  [S] [ready]
     Add unit tests for auth module

[P2] #7 deploy-feature             [L] [blocked]
     Deploy new feature to production
     Blocked by: #3 fix-auth-bug (in_progress)

---
Summary: 3 tasks | 1 ready | 1 in progress | 1 blocked | 0 completed
Legend: ready | in_progress | blocked | completed
```

## Field Mappings
- Task ID: `id` from TaskList
- Shortname: `metadata.shortname` (display alongside ID)
- Priority: `metadata.priority` (P0-P3)
- Size: `metadata.size` (XS-XL)
- Status: Derived from `status`, `blockedBy`, and `metadata.blocked_reason`
- Owner: `owner` field
- Subject: `subject` field

## Empty State
If no tasks exist:
```
No tasks found. Create your first task with `/task-add`
```

## Important
- **ALWAYS provide output** - never exit silently
- Display the full formatted list, don't just announce you're fetching it
- If filtering results in empty list, show: "No tasks match the specified filters"
