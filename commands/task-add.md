---
name: task-add
description: Create a new task with metadata and acceptance criteria
argument-hint: [task details or empty for interactive mode]
match: always
---

Create a new task in the project's `.tasks/` directory.

Arguments: $ARGUMENTS

## Steps:
1. Check if `.tasks/` directory exists, create if not
2. Generate task ID using UUIDv7: `npx uuid v7`
3. Create short name from title (e.g., "fix-auth-bug")
4. Create task markdown file with provided or prompted information
5. Save to `.tasks/active/` directory as `{shortname}-{uuid}.md`

## Task Template:
```markdown
---
id: 01234567-89ab-7cde-f012-456789abcdef
shortname: fix-auth-bug
title: Fix authentication bug in login flow
priority: P1
size: M
status: ready
tags: []
blocks: []  # Tasks that this task blocks (dependent tasks)
blocked_by: []  # Tasks that block this task (dependencies)
assignee:
created_at: 2025-01-01T00:00:00Z
updated_at: 2025-01-01T00:00:00Z
---

# Task: Brief task description

## Description
Detailed explanation of what needs to be done, why it's important, and any relevant context.

## Acceptance Criteria
- [ ] First acceptance criterion
- [ ] Second acceptance criterion
- [ ] All tests pass
- [ ] Documentation updated

## Technical Details
Any implementation notes, approaches to consider, or technical constraints.

## Dependencies
### Blocks (tasks waiting on this)
- List task IDs or shortnames that depend on this task

### Blocked By (waiting on these)
- List task IDs or shortnames that must complete first

### External Dependencies
- Non-task blockers (APIs, documentation, approvals, etc.)
```

## Priority Levels:
- P0: Critical blocker
- P1: High priority
- P2: Medium priority
- P3: Low priority

## Size Estimates:
- XS: < 1 hour
- S: 1-4 hours
- M: 4-8 hours
- L: 1-3 days
- XL: 3+ days

First check for parameters, if not provided, create template and work interactively with user to fill it out.