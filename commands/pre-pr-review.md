---
description: "Review local changes before creating a pull request"
argument-hint: "[branch_name] or [base_branch...head_branch]"
allowed-tools:
  - Task
  - Bash
  - Read
  - Write
  - Edit
  - Grep
  - Glob
  - TodoWrite
---

# Pre-PR Review

Review my local changes before I create a pull request.

**Branch/Comparison**: $ARGUMENTS

## Review Process:

1. **Analyze Local Changes**
   ```bash
   # If branch name provided, compare against main/master
   # If comparison provided (base...head), use that
   # Otherwise, review uncommitted changes
   ```

2. **Pre-flight Checks**
   - Run build and tests
   - Check linting
   - Verify no debug code or TODOs
   - Ensure proper error handling
   - Check for adequate test coverage

3. **Generate Pre-PR Report**
   Create a markdown file with:
   - Summary of changes
   - Potential issues to address
   - Missing tests or documentation
   - Breaking changes detected
   - Suggested PR title and description
   - Checklist of items to complete before PR

4. **Recommendations**
   - Code improvements before submission
   - Additional tests to write
   - Documentation to update
   - Related issues to link

Save the review as: `pre_pr_review_{branch_name}_{date}.md`

This helps ensure the PR will pass review on first submission.