# Example Usage of /incremental-commit

## Basic Usage
```
/incremental-commit
```
This analyzes all changes and creates appropriate incremental commits.

## Focused Usage
```
/incremental-commit lnwallet/
```
This focuses on changes in the lnwallet package.

## Example Scenario

Suppose you have changes across multiple files:
- Fixed a bug in `lnwallet/channel.go`
- Added new validation in `lnwallet/validation.go`
- Refactored test helpers in `lnwallet/test_utils.go`
- Updated integration in `htlcswitch/link.go`

The command would create commits like:

### Commit 1: Bug Fix
```
lnwallet: fix race condition in channel state update

In this commit, we address a race condition that could occur when
concurrent goroutines attempted to update the channel state. The
issue manifested when multiple HTLCs were being processed
simultaneously, leading to inconsistent state.

The fix introduces proper mutex locking around the critical
section where channel state is modified. We ensure that all
state transitions are atomic and that no partial updates can
be observed by concurrent readers.

```

### Commit 2: Refactoring
```
lnwallet/test: refactor test utilities for clarity

In this commit, we reorganize the test helper functions to improve
code reuse and readability. The previous implementation had
duplicated logic across multiple test files, making maintenance
difficult.

We consolidate common test fixtures into dedicated helper
functions and introduce a consistent naming convention. This
refactoring does not change any functionality but makes the
test suite more maintainable.

```

### Commit 3: New Feature
```
lnwallet: add enhanced validation for commitment transactions

In this commit, we introduce additional validation checks for
commitment transactions to prevent edge cases that could lead
to invalid states. The validation ensures that all HTLCs are
properly accounted for and that fee calculations remain within
acceptable bounds.

The new validation logic catches potential issues early in the
transaction construction process, providing clearer error
messages and preventing downstream failures.

```

### Commit 4: Integration
```
htlcswitch+lnwallet: integrate new validation into forwarding logic

In this commit, we wire up the new validation functionality from
lnwallet into the HTLC forwarding logic. The integration ensures
that all forwarded payments undergo the enhanced validation checks
before being accepted.

This completes the validation feature by ensuring it is actively
used in the payment flow. We update the link logic to handle the
new validation errors appropriately, providing meaningful feedback
to callers.

```

## Advanced git apply Example

For complex changes that need careful separation:

```bash
# 1. Create a diff of all changes
git diff > all-changes.patch

# 2. Extract specific hunks for first commit
# Create a new patch with only bug fix hunks
cat all-changes.patch | sed -n '/channel\.go/,/diff --git/p' > bugfix.patch

# 3. Apply only the bug fix
git apply --cached bugfix.patch

# 4. Commit the bug fix
git commit -m "$(cat <<'EOF'
lnwallet: fix race condition in channel state update

In this commit, we address a race condition...
EOF
)"

# 5. Continue with remaining changes
```