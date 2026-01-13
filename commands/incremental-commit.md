# incremental-commit
---
allowed-tools: Bash(hunk diff:*), Bash(hunk stage:*), Bash(hunk preview:*), Bash(hunk commit:*), Bash(hunk reset:*), Bash(git status:*), Bash(git diff:*), Bash(git log:*), Bash(git add:*), Bash(git commit:*), Bash(sg run:*), Read, Grep, Glob
description: Create incremental, atomic commits that tell a story with detailed commit messages
argument-hint: [optional: specific files or directories to focus on]
---

Create incremental, atomic commits for the current changes. Think deeply about how to break down changes into logical, atomic commits that each tell part of the story.

## Initial Analysis Phase

1. `git status` to see all modified/untracked files
2. `git diff` to analyze all unstaged changes
3. `git diff --cached` to see any already staged changes
4. `hunk diff --json` to get machine-readable output with exact line numbers
5. `git log --oneline -10` to understand recent commit style
6. Check CLAUDE.md for project-specific guidelines

## Commit Planning Phase

Analyze changes comprehensively and develop a mental model of the commit sequence.

Categorize changes into logical groups:
- **Isolated bug fixes**: Can be committed independently
- **Refactoring**: File moves, function reorganization, code cleanup
- **New features**: Core implementation separate from integration
- **Test additions**: Separate from implementation when possible
- **Documentation updates**: Usually their own commit

## Special File Classification

Identify files that should be committed separately:
- **Lock files**: `go.sum`, `package-lock.json`, `yarn.lock`, `Cargo.lock`
- **Generated files**: `*.pb.go`, `*_gen.go`, `*.generated.*`
- **Test files**: Can often be separate from implementation

## Staging Strategies

### Whole Files
When entire files form a logical unit:
```bash
git add src/feature.go src/feature_test.go
git commit -m "feature: add new capability"
```

### Line-Level with Hunk
When changes within a file need to be split:
```bash
hunk diff                     # See line numbers
hunk stage file.go:10-25      # Stage bug fix lines
hunk preview                  # Verify patch
hunk commit -m "fix: ..."
hunk stage file.go:40-60      # Stage feature lines
hunk commit -m "feat: ..."
```

### Understanding Hunk Line Numbers
The `hunk diff` output shows two columns of line numbers:
- **Left column (OLD)**: Line numbers in the original file (before changes)
- **Right column (NEW)**: Line numbers in the modified file (after changes)

Line number semantics by operation type:
- **Additions** (lines with `+`): Use NEW file line numbers (right column)
- **Deletions** (lines with `-`): Use OLD file line numbers (left column)
- **Replacements** (delete + add): Include BOTH old and new line numbers

### Staging Replacements
When staging changes that replace existing code (deletions followed by additions),
you must include line numbers for both the deleted and added lines.

Example: Replacing a field assignment in a struct initialization:
```
   45     45   func NewService(cfg Config) *Service {
   46     46       return &Service{
-  47            backend:  cfg.Backend,    // OLD line 47 being deleted
+       47       client:   cfg.Client,     // NEW line 47 being added
   48     48       timeout: cfg.Timeout,
```

To stage this replacement correctly:
```bash
# Include BOTH the old line (47) AND new line (47)
hunk stage service.go:47

# Verify the patch includes both deletion and addition
hunk preview
```

For more complex replacements spanning multiple lines:
```
   100    100   type Handler struct {
-  101          conn    net.Conn       // OLD lines 101-102 deleted
-  102          active  bool
+      101      client  *Client        // NEW lines 101-103 added
+      102      ready   bool
+      103      ctx     context.Context
   103    104   }
```

Stage with ranges covering both old and new:
```bash
# Old lines 101-102 (deletions) + new lines 101-103 (additions)
hunk stage handler.go:101-103
```

### Verification Workflow
Always verify before committing:
```bash
hunk diff --json              # Get exact line numbers
hunk stage file.go:LINES      # Stage the change
hunk preview                  # Verify patch looks correct
# If wrong, reset and try again:
hunk reset
```

### Multiple Files, Specific Lines
```bash
hunk stage api.go:15-30 handler.go:8-12
hunk commit -m "refactor: extract validation logic"
```

### Pattern-Based
```bash
git add "*_test.go"           # All test files
git add "lnwallet/*.go"       # All files in a package
```

## Dependency Detection

When function signatures change, find all callers:
```bash
# Using ast-grep for structural search
sg run -p '$FUNC($$$ARGS)' -l go

# Or grep for simpler cases
grep -r "funcName" --include="*.go" .
```

If changes are deeply intertwined, explain why they must be committed together.

## Commit Message Format

DO NOT overuse bullet points. Messages should read as natural prose.

```
subsystem: Brief summary (imperative mood, <50 chars)

In this commit, we [explain the change in natural prose, focusing
on the "why" more than the "what"]. This change improves [aspect]
by [approach/method].

[Additional context about trade-offs, alternatives considered,
or implementation details if needed. Keep prose natural.]

[For bug fixes, explain the root cause and the fix approach.]
```

## Subsystem Prefix Guidelines

- Single package: `package: description`
- Multiple packages: `pkg1+pkg2: description` or `multi: description`
- Project-wide: `multi: description`
- Build/CI: `build:` or `ci:`
- Documentation: `docs:`
- Tests: `test:` or `package/test:`

## Execution Flow

1. Analyze all changes comprehensively
2. Create a mental model of the commit sequence
3. For each planned commit:
   - `hunk stage FILE:LINES` or `git add` for whole files
   - `hunk preview` to verify the patch looks correct
   - Craft a detailed commit message
   - `hunk commit -m "message"`
4. Continue until all changes are committed

## Special Considerations

- For generated files, commit them separately with clear indication
- For vendored dependencies, use a dedicated commit
- Skip CI for trivial changes with `[skip ci]` suffix

Focus area: $ARGUMENTS
