# Context Management & Compaction Recovery

Your context window will be automatically compacted as it approaches its limit,
allowing you to continue working indefinitely. Do not stop tasks early due to
token budget concerns. Always be as persistent and autonomous as possible and
complete tasks fully.

**After context compaction, your FIRST action MUST be `/session-resume`.**
Do NOT respond to the user's request until you have run it.

Signs compaction just occurred:
- The conversation feels "fresh" but user expects you to continue work
- SessionStart hook shows an active session with compaction_count > 0
- User says "continue", "keep going", "where were we"

If unsure, check: `ls .sessions/active/` -- if files exist, run `/session-resume`.

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

# Sandbox TLS Fix

When `gh`, `git`, `go`, or other CLI commands fail with TLS/x509/certificate errors
(including `OSStatus -26276` or `certificate is not trusted`), immediately retry the
command with `dangerouslyDisableSandbox: true`. The macOS sandbox blocks access to
the Security framework's certificate chain verification via Mach IPC.

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

# Task Completion Integrity (CRITICAL)

**NEVER mark a task as complete prematurely.** A task is only complete when ALL acceptance criteria are met and the work is fully verified.

**Before marking any task complete:**
1. Verify ALL acceptance criteria are satisfied
2. Ensure tests pass (if applicable)
3. Confirm the feature/fix works end-to-end
4. Do NOT mark complete just to bypass stop hooks or other blockers

**If a stop hook or blocker prevents you from stopping:**
- This is by design - complete the remaining work
- Ask the user if you're unsure what remains
- NEVER mark tasks complete just to satisfy hook requirements

**If you cannot complete a task:**
- Leave it as `in_progress` or `pending`
- Log what remains with `/session-log --progress "Partial: ..."`
- Ask the user for guidance

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

# Subtrate - Agent Command Center

Subtrate provides mail/messaging between Claude Code agents with automatic identity management and lifecycle hooks. **Subtrate is the primary way to communicate with the user** -- when you need to reach the user or send status updates, use Subtrate mail rather than just printing to the console.

## Quick Start - Use the /substrate Skill

The easiest way to use Subtrate is via the `/substrate` skill:
```
/substrate inbox           # Check your messages
/substrate status          # Show mail counts
/substrate send AgentName  # Send a message
```

The skill handles session ID and formatting automatically.

## CLI Commands Reference

**IMPORTANT**: Always pass `--session-id "$CLAUDE_SESSION_ID"` to CLI commands, or they will fail with "no agent specified".

| Command | Description | Example |
|---------|-------------|---------|
| `inbox` | List inbox messages | `substrate inbox --session-id "$CLAUDE_SESSION_ID"` |
| `read <id>` | Read a specific message | `substrate read 42 --session-id "$CLAUDE_SESSION_ID"` |
| `send` | Send a new message | `substrate send --session-id "$CLAUDE_SESSION_ID" --to User --subject "Hi" --body "..."` |
| `status` | Show mail counts | `substrate status --session-id "$CLAUDE_SESSION_ID"` |
| `poll` | Wait for new messages | `substrate poll --session-id "$CLAUDE_SESSION_ID" --wait=30s` |
| `heartbeat` | Send liveness signal | `substrate heartbeat --session-id "$CLAUDE_SESSION_ID"` |
| `identity current` | Show your agent name | `substrate identity current --session-id "$CLAUDE_SESSION_ID"` |

**There is NO `reply` command** - to reply, use `send` with the sender as recipient:
```bash
# Read message #42 from AgentX, then reply:
substrate send --session-id "$CLAUDE_SESSION_ID" \
  --to AgentX \
  --subject "Re: Original Subject" \
  --body "Your reply here..."
```

## Setup

```bash
# Check if hooks are installed
substrate hooks status

# Install hooks (idempotent - safe to run multiple times)
substrate hooks install
```

No manual identity setup needed - your agent identity is auto-created on first use and persists across sessions and compactions.

## What the Hooks Do

| Hook | Behavior |
|------|----------|
| **SessionStart** | Heartbeat + inject unread messages as context |
| **UserPromptSubmit** | Silent heartbeat + check for new mail |
| **Stop** | Long-poll 9m30s, always block to keep agent alive (Ctrl+C to force exit) |
| **SubagentStop** | Block once if messages exist, then allow exit |
| **PreCompact** | Save identity for restoration after compaction |

The Stop hook keeps your agent alive indefinitely, checking for work from other agents. Press **Ctrl+C** to force exit.

## When Stop Hook Shows Mail (ACTION REQUIRED)

**CRITICAL**: When the stop hook blocks with "You have X unread messages", you MUST:

1. **Read your mail immediately**:
   ```bash
   substrate inbox --session-id "$CLAUDE_SESSION_ID"
   ```

2. **Process each message** - read the full content with `substrate read <id> --session-id "$CLAUDE_SESSION_ID"`

3. **Respond or act** on what's requested in the messages

4. **Only then** should you wait for the next user request

**DO NOT** just say "Standing by" or "Ready" when you have mail - this ignores messages from other agents who need your help!

**Example flow when stop hook shows mail:**
```
Stop hook: "You have 1 unread message from AgentX"
→ Run: substrate inbox --session-id "$CLAUDE_SESSION_ID"
→ Run: substrate read 42 --session-id "$CLAUDE_SESSION_ID"
→ Process the request in the message
→ Reply if needed: substrate send --session-id "$CLAUDE_SESSION_ID" --to AgentX --subject "Re: ..." --body "Done!"
```

## Agent Message Context (IMPORTANT)

When sending messages via Subtrate, **ALWAYS** include a brief context intro so recipients understand your situation:

**Format:**
```
[Context: Working on <project> in <directory>, branch: <branch>]
[Current task: <brief description of what you're doing>]

<actual message body>
```

**Example:**
```
[Context: Working on subtrate in ~/gocode/src/github.com/roasbeef/subtrate, branch: main]
[Current task: Implementing gRPC integration tests]

Hi, I need help with the test harness setup. The embedded server starts
correctly but I'm not sure how to configure the notification hub...
```

**Why this matters:**
- Recipients see multiple agents across different projects
- Context helps them understand your situation without asking
- Makes replies more relevant and helpful
- Essential for async communication between agents

**Include in your context:**
- Project name and directory
- Current git branch
- What task or goal you're working on
- Any relevant blockers or decisions needed
