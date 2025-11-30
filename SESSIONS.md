# Session Management System

A session-centric execution journal for maintaining continuity across context compactions and work periods.

## Why Sessions?

### The Problem
Claude Code's context window compacts automatically when it approaches its limit. While this allows indefinite work, it creates a challenge: **how do you resume work effectively after compaction?**

Without a system:
- You lose track of what you were doing
- Decisions and their rationale are forgotten
- Discoveries made during work are lost
- You waste time re-understanding the codebase

### The Solution
Sessions provide a **living document** that:
- Survives context compactions
- Captures progress, decisions, and discoveries
- Enables rapid context restoration
- Maintains execution continuity across work periods

Sessions are inspired by:
- [Codex ExecPlans](https://github.com/openai/openai-cookbook/blob/main/articles/codex_exec_plans.md) - living execution documents
- [Anthropic's Long-Running Agent Patterns](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents) - session continuity techniques

---

## How It Works

### Session Lifecycle

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   INIT      │────▶│   ACTIVE    │────▶│  COMPLETE   │
│ /session-   │     │   (work)    │     │ /session-   │
│   init      │     │             │     │   close     │
└─────────────┘     └─────────────┘     └─────────────┘
                          │
                          │ compaction
                          ▼
                    ┌─────────────┐
                    │  AUTO-SAVE  │
                    │ (PreCompact │
                    │    hook)    │
                    └─────────────┘
                          │
                          │ /session-resume
                          ▼
                    ┌─────────────┐
                    │   RESUME    │
                    │  (continue) │
                    └─────────────┘
```

### File Structure

Sessions are stored per-project in `.sessions/`:

```
.sessions/
├── active/                     # Currently active sessions
│   └── fix-sweep-bug-abc123.md
├── archive/                    # Completed/closed sessions
│   └── implement-rbf-def456.md
└── journal/                    # Per-session journals
    └── abc123/
        ├── progress.md         # Quick context file
        └── entries/            # Timestamped entries
            └── 2025-01-15T10:00:00.md
```

---

## Session File Format

Sessions use YAML frontmatter + structured markdown:

```markdown
---
id: 019abc12-3456-7def-8901-234567890abc
shortname: fix-sweep-bug
status: active
task_ids:
  - 019def45-6789-7abc-0123-456789abcdef
created_at: 2025-01-15T10:00:00Z
updated_at: 2025-01-15T14:30:00Z
compaction_count: 2
git_branch: fix/sweep-trigger
git_last_commit: abc123f
---

# Session: Fix Sweep Trigger Bug

## TL;DR (Read This First)
Fixing race condition in sweep trigger. Found root cause in sweeper.go:245.
Currently implementing mutex fix. Next: complete mutex, add tests.

**Progress**: 3/5 steps complete
**Current**: Implement fix (step 4)
**Blocker**: None

## Context
**Objective**: Fix race condition causing intermittent sweep failures

**Key Files**:
- `sweep/sweeper.go` - Main sweeper logic
- `contractcourt/chain_watcher.go` - Triggers sweeps

**Key Context** (survives compaction):
- Race condition: state check before lock acquisition
- Lock ordering: chain_watcher -> sweeper (not reverse!)
- Test command: `go test -v -run TestSweepTrigger ./itest`

## Progress
### Completed
- [x] Reproduce bug in test (2025-01-15T10:30)
- [x] Identify root cause (2025-01-15T11:00)
- [x] Draft fix approach (2025-01-15T12:00)

### In Progress
- [ ] Implement fix          <- CURRENT

### Remaining
- [ ] Add regression test

## Decisions
### 1. Lock-first approach (2025-01-15T12:15)
**Context**: Need to fix race without major refactor
**Options**: (1) Lock before check, (2) Atomic state, (3) Separate mutex
**Choice**: Lock before check
**Rationale**: Simplest fix, performance impact minimal

## Discoveries
1. **Lock ordering** (2025-01-15T13:00): chain_watcher must acquire
   lock before sweeper to avoid deadlock. Document this!

## Blockers
- None currently

## Next Steps
1. Complete mutex implementation in sweeper.go
2. Add regression test
3. Document lock ordering in CONTRIBUTING.md
```

---

## Commands Reference

### `/session-init`
Start a new work session.

```bash
/session-init                           # Interactive mode
/session-init --task=<id>               # Link to existing task
/session-init --tasks=<id1,id2>         # Link multiple tasks
/session-init --name="fix-auth"         # Custom shortname
/session-init --goal="Description"      # Set objective
```

### `/session-resume`
Resume an existing session with full context restoration.

```bash
/session-resume                         # Resume active/most recent
/session-resume <session-id>            # Resume specific session
/session-resume --list                  # List available sessions
```

### `/session-log`
Add entries to the session's execution log.

```bash
/session-log --progress "Completed fee estimation"
/session-log --decision "Using mutex" --rationale="simpler than channels"
/session-log --discovery "Found lock ordering issue"
/session-log --blocker "Waiting on API docs"
/session-log --next "Add unit tests"
/session-log --done "Implement fee estimation"
```

### `/session-checkpoint`
Save an explicit checkpoint of current state.

```bash
/session-checkpoint                     # Save now
/session-checkpoint --note="Before refactor"
```

### `/session-view`
View session details.

```bash
/session-view                           # Current session
/session-view <session-id>              # Specific session
/session-view --progress                # Just progress
/session-view --decisions               # Just decisions
/session-view list                      # List all sessions
```

### `/session-pause`
Pause the current session.

```bash
/session-pause                          # Pause with checkpoint
/session-pause --note="Switching to urgent bug"
```

### `/session-close`
Complete or abandon a session.

```bash
/session-close --complete               # Mark complete, archive
/session-close --abandon --reason="Requirements changed"
```

---

## Hooks Integration

### PreCompact Hook (Automatic)
Before context compaction, the system automatically:
1. Increments `compaction_count` in session file
2. Updates timestamps
3. Outputs key context (TL;DR, progress, next steps) to compaction summary

### SessionStart Hook (Automatic)
When starting Claude Code in a project with an active session:
1. Displays session TL;DR
2. Shows current progress
3. Lists any blockers
4. Suggests `/session-resume` for full context

### UserPromptSubmit Hook (Automatic)
When you say "continue", "resume", "where were we", etc.:
1. Detects continuation intent
2. Automatically injects session context
3. Helps Claude resume seamlessly

---

## Best Practices

### When to Start a Session
- Starting work on a task that will take multiple hours
- Work that might span multiple conversations
- Complex debugging that needs decision tracking
- Any work where you'd want to "pick up where you left off"

### What to Log

**Progress** - Concrete completed or in-progress items:
```bash
/session-log --progress "Implemented mutex wrapper in sweeper.go:245-260"
```

**Decisions** - Choices with rationale (survives compaction!):
```bash
/session-log --decision "Using sync.Mutex over channels" \
  --rationale="Simpler, sufficient for this use case, easier to reason about"
```

**Discoveries** - Unexpected findings:
```bash
/session-log --discovery "Lock ordering: chain_watcher must lock before sweeper"
```

**Blockers** - What's preventing progress:
```bash
/session-log --blocker "Need clarification on API behavior from team"
```

### Minimum Logging
At minimum, log before:
- Switching to a different task
- Ending your work session
- Context compaction is approaching
- Asking a blocking question
- Making significant implementation decisions

### Session File Conventions
- **TL;DR**: One paragraph + progress fraction + current step + blocker status
- **Key Context**: Technical details that MUST survive compaction
- **Progress**: Use `<- CURRENT` marker for active step
- **Decisions**: Always include rationale
- **Next Steps**: Immediate actions first

---

## Example Workflow

### Day 1: Starting Work

```
You: /task-next

Claude: Starting task: fix-sweep-bug (P1, M)
        Would you like to start a session? (recommended for tracking)

You: yes

Claude: /session-init --task=fix-sweep-bug

📋 Session Initialized: fix-sweep-bug
   ID: abc123...
   Branch: main
   ...

You: Let's investigate the bug

Claude: [investigates, finds root cause]
        /session-log --progress "Found root cause: race condition in sweeper.go:245"
        /session-log --discovery "State check happens before lock acquisition"

[Context compaction occurs]

💾 Pre-Compaction Checkpoint
   Session: fix-sweep-bug
   Compaction: #1

   ## TL;DR
   Investigating sweep trigger bug. Found race condition in sweeper.go:245.
   ...
```

### Day 2: Resuming Work

```
[SessionStart hook displays:]

📋 Active Session: fix-sweep-bug
   (after 1 compaction(s))

   ## TL;DR
   Found race condition in sweeper.go:245...

You: continue where we left off

Claude: [UserPromptSubmit hook injects context]

        /session-resume

📋 Resuming Session: fix-sweep-bug
   ...
   ## Where You Left Off
   Found race condition. Ready to implement fix.
   ...

[Work continues seamlessly]
```

### Completing Work

```
You: /session-close --complete

✅ Session Completed: fix-sweep-bug
   Duration: 2 days
   Compactions: 3
   Progress: 5/5 steps

   ## Summary
   Fixed race condition in sweep trigger...

   ## Decisions Made
   2 decisions recorded

   ## Discoveries
   1 discovery logged

   Linked task fix-sweep-bug: completed ✅
```

---

## Relationship to Tasks

Sessions and tasks are complementary:

| Aspect | Tasks (.tasks/) | Sessions (.sessions/) |
|--------|-----------------|----------------------|
| Purpose | What to do | How work progresses |
| Lifetime | Project lifetime | Single work effort |
| Granularity | Feature/bug level | Execution level |
| Survives | Project history | Compactions |

**Best practice**: Link sessions to tasks with `/session-init --task=<id>`.
This connects *what* you're doing with *how* you're doing it.

---

## Troubleshooting

### No session context after compaction
Run `/session-resume` to restore full context. The SessionStart hook shows
a summary, but `/session-resume` provides complete restoration.

### Lost progress
Check `.sessions/active/` for your session file. The full history is preserved.
Use `/session-view` to see all details.

### Want to switch tasks
Use `/session-pause` to save current session, then start a new one with
`/session-init --task=<new-task>`.

### Session feels stale
Update the TL;DR and Key Context sections with `/session-checkpoint`.
This regenerates the quick-resume context.
