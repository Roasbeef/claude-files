---
name: task-view
description: View detailed information about a specific task
argument-hint: <task-id> [--current] [--raw]
match: always
---

Display full details of a specific task. Task ID or shortname: $1

Arguments: $ARGUMENTS

## Steps:
1. Find task by ID (partial match) or shortname
2. Load and parse the full markdown file
3. Display all metadata and content
4. Show acceptance criteria status
5. Show related tasks (dependencies and dependents)

## Parameters:
- `<task-id>`: Task ID or shortname (partial match supported)
- `--current`: View current in_progress task
- `--raw`: Show raw markdown content

## Output Format:
```
📋 Task Details: fix-auth-bug-01234567-89ab-7cde
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
ID: 01234567-89ab-7cde-f012-456789abcdef
Status: in_progress
Priority: P0 | Size: M
Assignee: agent-1
Tags: [security, authentication]
Created: 2025-01-01T10:00:00Z
Updated: 2025-01-01T14:00:00Z

TITLE
Fix authentication bug in login flow

DESCRIPTION
Users are unable to login when their session expires...

ACCEPTANCE CRITERIA
✅ Bug is reproduced in test environment
✅ Root cause identified
☐ Fix implemented
☐ Tests added
☐ Documentation updated

DEPENDENCIES
← Blocks: implement-2fa-0987654
→ Blocked by: none

TECHNICAL DETAILS
The issue appears to be in the JWT refresh logic...
```

## Quick Actions:
After viewing, offer quick actions:
- [S]tart working (if ready)
- [B]lock with reason
- [C]omplete task
- [E]dit task
- [A]ssign to someone