---
name: task-list
description: List all tasks with optional filters
match: always
---

List tasks from the project's `.tasks/` directory.

## Steps:
1. Check for `.tasks/active/` and `.tasks/archive/` directories
2. Parse all task markdown files (extract YAML frontmatter)
3. Apply any filters from parameters
4. Display in priority/status order

## Display Format:
```
📋 Active Tasks:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[P0] TASK-20250101-001: Critical bug fix [L] [in_progress] @agent-1
[P1] TASK-20250101-002: Add new feature [M] [ready]
[P2] TASK-20250101-003: Refactor module [S] [ready]
```

## Filter Options:
- `--status=<status>`: Filter by status
- `--priority=<P0-P3>`: Filter by priority
- `--assignee=<name>`: Filter by assignee
- `--size=<XS-XL>`: Filter by size
- `--tag=<tag>`: Filter by tag
- `--all`: Include archived tasks

## Sorting:
Default sort order:
1. Status (in_progress > ready > blocked > completed)
2. Priority (P0 > P1 > P2 > P3)
3. Size (smaller first for same priority)
4. Created date (older first)