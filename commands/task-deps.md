---
name: task-deps
description: Manage task dependencies and view dependency graph
argument-hint: <action> [task-id] [--blocks=id] [--blocked-by=id]
match: always
---

Manage and visualize task dependencies using Claude Code's built-in task system.

Action: $1, Task ID: $2
Arguments: $ARGUMENTS

## Actions

### add
Add a dependency relationship between tasks.

```
/task-deps add 3 --blocked-by=1
```

**Steps**:
1. Use TaskGet to verify both tasks exist
2. Use TaskUpdate on task 3 with `addBlockedBy: ["1"]`
3. Use TaskUpdate on task 1 with `addBlocks: ["3"]`
4. Display updated dependency info

**TaskUpdate calls**:
```json
// Update the dependent task
{"taskId": "3", "addBlockedBy": ["1"]}

// Update the blocking task
{"taskId": "1", "addBlocks": ["3"]}
```

### remove
Remove a dependency relationship.

```
/task-deps remove 3 --blocked-by=1
```

**Steps**:
1. Use TaskGet on task 3 to get current `blockedBy`
2. Use TaskGet on task 1 to get current `blocks`
3. Note: There's no `removeBlockedBy` - dependencies are managed via the array fields
4. Inform user that dependency removal requires checking if the blocking task is completed

**Note**: The built-in system doesn't have explicit remove operations for dependencies. Once a blocking task is completed, the dependent task becomes unblocked automatically.

### check
Check dependencies for a specific task.

```
/task-deps check 3
```

**Steps**:
1. Use TaskGet to get the target task
2. Use TaskList to get all tasks (for resolving IDs to names)
3. For each ID in `blockedBy`: Show task subject and status
4. For each ID in `blocks`: Show task subject and status
5. Calculate if task is ready or blocked

**Output**:
```
Dependencies for: #3 (fix-auth-bug)
---
Blocked By (must complete first):
  [completed] #1 update-api
  [in_progress] #2 review-security

Blocks (waiting on this task):
  [pending] #4 deploy-prod
  [pending] #5 update-docs

Status: BLOCKED (1 dependency incomplete)
```

### graph
Visualize the full dependency graph.

```
/task-deps graph
```

**Steps**:
1. Use TaskList to get all tasks
2. Build adjacency list from `blocks`/`blockedBy` fields
3. Detect circular dependencies using DFS
4. Display ASCII graph

**Output**:
```
Task Dependency Graph
---
#1 update-api ──┬──> #3 fix-auth ──┬──> #4 deploy-prod
                │                   └──> #5 update-docs
#2 review-security ─┘

Legend:
──> Dependency flow (left must complete before right)
[*] In progress
[v] Completed
[!] Blocked
```

## Parameters
- `<action>`: add | remove | check | graph
- `<task-id>`: Task ID (numeric) or shortname
- `--blocks=<id>`: Task that will be blocked by this task
- `--blocked-by=<id>`: Task that blocks this task

## Batch Operations

Add multiple dependencies at once:
```
/task-deps add 5 --blocks="6,7,8"
```

Translates to:
```json
{"taskId": "5", "addBlocks": ["6", "7", "8"]}
{"taskId": "6", "addBlockedBy": ["5"]}
{"taskId": "7", "addBlockedBy": ["5"]}
{"taskId": "8", "addBlockedBy": ["5"]}
```

## Circular Dependency Detection

Before adding a dependency:
1. Build current dependency graph from TaskList
2. Simulate adding the new edge
3. Run cycle detection (DFS from target node)
4. If cycle found: Warn and abort

```
Cannot add dependency: #3 -> #1
This would create a circular dependency:
  #1 -> #2 -> #3 -> #1

Dependency not added.
```

## Dependency Resolution Order

When running `graph`, also show optimal execution order:
```
Suggested Execution Order:
1. #1 update-api (no dependencies)
2. #2 review-security (no dependencies)
3. #3 fix-auth (depends on #1, #2)
4. #4 deploy-prod (depends on #3)
5. #5 update-docs (depends on #3)
```

Uses topological sort on the dependency graph.

## Error Handling
- Task not found: "Task #N not found. Use `/task-list` to see available tasks."
- Circular dependency: Show the cycle path and abort
- Self-dependency: "Cannot add self-dependency on task #N"

## Important
- Always update BOTH sides of a dependency (blocks AND blockedBy)
- The built-in system tracks dependencies but doesn't prevent starting blocked tasks
- Use `/task-next` for automatic dependency-aware task selection
