- for comments, always use complete sentences ending with a period
- if you're unsure about the API of a Go package or struct, you can also use the `go doc` command to find more information

# Task Management System

## Overview
Each project has a lightweight task management system stored in `.tasks/` directory. Tasks are markdown files with YAML frontmatter containing metadata. This system enables systematic progress tracking and multi-agent collaboration.

## Task Structure
Tasks are stored in:
- `.tasks/active/` - Current and upcoming tasks
- `.tasks/archive/` - Completed tasks
- `.tasks/templates/` - Task templates

File naming: `{shortname}-{uuidv7}.md` (e.g., `fix-auth-bug-01234567-89ab-7cde.md`)

## Task Dependencies
Tasks can specify dependencies using:
- `blocked_by: [task-id, ...]` - Tasks that must complete before this one
- `blocks: [task-id, ...]` - Tasks that depend on this one

Dependencies are automatically checked when:
- Selecting next task with `/task-next`
- Updating task status
- Completing tasks (unblocks dependents)

## Available Commands
- `/task-add` - Create a new task interactively
- `/task-list` - List all tasks with filters
- `/task-next` - Pick the next task to work on
- `/task-view <id>` - View task details
- `/task-status <id> <status>` - Update task status
- `/task-complete <id>` - Mark task as complete and archive
- `/task-deps <action> <id>` - Manage task dependencies

## Task Workflow
1. **Creating Tasks**: Use `/task-add` or create markdown file directly
2. **Starting Work**: Use `/task-next` to select highest priority task
3. **During Work**: Update acceptance criteria checkboxes as you progress
4. **Completion**: Use `/task-complete` when all criteria are met
5. **Blocking**: Use `/task-status <id> blocked --reason "..."` if stuck

## Task Priorities
- P0: Critical blocker (drop everything)
- P1: High priority (do soon)
- P2: Medium priority (normal work)
- P3: Low priority (nice to have)

## Task Sizes
- XS: < 1 hour
- S: 1-4 hours
- M: 4-8 hours
- L: 1-3 days
- XL: 3+ days

## Automatic Task Management
When working on a codebase:
1. Check for existing tasks: `/task-list`
2. Pick up work: `/task-next`
3. Track progress by updating acceptance criteria checkboxes
4. Complete tasks: `/task-complete --current`
5. Create new tasks as you discover work: `/task-add`

## Multi-Agent Collaboration
- Tasks have an `assignee` field for tracking who's working on what
- Check for `in_progress` tasks before starting new work
- Use task dependencies to coordinate complex work
- Update task status when blocking/unblocking others

## Best Practices
- Break large tasks into smaller ones (prefer S/M over L/XL)
- Write clear acceptance criteria that can be verified
- Update tasks immediately when status changes
- Use tags for categorization (e.g., "bug", "feature", "docs")
- Check dependencies before starting work
- Keep task descriptions focused and actionable