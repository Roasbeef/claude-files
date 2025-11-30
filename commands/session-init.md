---
description: Initialize a new work session for execution tracking across compactions
argument-hint: [--task=<id>] [--tasks=<id1,id2>] [--name=<shortname>] [--goal="description"]
---

Initialize a new work session for tracking execution progress across context compactions and multiple work periods.

Arguments: $ARGUMENTS

## Pre-flight Checks:
1. Check if `.sessions/` directory structure exists, create if not:
   - `.sessions/active/` - Active session files
   - `.sessions/archive/` - Completed/closed sessions
   - `.sessions/journal/` - Per-session journal directories
2. Check for existing active session in `.sessions/active/`
   - If found: Ask user to close, pause, or continue existing session
   - If `--force` specified: Close existing session and create new

## Steps:
1. **Generate Session Identity:**
   - Generate session ID using UUIDv7: `npx uuid v7`
   - Create shortname from task title, --name parameter, or --goal
   - Format: `{shortname}-{uuid}.md`

2. **Task Linking (if specified):**
   - If `--task=<id>`: Validate task exists in `.tasks/active/`, extract metadata
   - If `--tasks=<id1,id2>`: Validate all tasks exist, first is primary
   - Copy task title, acceptance criteria, and description to session context

3. **Gather Session Context:**
   - Git branch: `git rev-parse --abbrev-ref HEAD`
   - Git last commit: `git rev-parse --short HEAD`
   - Working directory files (if task linked, use task's key files)

4. **Create Session File:**
   - Write to `.sessions/active/{shortname}-{uuid}.md`
   - Use template below with YAML frontmatter

5. **Create Journal Directory:**
   - Create `.sessions/journal/{uuid}/`
   - Create initial `progress.md` file

6. **Update Linked Tasks (if any):**
   - Set task status to `in_progress`
   - Add `session_id` to task frontmatter (optional)

## Session File Template:
```markdown
---
id: {uuid}
shortname: {shortname}
status: active
task_ids: []
created_at: {ISO-8601-timestamp}
updated_at: {ISO-8601-timestamp}
compaction_count: 0
git_branch: {branch}
git_last_commit: {commit-hash}
---

# Session: {Title from goal or task}

## TL;DR (Read This First)
{One paragraph summary of objective}

**Progress**: 0/N steps complete
**Current**: Starting - {first step}
**Blocker**: None

## Context
**Objective**: {From --goal or task description}

**Starting State**:
- Branch: {git_branch}
- Primary task: {task shortname if linked}

**Key Files**:
- List key files to work with

**Key Context** (survives compaction):
- Technical context that must survive compactions

## Progress
### Completed
(none yet)

### In Progress
- [ ] First step          <- CURRENT

### Remaining
- [ ] Remaining steps...

## Decisions
(none yet)

## Discoveries
(none yet)

## Blockers
- None currently

## Next Steps
1. {First action to take}

## Files Modified This Session
(none yet)

## Resume Commands
```bash
# Commands to run when resuming this session
git status
```
```

## Progress File Template (journal/{uuid}/progress.md):
```markdown
---
session_id: {uuid}
shortname: {shortname}
last_updated: {ISO-8601-timestamp}
compaction_count: 0
progress_pct: 0
current_step: 1
total_steps: N
---

# Quick Resume: {shortname}

## TL;DR
{One sentence summary}. Starting work on {objective}.

## Checklist
1. [ ] First step      <- HERE
2. [ ] Next step
...

## Key Context
- Key technical context for quick resume

## Current Position
- File: (not started)
- Function: (not started)
- Last action: Session initialized

## Open Blockers
None

## Resume Commands
```bash
git status
```
```

## Output:
```
📋 Session Initialized: {shortname}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
ID: {uuid}
Branch: {git_branch}

Objective: {goal or task title}

Linked Tasks:
  - {task-shortname} (P1, M, ready → in_progress)

Progress tracking initialized. Use these commands:
  /session-log --progress "message"   Add progress entry
  /session-log --decision "..."       Log a decision
  /session-checkpoint                 Save checkpoint
  /session-view                       View session details
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Interactive Mode (no arguments):
If no arguments provided, prompt interactively:
1. "What is the goal of this session?" (creates --goal)
2. "Link to existing task? (y/n)" - if yes, show task list
3. "Key files to work with?" (optional)
4. Create session with gathered information
