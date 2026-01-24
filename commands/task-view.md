---
name: task-view
description: View detailed information about a specific task
argument-hint: <task-id> [--current] [--raw]
match: always
---

Display full details of a specific task using Claude Code's built-in task system.

Task ID or shortname: $1
Arguments: $ARGUMENTS

## Steps

1. **Determine which task to view**:
   - If `--current` flag: Use TaskList to find task with `status=in_progress` (prefer one with owner set)
   - If task ID provided: Use TaskGet with the numeric ID
   - If shortname provided: Use TaskList, then find task where `metadata.shortname` matches

2. **Use TaskGet tool** to retrieve full task details

3. **Derive display status**:
   - `status=completed` -> "completed"
   - `status=in_progress` -> "in_progress"
   - `status=pending` + non-empty `blockedBy` with incomplete tasks -> "blocked"
   - `status=pending` + `metadata.blocked_reason` set -> "blocked"
   - `status=pending` + empty/resolved `blockedBy` -> "ready"

4. **Display formatted output**

## Parameters
- `<task-id>`: Task ID (numeric) or shortname (partial match supported)
- `--current`: View current in_progress task
- `--raw`: Show raw JSON data from TaskGet

## Display Format
```
Task Details: #3 (fix-auth-bug)
---
ID: 3
Status: in_progress
Priority: P0 | Size: M
Owner: claude-code
Tags: [security, authentication]
Created: 2025-01-23T10:00:00Z
Updated: 2025-01-23T14:00:00Z

SUBJECT
Fix authentication bug in login flow

DESCRIPTION
Users are unable to login when their session expires. The JWT refresh
logic appears to have a race condition.

ACCEPTANCE CRITERIA
[x] Bug is reproduced in test environment
[x] Root cause identified
[ ] Fix implemented
[ ] Tests added
[ ] Documentation updated

DEPENDENCIES
Blocks: #4, #5
Blocked by: none
```

## Field Mappings
- ID: `id` from TaskGet
- Shortname: `metadata.shortname`
- Subject: `subject` field
- Description: `description` field
- Status: Derived (see above)
- Priority: `metadata.priority`
- Size: `metadata.size`
- Owner: `owner` field
- Tags: `metadata.tags` array
- Created/Updated: `metadata.created_at`, `metadata.updated_at`
- Acceptance Criteria: `metadata.acceptance_criteria` array
- Blocks: `blocks` array
- Blocked by: `blockedBy` array

## Quick Actions
After viewing, suggest available actions:
- Start working: `/task-status <id> in_progress`
- Block task: `/task-status <id> blocked --reason="..."`
- Complete task: `/task-complete <id>`
- Add dependency: `/task-deps add <id> --blocked-by=<other-id>`

## Error Handling
- If task not found: "Task not found. Use `/task-list` to see available tasks."
- If multiple matches for shortname: List all matches and ask user to specify by ID
