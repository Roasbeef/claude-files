# Context Management

Your context window will be automatically compacted as it approaches its limit,
allowing you to continue working indefinitely from where you left off.
Therefore, do not stop tasks early due to token budget concerns. As you
approach your token budget limit, save your current progress and state to
memory before the context window refreshes. Always be as persistent and
autonomous as possible and complete tasks fully, even if the end of your budget
is approaching. Never artificially stop any task early regardless of the
context remaining.

# Post-Compaction Recovery (CRITICAL)

After context compaction, your FIRST action MUST be:
```
/session-resume
```

Do NOT respond to the user's request until you have run `/session-resume`.

**Signs you just experienced compaction:**
- The conversation feels "fresh" but user expects you to continue work
- SessionStart hook shows an active session with compaction_count > 0
- User says "continue", "keep going", "where were we"

**If unsure whether compaction occurred, check:**
```bash
ls .sessions/active/  # If files exist, run /session-resume
```

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

# Hunk for Precision Staging

Hunk enables line-level git staging, designed for AI agents who know exactly which lines they changed.

**When to use hunk instead of regular git:**
- You modified multiple areas of a file but only want to commit some changes
- You want to make atomic, focused commits from a larger set of changes
- You need to stage specific line ranges without interactive prompts

**Core workflow:**
```bash
hunk diff --json                # See changes with line numbers (machine-readable)
hunk diff                       # Human-readable diff with line numbers
hunk stage main.go:42-45        # Stage specific lines
hunk stage main.go:10-20,30-40  # Stage multiple ranges
hunk preview                    # See what will be committed
hunk commit -m "message"        # Commit staged changes
hunk reset                      # Unstage if needed
```

**FILE:LINES syntax:**
- `file.go:10` - Single line
- `file.go:10-20` - Range (inclusive)
- `file.go:10-20,30-40` - Multiple ranges in one file
- `file.go:10 other.go:5-8` - Multiple files (space-separated)

Line numbers refer to **new file** lines (what editors display), not old file lines.

**Best practices:**
- Run `hunk diff --json` to get exact line numbers before staging.
- Use `hunk preview` to verify the patch looks correct before committing.
- For focused commits, stage only related changes together.

# Hunk for Programmatic Rebase

Hunk provides non-interactive rebase commands for AI agents who need to manipulate git history without prompts.

**When to use hunk rebase:**
- Squashing fixup commits into their parent
- Dropping debug/temporary commits before PR
- Reordering commits for logical grouping
- Running commands (tests) between commits during rebase

**Core workflow:**
```bash
hunk rebase list --onto main           # See commits to rebase
hunk rebase run --onto main <actions>  # Execute rebase
hunk rebase status                     # Check if rebase in progress
hunk rebase continue                   # Continue after resolving conflicts
hunk rebase abort                      # Abort and restore original state
```

**Action syntax (comma-separated):**
- `pick:abc123` - Keep commit as-is
- `squash:abc123` - Combine with previous (concat messages)
- `fixup:abc123` - Combine with previous (discard message)
- `drop:abc123` - Remove commit from history
- `reword:abc123:New message` - Change commit message
- `exec:make test` - Run command after previous commit

**Common patterns:**
```bash
# Squash last 2 commits into one
hunk rebase list --onto main --json  # Get commit hashes
hunk rebase run --onto main "pick:first,squash:second"

# Drop a debug commit
hunk rebase run --onto main "pick:a,drop:debug,pick:b"

# Run tests after each commit
hunk rebase run --onto main "pick:a,exec:make test,pick:b,exec:make test"
```

**Conflict handling:**
```bash
hunk rebase status --json  # Check for conflicts
# Resolve conflicts manually, then:
git add <resolved-files>
hunk rebase continue
# Or abort:
hunk rebase abort
```

**Best practices:**
- Always use `hunk rebase list --onto <base> --json` first to get exact commit hashes.
- Use fixup (not squash) when you want to silently fold in typo fixes.
- Run `hunk rebase status` after run to verify completion.

# Task Management

Tasks use Claude Code's built-in task system (TaskCreate, TaskGet, TaskUpdate, TaskList tools).

**Commands:**
- `/task-list` - List all tasks with optional filters
- `/task-add` - Create a new task
- `/task-view` - View detailed task information
- `/task-next` - Pick and start the next priority task
- `/task-complete` - Mark a task as completed
- `/task-status` - Update task status (ready, in_progress, blocked, completed)
- `/task-deps` - Manage task dependencies and view dependency graph

**Priority levels:** P0 (critical) > P1 (high) > P2 (medium) > P3 (low)
**Size estimates:** XS (<1h), S (1-4h), M (4-8h), L (1-3d), XL (3d+)

**Status mapping:**
- `ready` = pending with no blockers
- `in_progress` = actively being worked
- `blocked` = pending with blockedBy tasks or blocked_reason set
- `completed` = finished (stays in list, use filters to hide)

**Metadata schema:** priority, size, tags, shortname, acceptance_criteria, blocked_reason, timestamps

# Session Management
Sessions provide execution continuity across context compactions and work periods.
See `~/.claude/SESSIONS.md` for full documentation.

## Automatic Session Logging (MANDATORY)

When a session is active (`.sessions/active/` has files), you MUST log at these moments:

### Log Triggers (When to Log)

**1. Key component finished** → `--progress`
- Completed a function, method, or logical unit
- Fixed a bug (include file:line)
- Added/modified a test
```
/session-log --progress "Implemented validateTx in chain.go:145-180"
/session-log --progress "Fixed nil pointer bug in sweeper.go:245"
```

**2. Bug/task milestone** → `--progress`
- Root cause identified
- Fix verified working
- Tests passing
```
/session-log --progress "Root cause: missing lock in concurrent path"
/session-log --progress "Fix verified: all 12 tests passing"
```

**3. New information learned** → `--discovery`
- Found undocumented behavior
- Discovered a constraint or requirement
- Learned something that affects the approach
```
/session-log --discovery "channeldb uses big-endian for keys, not little-endian"
/session-log --discovery "Must acquire mutex before channel state access"
```

**4. Decision made** → `--decision`
- Chose between multiple approaches
- Made a tradeoff
```
/session-log --decision "Using mutex over channel" --rationale="simpler, no concurrent readers"
```

**5. Blocked** → `--blocker`
- Need user input
- Missing information
- Unexpected failure
```
/session-log --blocker "Need to know: should this return error or panic?"
```

### Quick Reference
```
/session-log --progress "What you completed"
/session-log --discovery "What you learned"
/session-log --decision "Choice" --rationale="Why"
/session-log --blocker "What's stopping you"
```

### When to Checkpoint
Run `/session-checkpoint` after:
- Completing a major milestone
- Before risky changes
- After 5+ log entries
- Every 30-45 min of active work

## Command Reference
- `/session-init` - Start new session (user runs this)
- `/session-resume` - Continue after compaction
- `/session-log` - YOU run this proactively during work
- `/session-checkpoint` - YOU run this to save state
- `/session-view` - Check current session state
- `/session-pause` - Pause session (user runs this)
- `/session-close --complete` - Complete session (user runs this)

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
