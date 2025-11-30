# Context Management

Your context window will be automatically compacted as it approaches its limit,
allowing you to continue working indefinitely from where you left off.
Therefore, do not stop tasks early due to token budget concerns. As you
approach your token budget limit, save your current progress and state to
memory before the context window refreshes. Always be as persistent and
autonomous as possible and complete tasks fully, even if the end of your budget
is approaching. Never artificially stop any task early regardless of the
context remaining.

# Coding Style
- For comments, always use complete sentences ending with a period.
- If unsure about a Go package/struct API, use `go doc` to look it up.

# Decision Checkpoints

Before making significant decisions during implementation, STOP and use `AskUserQuestion`.
Do not make autonomous choices for non-trivial cases in these categories:

- **Removing/skipping tests** - Never remove, skip, or disable tests without asking
- **Deleting code/files** - Ask before removing functionality (dead imports are fine)
- **Architectural changes** - New patterns, dependencies, or structural changes not in original plan
- **Workarounds** - When the intended approach fails, ask before implementing alternatives
- **Scope changes** - Adding features not requested, or omitting requested features
- **API/interface changes** - Modifying public interfaces or contracts

**Example - instead of deciding autonomously:**
> "I'll remove these tests since they're incompatible with the new approach"

**Ask with options:**
> "The mutex approach breaks 3 channel-based tests. Options:
> 1. Rewrite tests for mutex pattern
> 2. Remove tests (reduces coverage)
> 3. Keep both patterns (more complexity)
> Which do you prefer?"

# Git & PRs
- Don't include "Generated with Claude Code" or "Co-Authored-By: Claude" in commit messages or PR bodies.
- Don't add any AI attribution footers to commits or PRs.

# Task Management
- Projects use `.tasks/` directory for task tracking.
- Run `/task-list` when starting work on any project.
- Key commands: `/task-add`, `/task-next`, `/task-complete`, `/task-view`, `/task-status`, `/task-deps`
- Priorities: P0 (critical) > P1 (high) > P2 (medium) > P3 (low)
- Sizes: XS (<1h), S (1-4h), M (4-8h), L (1-3d), XL (3d+)

# Session Management
Sessions provide execution continuity across context compactions and work periods.
See `~/.claude/SESSIONS.md` for full documentation.

## Automatic Session Logging (IMPORTANT)
When a session is active (check `.sessions/active/`), PROACTIVELY log as you work:

**Log decisions immediately when you make them:**
```
/session-log --decision "Using mutex instead of channels" --rationale="simpler, sufficient for this use case"
```

**Log discoveries when you find something unexpected:**
```
/session-log --discovery "Lock ordering matters: chain_watcher must lock before sweeper"
```

**Log progress after completing significant steps:**
```
/session-log --progress "Implemented fix in sweeper.go:245-260"
```

**Log blockers when you hit them:**
```
/session-log --blocker "Need clarification on API behavior"
```

This logging is NOT optional when a session is active - it's how context survives compaction.
The user manages session lifecycle (init/pause/close), you do the logging during work.

## Quick Reference
- `/session-init` - Start new session (user runs this)
- `/session-resume` - Continue after compaction
- `/session-log` - YOU run this proactively during work
- `/session-checkpoint` - YOU run this to save state
- `/session-view` - Check current session state
- `/session-pause` - Pause session (user runs this)
- `/session-close --complete` - Complete session (user runs this)

## When to Log (Auto-log Triggers)
Log automatically when you:
- Choose between multiple implementation approaches → `--decision`
- Find unexpected behavior or undocumented quirks → `--discovery`
- Complete a logical unit of work → `--progress`
- Get stuck or need external input → `--blocker`
- Are about to make a significant code change → `--progress` (what you're about to do)

## When to Checkpoint (Auto-checkpoint Triggers)
Run `/session-checkpoint` automatically:
- After completing a major milestone or phase of work
- Before making risky or large-scale changes
- After accumulating several log entries (5+ entries since last checkpoint)
- When switching focus to a different part of the codebase
- Before asking the user a blocking question
- Periodically during long work sessions (every 30-45 min of active work)

# ast-grep for Code Search and Style

**Prefer ast-grep over grep for Go code searching:**
- Use `sg run -p 'pattern' -l go` instead of `grep` for structural code search.
- ast-grep understands Go syntax, so `sg run -p 'func $NAME($$$ARGS)' -l go` finds all functions.
- For simple text search, grep is fine; for code patterns, use ast-grep.

**Style enforcement in projects with `sgconfig.yml`:**
- Run `sg scan` to check style issues before committing.
- Run `sg scan --update-all` to auto-fix safe patterns.
- Key patterns enforced:
  - Multi-line calls need trailing comma before closing paren.
  - Visual symmetry: matching indentation for open/close parens.
  - Structured logs need trailing comma.

**ast-grep pattern examples:**
- Find all function calls: `sg run -p '$FUNC($$$ARGS)' -l go`
- Find method calls: `sg run -p '$OBJ.$METHOD($$$ARGS)' -l go`
- Find error returns: `sg run -p 'return $ERR' -l go`
- Find struct literals: `sg run -p '&$TYPE{$$$FIELDS}' -l go`
