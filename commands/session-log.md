---
description: Add entries to the current session's execution log
argument-hint: --progress|--decision|--discovery|--blocker|--next|--file <message>
---

Add a log entry to the current active session's execution log.

Arguments: $ARGUMENTS

## Entry Types:

### --progress "message"
Add a progress entry. Use for completed or in-progress work items.
```
/session-log --progress "Implemented fee estimation in fees.go"
/session-log --progress "Started working on RBF signing logic"
```

### --decision "title" [--context="..."] [--options="..."] [--rationale="..."]
Record a design decision with context and rationale.
```
/session-log --decision "Use mutex for thread safety" --rationale="simpler than channels, sufficient for this use case"
/session-log --decision "Incremental fee bumping" --context="choosing between incremental vs absolute" --options="1. increment, 2. absolute" --rationale="more predictable"
```

### --discovery "message"
Log an unexpected finding or insight.
```
/session-log --discovery "Found lock ordering issue - mempool lock must be acquired before wallet lock"
/session-log --discovery "BIP-125 test vectors don't cover all edge cases"
```

### --blocker "message" [--resolved]
Log a blocker or resolve an existing one.
```
/session-log --blocker "Waiting on API documentation from backend team"
/session-log --blocker "API docs received" --resolved
```

### --next "message"
Add to the Next Steps list.
```
/session-log --next "Add unit tests for edge cases"
/session-log --next "Update documentation in CONTRIBUTING.md"
```

### --file "filepath" [--description="..."]
Log a file modification.
```
/session-log --file "pkg/txbuilder/rbf.go" --description="New RBF implementation, +150 lines"
/session-log --file "pkg/txbuilder/fees.go" --description="Modified fee estimation, +45 lines"
```

### --done "step description"
Mark a progress item as complete.
```
/session-log --done "Implement fee estimation"
```

## Steps:
1. **Find Active Session:**
   - Look in `.sessions/active/` for session file
   - If no active session: Error with suggestion to run `/session-init`

2. **Parse Entry Type and Content:**
   - Determine entry type from flags
   - Parse message and optional parameters

3. **Update Session File:**
   - Add entry to appropriate section with timestamp
   - For --progress: Add to Progress section
   - For --decision: Add numbered entry to Decisions section
   - For --discovery: Add numbered entry to Discoveries section
   - For --blocker: Add to Blockers section (or mark resolved)
   - For --next: Add to Next Steps section
   - For --file: Add to Files Modified table
   - For --done: Mark checkbox as [x] in Progress

4. **Update Progress File:**
   - Update `.sessions/journal/{id}/progress.md`
   - Update TL;DR if significant change
   - Update checklist if progress changed
   - Update `last_updated` timestamp

5. **Update Frontmatter:**
   - Update `updated_at` in session file

## Output Format:
```
📝 Logged to session: {shortname}

Type: {progress|decision|discovery|blocker|next|file}
Entry: {message summary}

Session Progress: {N}/{M} ({pct}%)
Current Step: {current step description}
```

## Multiple Entries:
Can log multiple entries in one command:
```
/session-log --progress "Completed fee estimation" --next "Add unit tests" --file "fees.go"
```

## Auto-Timestamping:
All entries are automatically timestamped with ISO-8601 format:
```
### Completed
- [x] Implement fee estimation (2025-01-15T12:30:00Z)
```

## Progress Entry Format in Session File:
```markdown
## Progress
### Completed
- [x] Research BIP-125 requirements (2025-01-15T10:30)
- [x] Define RBFTransaction struct (2025-01-15T11:00)

### In Progress
- [ ] Implement RBF signing          <- CURRENT

### Remaining
- [ ] Add mempool submission
- [ ] Write unit tests
```

## Decision Entry Format:
```markdown
## Decisions
### 1. Incremental fee bumping (2025-01-15T10:45)
**Context**: Need to choose between incremental vs absolute fee replacement
**Options**: (1) Increment existing fee, (2) Set absolute new fee
**Choice**: Incremental bumping
**Rationale**: More predictable cost, easier to implement
```

## Discovery Entry Format:
```markdown
## Discoveries
1. **Lock ordering issue** (2025-01-15T13:00): Found that mempool lock must be
   acquired before wallet lock to avoid deadlock.
```

## No Active Session Error:
```
❌ No active session found.

To start a new session:
  /session-init --goal="description"
  /session-init --task=<task-id>

To resume an existing session:
  /session-resume
```
