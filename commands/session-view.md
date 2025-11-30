---
description: View detailed session status, progress, and history
argument-hint: [session-id] [--current] [--progress] [--decisions] [--discoveries]
---

Display detailed information about a session including progress, decisions, and discoveries.

Arguments: $ARGUMENTS

## Steps:
1. **Find Session:**
   - If `--current` or no args: Find active session in `.sessions/active/`
   - If `session-id` provided: Search by ID or shortname in active and archive
   - If no session found: Display "No active session" message with suggestion

2. **Parse Session File:**
   - Read YAML frontmatter for metadata
   - Parse markdown sections

3. **Display Session Details:**
   - Show based on flags (--progress, --decisions, --discoveries) or full view

## Output Format (Full View):
```
📋 Session: {shortname}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Status: {active|paused|completed}
Started: {created_at} ({relative time})
Updated: {updated_at} ({relative time})
Compactions: {compaction_count}
Branch: {git_branch}

## TL;DR
{TL;DR section content}

## Progress ({completed}/{total} steps)
{Progress section with checkboxes}

## Linked Tasks
- {task-shortname}: {status} ({priority}, {size})

## Current Blockers
{Blockers section or "None"}

## Next Steps
{Next Steps section}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Quick Actions:
  /session-log --progress "..."    Add progress
  /session-checkpoint              Save checkpoint
  /session-pause                   Pause session
  /session-close --complete        Complete session
```

## Output Format (--progress):
```
## Progress: {shortname}
{completed}/{total} steps ({percentage}%)

### Completed
{Completed items with timestamps}

### In Progress
{Current item}

### Remaining
{Remaining items}
```

## Output Format (--decisions):
```
## Decisions: {shortname}

### 1. {Decision Title} ({timestamp})
**Context**: {context}
**Options**: {options}
**Choice**: {choice}
**Rationale**: {rationale}

### 2. ...
```

## Output Format (--discoveries):
```
## Discoveries: {shortname}

1. **{Title}** ({timestamp}): {description}
2. **{Title}** ({timestamp}): {description}
...
```

## No Session Found:
```
No active session found.

To start a new session:
  /session-init --goal="description"
  /session-init --task=<task-id>

To resume a previous session:
  /session-resume --list
```

## List Mode (session-id = "list"):
If user runs `/session-view list`, show all sessions:
```
## Sessions

### Active
- {shortname}: {status}, {progress}%, updated {relative-time}

### Recent (Archived)
- {shortname}: completed {date}, {duration}
- ...
```
