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

# Decision Checkpoints

At a significant decision point, first decide **who** to ask.

- **Judgment call** ("is this approach sound," "what am I missing," "which of
  two designs is better") -> consult `/advisor`, but only when this session's
  main loop is running on the cheap tier (Sonnet). If the session is already on
  Opus/Fable, you *are* the advisor, so just reason it through yourself. Aim
  for roughly once per non-trivial task; over-consulting turns the cheap
  session expensive one call at a time.
- **Preference or scope call** that only the user can settle -> STOP and use
  `AskUserQuestion`. This covers:
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

# Parallel Work: /orchestrate

When a task decomposes into independent, parallel work items (a multi-file
refactor, N similar edits across a codebase, a broad research or audit sweep),
reach for `/orchestrate`: an expensive planner (Opus/Fable) splits the work,
cheap Sonnet/Haiku workers run it in parallel, and the planner synthesizes the
results. It has `disable-model-invocation: true`, so it never fires on its
own, you have to invoke it by name. Don't reach for it on inherently
sequential work or a single-file change; that's just doing the work, or
`/advisor` if you need a course-correction along the way.

# Who to Talk To

Three channels, three purposes: `AskUserQuestion` for a blocking
preference/scope decision only the user can make; Subtrate mail for async
status updates and reaching the user outside a blocking prompt; `/advisor` for
consulting a stronger model on a judgment call. Don't conflate them.

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

# Coding Style
- For comments, always use complete sentences ending with a period.
- If unsure about a Go package/struct API, use `go doc` to look it up.

# Writing Skills

Two orthogonal prose skills live in `~/.claude/skills/` and are available in
every project:

- **roasbeef-prose** is the *voice* skill: write the way I write (cadence,
  idioms, "In this commit, we...", my structure). Reach for it on PRs, commit
  messages, docs, and posts. A future pass will tighten it from a corpus of my
  own writing.
- **technical-writing** is the *clarity* skill, distilled from Pinker's *The
  Sense of Style*: audience-neutral principles for clear prose (the window
  metaphor, the curse of knowledge, coherence, syntax, usage).

Voice and clarity compose: my voice, made clear. When the two skills conflict
on a style choice (e.g. em-dashes, which technical-writing welcomes but
roasbeef-prose bans), **roasbeef-prose wins** because it encodes my actual
voice; let technical-writing inform structure and clarity underneath.

# Git & PRs
- Don't include "Generated with Claude Code" or "Co-Authored-By: Claude" in commit messages or PR bodies.
- Don't add any AI attribution footers to commits or PRs.
- When creating a new branch or worktree, never include "claude" in the name.

---

# Sandbox TLS Fix

When `gh`, `git`, `go`, or other CLI commands fail with TLS/x509/certificate errors
(including `OSStatus -26276` or `certificate is not trusted`), immediately retry the
command with `dangerouslyDisableSandbox: true`. The macOS sandbox blocks access to
the Security framework's certificate chain verification via Mach IPC.

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

# Hunk for Line-Level Staging and Rebase

Hunk gives line-level git staging and non-interactive rebase, built for agents
that know exactly which lines they changed. Use it for atomic commits carved
out of a larger diff, or to squash/drop/reorder/reword commits without
interactive prompts. Full command reference, syntax, and worked examples live
in the `hunk` skill, read it before a staging or rebase task rather than
guessing the flags.

# Reviewing Finished Work

Before opening a PR, get an independent look at the diff. Two options, pick whichever fits:

- **Substrate review** - `substrate review request --session-id "$CLAUDE_SESSION_ID"`
  spawns a reviewer agent against the current branch/PR; check back with
  `review status`/`review issues`, address findings, then `review resubmit`.
  Use `--type security`/`--type performance`/`--type architecture` when the
  change touches that surface specifically.
- **Background review agent** - spawn a `code-reviewer` agent (or the
  `review-loop` skill for a full adversarial review-fix-verify cycle) as a
  background task once the work is done, and fold in what it finds.

# Session Management

Sessions provide execution continuity across context compactions and work
periods. See `~/.claude/SESSIONS.md` for full documentation.

When a session is active (`.sessions/active/` has files), you must log at
these moments as you work:

```
/session-log --progress "What you completed"
/session-log --discovery "What you learned"
/session-log --decision "Choice" --rationale="Why"
/session-log --blocker "What's stopping you"
```

Run `/session-checkpoint` after a major milestone, before risky changes, after
5+ log entries, or every 30-45 min of active work.

Commands: `/session-init` (user starts) - `/session-resume` (you, immediately
after compaction) - `/session-log` / `/session-checkpoint` (you, proactively) -
`/session-view` (check state) - `/session-pause` / `/session-close --complete`
(user ends).

# Subtrate

Subtrate is how you talk to the user and other agents outside a blocking
prompt: async mail, status updates, code review requests. Use the
`/substrate` skill for the CLI (`inbox`, `status`, `send`, `review request`,
etc.) rather than memorizing flags here.

Gotchas that aren't obvious from `--help`:
- Every CLI call needs `--session-id "$CLAUDE_SESSION_ID"` or it fails with "no agent specified."
- There is no `reply` command - reply by `send`-ing to the original sender.
- When the Stop hook blocks with unread mail, read and act on it before doing
  anything else; don't just say "standing by."
- After making commits on a feature branch, `substrate send-diff --session-id
  "$CLAUDE_SESSION_ID" --to User --base main` sends the diff to the user with
  syntax highlighting in the web UI.
