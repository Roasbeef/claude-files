---
name: task-list
description: List all tasks with optional filters
argument-hint: [--status=...] [--priority=P0-P3] [--assignee=...] [--size=XS-XL] [--tag=...] [--all]
match: always
allowed-tools: Bash, Glob, Read
---

Display all tasks from the project's `.tasks/` directory with filtering options.

Arguments: $ARGUMENTS

## Pre-flight Check:
1. **Check for task directory**: Use Bash to check if `.tasks/active/` exists
   - If missing: Tell user "No task management system found. Create tasks with `/task-add` or initialize `.tasks/active/` directory"
   - STOP execution and return early

## Main Steps:
1. **Execute list-tasks.sh script**: Use Bash to run `~/.claude/scripts/list-tasks.sh` from current directory
2. **Display the output**: Show the script output to the user
3. **Handle errors**: If script fails, fall back to manual task listing (see Fallback Method below)

## Bash Command:
```bash
~/.claude/scripts/list-tasks.sh
```

## Fallback Method:
If the script is not available or fails, manually list tasks:
1. Use Glob to find all `.md` files in `.tasks/active/`
2. For each file, use Read to extract YAML frontmatter
3. Parse fields: id, shortname, title, priority, size, status, assignee, blocked_by
4. Sort by: status (in_progress first), then priority (P0 > P1 > P2 > P3), then size
5. Display in the format shown below

## Display Format:
```
📋 Active Tasks in [project-name]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔄 [P0] shortname                    [L] [in_progress]
   Full task title here

✅ [P1] another-task                 [M] [ready]
   Another task title

🔒 [P2] blocked-task                 [S] [blocked]
   Blocked task title
   ⚠️  Blocked by: [dependency-id]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Summary: X active tasks | Y ready | Z in progress
Legend: ✅ Ready | 🔄 In Progress | 🔒 Blocked
```

## Filter Options (Future Enhancement):
The $ARGUMENTS may contain filters to apply:
- `--status=<status>`: Only show tasks with matching status
- `--priority=<P0-P3>`: Only show tasks with matching priority
- `--assignee=<name>`: Only show tasks assigned to name
- `--size=<XS-XL>`: Only show tasks of specified size
- `--tag=<tag>`: Only show tasks with matching tag
- `--all`: Include archived tasks from `.tasks/archive/`

Note: Filtering is currently handled by the script. If implementing manually, filter the task list before displaying.

## Important:
- **ALWAYS provide output** - never exit silently
- If no tasks exist, tell the user "No tasks found. Create your first task with `/task-add`"
- If script fails, report the error and try the fallback method
- Display the full task list, don't just say you're running it
