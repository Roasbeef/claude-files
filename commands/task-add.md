---
name: task-add
description: Create a new task with metadata and acceptance criteria
argument-hint: [task details or empty for interactive mode]
match: always
---

Create a new task using Claude Code's built-in task system.

Arguments: $ARGUMENTS

## Steps

1. **Gather task information** (interactive if no arguments provided):
   - Subject: Task title in imperative form (e.g., "Fix auth bug", "Add unit tests")
   - Description: Detailed explanation of what needs to be done
   - Priority: P0 (critical), P1 (high), P2 (medium, default), P3 (low)
   - Size: XS (<1h), S (1-4h), M (4-8h, default), L (1-3d), XL (3d+)
   - Tags: Optional comma-separated list
   - Acceptance criteria: Optional list of completion requirements

2. **Generate derived fields**:
   - `shortname`: Generate from subject (e.g., "Fix auth bug" -> "fix-auth-bug")
   - `activeForm`: Convert subject to present continuous (e.g., "Fix auth bug" -> "Fixing auth bug")

3. **Use TaskCreate tool** with:
   - `subject`: The task title in imperative form
   - `description`: Include full description. If acceptance criteria provided, append as checklist
   - `activeForm`: Present continuous form for spinner display
   - `metadata`: Object containing:
     - `shortname`: Generated short identifier
     - `priority`: P0/P1/P2/P3
     - `size`: XS/S/M/L/XL
     - `tags`: Array of tag strings
     - `acceptance_criteria`: Array of `{text: string, completed: boolean}` objects
     - `created_at`: ISO 8601 timestamp
     - `updated_at`: ISO 8601 timestamp

4. **Display created task summary**

## Priority Levels
- P0: Critical blocker - must be addressed immediately
- P1: High priority - important work to complete soon
- P2: Medium priority - standard work (default)
- P3: Low priority - nice to have, can wait

## Size Estimates
- XS: Less than 1 hour
- S: 1-4 hours
- M: 4-8 hours (default)
- L: 1-3 days
- XL: 3+ days

## Examples

### With arguments
```
/task-add Fix authentication bug causing session timeout
```
Creates task with subject "Fix authentication bug causing session timeout", prompts for priority/size/etc.

### Interactive mode
```
/task-add
```
Prompts for all fields interactively.

## Output Format
```
Task Created: #3
Subject: Fix authentication bug causing session timeout
Shortname: fix-authentication-bug
Priority: P1 | Size: M
Status: pending (ready)

Description:
Users are experiencing session timeouts...

Acceptance Criteria:
- [ ] Bug is reproduced
- [ ] Root cause identified
- [ ] Fix implemented
- [ ] Tests added
```
