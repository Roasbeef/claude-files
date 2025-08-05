# incremental-commit
---
allowed-tools: Bash(git diff:*), Bash(git status:*), Bash(git add:*), Bash(git commit:*), Bash(git apply:*), Bash(git log:*), Bash(git reset:*), Write, Read, Grep, Glob, LS, MultiEdit
description: Create incremental, atomic commits that tell a story with detailed commit messages
argument-hint: [optional: specific files or directories to focus on]
---

Create incremental, atomic commits for the current changes. IMPORTANT: Think deeply about how to break down changes into logical, atomic commits that each tell part of the story.

DO NOT overuse bullet points in the commit messages. They should read as normal
prose, with paragraphs, etc.

## Initial Analysis Phase
1. Run `git status` to see all modified/untracked files
2. Run `git diff` to analyze all unstaged changes
3. Run `git diff --cached` to see any already staged changes
4. Run `git log --oneline -10` to understand recent commit style
5. Check for project-specific guidelines in CLAUDE.md

## Commit Planning Phase
Categorize changes into logical groups:
- **Isolated bug fixes**: Can be committed independently
- **Refactoring**: File moves, function reorganization, code cleanup
- **New features**: Core implementation separate from integration
- **Test additions**: Separate from implementation when possible
- **Documentation updates**: Usually their own commit

## Incremental Commit Strategy

### Strategy A: File-Level Staging (Simplest)
```bash
# Stage complete files that form a logical unit
git add src/feature.go src/feature_test.go
git commit -m "feature: add new capability"
```

### Strategy B: Using git apply (For Mixed Changes)
```bash
# 1. Create patches for specific files or functions
git diff path/to/file.go > file.patch

# 2. For specific functions/sections, use line ranges
git diff -U10 path/to/file.go | grep -A20 -B20 "funcName" > func.patch

# 3. Or use git's built-in function detection
git diff --function-context -U0 path/to/file.go > changes.patch

# 4. Apply specific patches to staging
git apply --cached func.patch

# 5. Verify what was staged
git diff --cached
```

### Strategy C: Multiple Changes in Same File
```bash
# Example: file has bug fix + new feature
# 1. Create separate patches
git diff file.go > all_changes.patch

# 2. Extract bug fix lines (using line numbers from diff)
sed -n '10,50p' all_changes.patch > bugfix.patch

# 3. Stage and commit bug fix
git apply --cached bugfix.patch
git commit -m "fix: resolve null pointer in validator"

# 4. Stage remaining changes
git add file.go
git commit -m "feature: add validation for new field type"
```

### Strategy D: Pattern-Based Staging
```bash
# Stage all test files
git add "*_test.go"

# Stage all files in a module
git add "lnwallet/*.go"

# Stage by diff pattern
git diff --name-only | grep -E "fix|bug" | xargs git add
```

### Strategy E: Extracting Specific Hunks (GUI-like behavior)
```bash
# 1. Generate a patch with hunk headers
git diff -U0 file.go | grep -E "^@@|^diff|^index" > hunks.txt

# 2. Create patch for specific hunk (e.g., lines 45-67)
git diff -U0 file.go | awk '/^@@ -45,/,/^@@ -68,/' > hunk.patch

# 3. Or extract by function name
git diff file.go | awk '/^@@.*funcName/,/^@@/' > function.patch

# 4. Apply the specific hunk
git apply --cached function.patch

# This mimics what GUIs do when you click checkboxes next to hunks
```

## Dependency Detection
```bash
# Analyze which files must be committed together
check_dependencies() {
    # Find imports/requires for staged files
    for file in $(git diff --cached --name-only); do
        # Check what this file imports
        grep -E "import|require|include" "$file" 2>/dev/null | extract_deps
        
        # Check what imports this file  
        grep -r "$(basename $file)" --include="*.go" --include="*.js" --include="*.py" .
    done
}

# If function signatures changed, find all callers
git diff --cached | grep "^-.*func\|^-.*function\|^-.*def" | while read line; do
    func_name=$(echo $line | extract_function_name)
    echo "Function $func_name changed, checking callers..."
    grep -r "$func_name" --include="*.go" --include="*.js" --include="*.py" .
done
```

## Special File Handling
```bash
# Classify files by type
classify_files() {
    # Lock files - commit separately
    git diff --name-only | grep -E "package-lock.json|yarn.lock|go.sum|Cargo.lock" > lock_files.txt
    
    # Generated files - commit separately  
    git diff --name-only | grep -E "\.pb\.go|\.generated\.|_gen\.go" > generated_files.txt
    
    # Test files - can often be separate
    git diff --name-only | grep -E "_test\.|\.test\.|\.spec\." > test_files.txt
}

# Commit lock files first
if [ -s lock_files.txt ]; then
    cat lock_files.txt | xargs git add
    git commit -m "deps: update dependency lock files"
fi
```

## Enhanced Build & Test Verification
```bash
# Detect appropriate verification commands
detect_verification_commands() {
    # Check CLAUDE.md first for project-specific commands
    if grep -q "commit-checks:" CLAUDE.md 2>/dev/null; then
        grep "commit-checks:" CLAUDE.md | cut -d: -f2-
        return
    fi
    
    # Otherwise detect by project type
    if [ -f "go.mod" ]; then
        echo "go build ./... && go test -short ./..."
    elif [ -f "package.json" ]; then
        # Check available scripts
        if grep -q "\"typecheck\":" package.json; then
            echo "npm run typecheck && npm run build"
        else
            echo "npm run build && npm test"
        fi
    elif [ -f "Cargo.toml" ]; then
        echo "cargo check && cargo test --lib"
    elif [ -f "Makefile" ]; then
        echo "make test"
    fi
}

# Run verification after each commit
VERIFY_CMD=$(detect_verification_commands)
if [ -n "$VERIFY_CMD" ]; then
    echo "Running verification: $VERIFY_CMD"
    eval $VERIFY_CMD || echo "⚠️  Verification failed - commit anyway? (build may be broken)"
fi
```

## Commit State Tracking (for recovery)
```bash
# Save state for potential rollback
save_commit_progress() {
    # Track what we've committed so far
    echo "$(date): Commit $1 of $2 - $(git log -1 --format=%s)" >> .git/incremental_commits.log
    
    # Save the original full diff
    if [ ! -f .git/original_changes.patch ]; then
        git diff HEAD > .git/original_changes.patch
    fi
}

# If something goes wrong, offer recovery
recover_from_partial_commits() {
    echo "Found incomplete incremental commit session"
    echo "Options:"
    echo "1. Continue from where you left off"
    echo "2. Roll back all commits and start over"
    echo "3. Keep commits and exit"
}
```

## Commit Message Format
```
subsystem: Brief summary (imperative mood, <50 chars)

In this commit, we [explain the change in natural prose, focusing
on the "why" more than the "what"]. This change improves [aspect]
by [approach/method].

[If needed, additional context about trade-offs, alternatives
considered, or implementation details. Keep prose natural and
avoid excessive bullet points.]

[For bug fixes, explain the root cause and the fix approach]
```

## Subsystem Prefix Guidelines
- Single package: `package: description`
- Multiple packages: `pkg1+pkg2: description` or `multi: description`
- Project-wide: `multi: description`
- Build/CI: `build: description` or `ci: description`
- Documentation: `docs: description`
- Tests: `test: description` or `package/test: description`

## Execution Flow
1. Analyze all changes comprehensively
2. Create a mental model of the commit sequence
3. For each planned commit:
   - Extract relevant changes (using git apply or selective staging)
   - Craft detailed commit message
   - Verify it builds
   - Commit with proper message
4. Continue until all changes are committed

## Special Considerations
- If changes are deeply intertwined, explain why they must be committed together
- For generated files, commit them separately with clear indication
- For vendored dependencies, use dedicated commit
- Skip CI for trivial changes with `[skip ci]` suffix

Focus area: $ARGUMENTS
