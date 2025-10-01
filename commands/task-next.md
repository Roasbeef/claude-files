---
name: task-next
description: Pick and start working on the next highest priority task
argument-hint: [task-id] [--priority=P0-P3] [--size=XS-XL] [--tag=tag] [--force]
match: always
allowed-tools: Glob, Read, Edit, Bash(ls:*)
---

Select and begin work on the next appropriate task. If task-id is provided as $1, use that specific task. Otherwise, select based on the algorithm below.

Arguments: $ARGUMENTS

## Pre-flight Checks:
1. **Verify task directory exists**: Check if `.tasks/active/` exists using Glob or Bash
   - If missing: Tell user "No task management system found. Create tasks with `/task-add` or initialize `.tasks/active/` directory"
   - STOP execution and return early

2. **Find task files**: Use Glob to find all `.md` files in `.tasks/active/`
   - If no files found: Tell user "No tasks found. Create your first task with `/task-add`"
   - STOP execution and return early

## Main Steps:
1. Load all tasks from `.tasks/active/` using Glob then Read for each file
2. **Check WIP limits**: Count tasks with status `in_progress`
   - If count >= 2: Warn user "You have {count} tasks in progress. Consider completing current work before starting new tasks. Use `--force` to override."
   - If `--force` not present: STOP execution and return early
3. Filter for `status: ready` tasks (not in_progress, blocked, or completed)
   - If no ready tasks: Report status summary (e.g., "Found 3 tasks: 2 in_progress, 1 blocked. No ready tasks available.")
   - STOP execution and return early
4. Check dependencies are met (see Dependency Checking below)
5. Sort by priority and size (see Selection Algorithm below)
6. Present top candidate to user for confirmation (skip if --force flag present)
7. Update task status to `in_progress` using Edit
8. Set assignee field (see Assignee Handling below)
9. Display task details and acceptance criteria

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
- Parse `blocked_by` field from task frontmatter (list of task IDs or shortnames)
- For each blocking task ID in the list:
  - Search `.tasks/active/` and `.tasks/archive/` for matching task (partial match supported)
  - Check if task exists (if not, warn and treat as unmet dependency)
  - Check if status is `completed` (task must be in archive or have completed status)
- Only consider task ready if ALL dependencies are completed
- If task has unmet dependencies, exclude from selection and show in status summary

## Assignee Handling:
- Set `assignee` field to "claude-code" or the current user's name if available
- If task already has an assignee and it's different: Ask user if they want to reassign
- Preserve assignee field format from other commands for consistency

## Parameters:
- `[task-id]`: Specific task ID or shortname to start (partial match supported, bypasses selection algorithm)
- `--current`: Resume current in_progress task (if one exists)
- `--priority=<P0-P3>`: Focus on specific priority
- `--size=<XS-XL>`: Prefer specific size
- `--tag=<tag>`: Filter by tag
- `--force`: Skip confirmation prompt

## Current Task Selection:
If `--current` flag is present:
1. Search all tasks in `.tasks/active/` for status `in_progress`
2. If no in_progress tasks: Tell user "No tasks currently in progress. Use `/task-list` to see available tasks"
3. If multiple in_progress tasks: List all and ask which to resume (or resume most recently updated)
4. Display task details and acceptance criteria progress

## Specific Task Selection:
If task-id is provided as first positional argument:
1. Search all tasks in `.tasks/active/` for matching ID or shortname (partial match supported)
2. If not found: Tell user "Task not found: {task-id}. Use `/task-list` to see available tasks"
3. If found but status is `completed`: Tell user "Task already completed: {task-id}"
4. If found but status is `blocked`: Show blocking tasks using task-deps logic and ask if user wants to proceed anyway
5. If found and ready: Proceed to step 5 of Main Steps (confirmation)

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

After confirmation (or if --force flag is present):
1. Update task file status to `in_progress` using Edit tool
2. Set `assignee` field to "claude-code" or current user
3. Update `updated_at` timestamp to current ISO 8601 datetime
4. **Display success message** showing:
   - Task ID and title
   - Updated status (ready → in_progress)
   - Priority, size, tags
   - Full description
   - Acceptance criteria checklist (show progress: X/Y completed)
5. **Begin work** on the task immediately

## Post-Action:
After successfully starting a task:
1. Check if any other tasks are blocked by dependencies that might be related
2. Display quick actions available:
   - View full details: `/task-view <shortname>`
   - Block task: `/task-status <shortname> blocked --reason="..."`
   - Complete task: `/task-complete <shortname>`
   - Check dependencies: `/task-deps check <shortname>`

## Important Error Prevention:
- **ALWAYS provide feedback** to the user about what happened or why nothing happened
- **NEVER exit silently** - if no action is taken, explain why (no tasks, all blocked, etc.)
- **ALWAYS use the Edit tool** to update the task file - don't just display what would be done
- If Edit fails, report the error to the user and suggest manual update
- If multiple steps fail, report all failures to help user diagnose the issue
