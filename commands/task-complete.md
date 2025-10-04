---
name: task-complete
description: Mark a task as completed and archive it
argument-hint: [task-id] [--current] [--no-archive] [--force]
match: always
allowed-tools: Glob, Read, Edit, Bash(ls:*,mv:*)
---

Mark a task as completed and optionally archive it. Task ID (if provided): $1

Arguments: $ARGUMENTS

## Pre-flight Checks:
1. **Verify task directory exists**: Check if `.tasks/active/` exists using Glob or Bash
   - If missing: Tell user "No task management system found in this project"
   - STOP execution and return early

2. **Find the target task**:
   - If `--current` flag: Search `.tasks/active/` for tasks with `status: in_progress`
   - If task-id provided: Search `.tasks/active/` for matching ID or shortname (partial match)
   - Use Glob to list files, then Read to parse frontmatter
   - If no task found: Tell user exactly what was searched and why nothing matched
   - STOP execution and return early if no match

## Main Steps:
1. **Load task file**: Use Read tool to load the full task markdown file
   - Parse YAML frontmatter for metadata
   - Parse markdown body for acceptance criteria checkboxes

2. **Verify acceptance criteria** (unless `--force` flag present):
   - Search markdown body for checkbox patterns: `- [x]`, `- [X]`, `- [ ]`
   - Count completed vs total checkboxes
   - If any unchecked: Display list of incomplete criteria and ask for confirmation
   - If user declines: STOP execution and return early

3. **Update task metadata** using Edit tool:
   - Change `status: <old-status>` to `status: completed`
   - Update `updated_at:` to current ISO 8601 timestamp
   - Verify Edit succeeded before proceeding

4. **Archive task** (unless `--no-archive` flag present):
   - Create `.tasks/archive/` directory if it doesn't exist (use Bash mkdir -p)
   - Move file from `.tasks/active/` to `.tasks/archive/` using Bash mv command
   - Verify move succeeded

5. **Find dependent tasks**:
   - Search all tasks in `.tasks/active/` for tasks that have this task's ID in their `blocked_by` list
   - For each dependent task found: Display notification that it's now unblocked

## Parameters:
- `[task-id]`: Task ID or shortname (partial match supported)
- `--current`: Complete current in_progress task
- `--no-archive`: Keep in active directory
- `--force`: Skip acceptance criteria check

## Acceptance Criteria Verification:
Parse the markdown body (after frontmatter) for checkboxes:
- `- [x]` or `- [X]` = completed
- `- [ ]` = not completed
- Look for patterns under "## Acceptance Criteria" section
- Count total checkboxes and completed checkboxes
- If not all checked and `--force` not present, ask user for confirmation

## Current Task Selection:
If `--current` flag is present:
1. Search all tasks in `.tasks/active/` for `status: in_progress`
2. If no in_progress tasks: Tell user "No tasks currently in progress. Use `/task-list` to see available tasks"
   - STOP execution and return early
3. If multiple in_progress tasks: List all and ask which to complete
4. Proceed with selected task

## Specific Task Selection:
If task-id provided as first positional argument:
1. Search all tasks in `.tasks/active/` for matching ID or shortname (partial match supported)
2. If not found: Tell user "Task not found: {task-id}. Use `/task-list` to see available tasks"
   - STOP execution and return early
3. If found and already completed: Tell user "Task already completed: {task-id}"
   - STOP execution and return early
4. Proceed with found task

## Output:
After successfully completing a task, display:
```
✅ Task Completed: fix-auth-bug-01234567
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Title: Fix authentication bug
Status: in_progress → completed
All acceptance criteria met: ✓ (5/5)

Archived to: .tasks/archive/fix-auth-bug-01234567-89ab-7cde.md

Unblocked tasks:
  → implement-2fa-feature
```

## Post-completion:
1. Search `.tasks/active/` for tasks with this task's ID in their `blocked_by` list
2. For each unblocked task: Display notification with task title and ID
3. Suggest running `/task-next` to pick up the next task
4. If no more ready tasks: Display summary of remaining tasks by status

## Important Error Prevention:
- **ALWAYS provide feedback** to the user about what happened or why nothing happened
- **NEVER exit silently** - if no action is taken, explain why (no task found, already completed, etc.)
- **ALWAYS use the Edit tool** to update the task file - don't just display what would be done
- **ALWAYS use Bash mv** to archive the file - don't just describe the move operation
- If Edit or mv fails, report the error to the user and suggest manual completion
- If multiple steps fail, report all failures to help user diagnose the issue
- **Before each major step**, verify prerequisites are met and STOP early if not