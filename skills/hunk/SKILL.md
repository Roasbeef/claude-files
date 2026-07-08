---
name: hunk
description: Line-level git staging and non-interactive rebase for agents that know exactly which lines they changed. Use when you need atomic commits carved out of a larger diff, or when you need to squash/drop/reorder/reword commits without interactive prompts.
---

# Hunk

Hunk gives line-level git staging and non-interactive rebase, built for AI agents rather than a human at a terminal.

## Precision staging

Use hunk instead of plain `git add` when:
- You modified multiple areas of a file but only want to commit some of the changes.
- You want atomic, focused commits carved out of a larger set of changes.
- You need to stage specific line ranges without interactive prompts.

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

**Atomic change groups:** when a replacement (deletions + additions with no context between them) is partially selected, hunk automatically includes the entire group. You cannot stage half a replacement, the deletions and additions are atomic. Pure-addition and pure-deletion groups can still be individually line-selected.

**Fallback when staging fails:** if `hunk stage` fails with "patch does not apply," fall back to `git add <file>` for whole-file staging, broader line ranges that cover entire change groups, or stage file-by-file instead of cherry-picking lines across many hunks.

**Best practices:** run `hunk diff --json` to get exact line numbers before staging. Use `hunk preview` to verify the patch looks correct before committing. For focused commits, stage only related changes together.

## Programmatic rebase

Use hunk's rebase commands when you need to squash fixup commits into their parent, drop debug/temporary commits before a PR, reorder commits for logical grouping, or run commands (tests) between commits during a rebase.

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
hunk rebase list --onto main --json           # Get commit hashes
hunk rebase run --onto main "pick:abc123,squash:def456"

# Drop a debug/temporary commit from history
hunk rebase run --onto main "pick:abc123,drop:debug1,pick:ghi789"

# Run tests after each commit to verify history is clean
hunk rebase run --onto main "pick:abc123,exec:go test ./...,pick:def456,exec:go test ./..."

# Reword a commit message
hunk rebase run --onto main "pick:abc123,reword:def456:fix: correct nil check in handshake"

# Squash multiple fixup commits into their targets
hunk rebase run --onto main "pick:feat1,fixup:typo1,pick:feat2,fixup:typo2"
```

**Auto-squash** (preferred for fixup commits):
```bash
hunk rebase autosquash --onto main              # Squash all fixup!/squash! commits
hunk rebase autosquash --onto main --dry-run    # Preview what would be squashed
```

Create fixups with `git commit --fixup=<sha>`, then auto-squash.

**Conflict handling:**
```bash
hunk rebase status                  # Check for conflicts
# Resolve conflicts manually, then:
git add <resolved-files>
hunk rebase continue
# Or abort:
hunk rebase abort
```

**Best practices:** always run `hunk rebase list --onto <base> --json` first to get exact hashes. Prefer `autosquash` over manual `run` when squashing fixups. Use fixup (not squash) when you want to silently fold in typo fixes. Run `hunk rebase status` after a run to verify completion.
