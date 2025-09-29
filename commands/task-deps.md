---
name: task-deps
description: Manage task dependencies and view dependency graph
match: always
---

Manage and visualize task dependencies.

## Steps:
1. Load all tasks from `.tasks/active/`
2. Build dependency graph from blocks/blocked_by fields
3. Execute requested action (add, remove, check, visualize)

## Parameters:
- `<action>`: add|remove|check|graph
- `<task-id>`: Task ID or shortname
- `--blocks=<id>`: Task that will be blocked by this task
- `--blocked-by=<id>`: Task that blocks this task
- `--recursive`: Show full dependency chain

## Actions:

### Add Dependency
```bash
/task-deps add fix-auth --blocked-by=update-api
```
Updates fix-auth to wait for update-api to complete.

### Remove Dependency
```bash
/task-deps remove fix-auth --blocked-by=update-api
```

### Check Dependencies
```bash
/task-deps check fix-auth
```
Output:
```
📊 Dependencies for: fix-auth-01234567
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Blocked By (must complete first):
  ✅ update-api-0987654 [completed]
  ❌ review-security-0456789 [in_progress]

Blocks (waiting on this task):
  ⏸️ deploy-prod-0234567 [ready]
  ⏸️ update-docs-0345678 [ready]

Status: BLOCKED (1 dependency incomplete)
```

### Dependency Graph
```bash
/task-deps graph
```
Output:
```
📊 Task Dependency Graph
━━━━━━━━━━━━━━━━━━━━━━━━
update-api ──┬──> fix-auth ──┬──> deploy-prod
             │                └──> update-docs
review-security ─┘

Legend:
→ Dependencies flow left to right
[*] In progress
[✓] Completed
[!] Blocked
```

## Circular Dependency Detection:
- Automatically detect circular dependencies
- Warn when adding a dependency would create a cycle
- Show the cycle path for debugging

## Batch Operations:
```bash
/task-deps add implement-feature --blocks="test-feature,document-feature,deploy-feature"
```

## Dependency Resolution Order:
When all dependencies are met, suggests optimal task order:
```
Suggested execution order:
1. update-api (no dependencies)
2. review-security (no dependencies)
3. fix-auth (depends on 1,2)
4. deploy-prod (depends on 3)
5. update-docs (depends on 3)
```