---
name: task-next
description: Pick and start working on the next highest priority task
argument-hint: [task-id] [--priority=P0-P3] [--size=XS-XL] [--tag=tag] [--force]
match: always
---

Select and begin work on the next appropriate task. If task-id is provided as $1, use that specific task. Otherwise, select based on the algorithm below.

Arguments: $ARGUMENTS

## Steps:
1. Load all tasks from `.tasks/active/`
2. Filter for `status: ready` tasks (not in_progress, blocked, or completed)
3. Check dependencies are met
4. Sort by priority and size
5. Present top candidate to user for confirmation
6. Update task status to `in_progress`
7. Set assignee to current agent/user
8. Display task details and acceptance criteria

## Selection Algorithm:
```
1. Exclude tasks with unmet dependencies (blocked_by list)
2. Exclude blocked or completed tasks
3. Prefer tasks already in_progress (resume work)
4. Then by priority: P0 > P1 > P2 > P3
5. Within same priority, prefer smaller sizes (quick wins)
6. Prefer tasks that unblock others (have items in blocks list)
7. Break ties by creation date (FIFO)
```

## Dependency Checking:
- Parse `blocked_by` field from task frontmatter
- For each blocking task ID:
  - Check if task exists
  - Check if status is `completed`
- Only consider task ready if all dependencies are completed

## Parameters:
- `--priority=<P0-P3>`: Focus on specific priority
- `--size=<XS-XL>`: Prefer specific size
- `--tag=<tag>`: Filter by tag
- `--force`: Skip confirmation prompt

## Output:
```
🎯 Selected Task: TASK-20250101-001
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Title: Fix critical authentication bug
Priority: P0 | Size: M | Status: ready → in_progress

Description:
[Full task description]

Acceptance Criteria:
- [ ] Bug is fixed
- [ ] Tests added
- [ ] Documentation updated

Ready to start? (y/n)
```

After confirmation:
1. Update task file status to `in_progress`
2. Set `assignee` field
3. Update `updated_at` timestamp
4. Create a git branch (optional): `task/TASK-YYYYMMDD-NNN`