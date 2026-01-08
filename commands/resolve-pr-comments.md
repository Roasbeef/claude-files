---
description: "Work through PR review comments, resolving each with fixup commits"
argument-hint: "<pr-number>"
allowed-tools:
  - Bash(gh:*)
  - Bash(git:*)
  - Bash(hunk:*)
  - Read
  - Write
  - Edit
  - Grep
  - Glob
  - TodoWrite
  - AskUserQuestion
  - Task
---

# Resolve PR Comments

Work through all review comments on PR #$ARGUMENTS, creating fixup commits for each resolved item.

## Phase 1: Fetch PR Data

The `gh` CLI auto-detects the repository from the current directory - no manual parsing needed.

### Fetch PR Info and Comments

```bash
# Get basic PR info
gh pr view $ARGUMENTS --json number,title,url,headRefOid,commits

# Get review comments (code-specific, with file/line info)
gh api repos/{owner}/{repo}/pulls/$ARGUMENTS/comments

# Get conversation comments (general discussion)
gh api repos/{owner}/{repo}/issues/$ARGUMENTS/comments

# Get review threads with resolution status (GraphQL, for isResolved field)
gh api graphql -F number=$ARGUMENTS -F owner='{owner}' -F repo='{repo}' -f query='
query($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      reviewThreads(first: 100) {
        nodes {
          id
          isResolved
          path
          line
          comments(first: 50) {
            nodes {
              body
              author { login }
              originalCommit { oid }
            }
          }
        }
      }
    }
  }
}
'
```

The `{owner}` and `{repo}` placeholders are auto-filled by `gh` based on the current git repo.

## Phase 2: Analyze & Create TODOs

After fetching the comments, analyze them and create a TODO for each item that needs attention:

### For Review Thread Comments (Code-Specific)
- Filter to **unresolved** threads only (`isResolved: false`)
- Each TODO should include:
  - File path and line number
  - Reviewer's comment (truncated if long)
  - Original commit SHA (for fixup targeting)
  - Reviewer name

### For Conversation Comments (General Discussion)
- Review each conversation comment for actionable items
- Skip comments that are:
  - Simple acknowledgments ("Thanks!", "LGTM")
  - Questions that have already been answered in thread
  - Bot comments (CI status, etc.)
- If comment contains action items, create a TODO

### Example TODO Format
Use `TodoWrite` to track each comment:
```
content: "[path/to/file.go:42] @reviewer: Fix the nil check here"
status: "pending"
activeForm: "Fixing nil check in path/to/file.go:42"
```

## Phase 3: Work Through Each TODO

For each TODO, follow this process:

### 3.1 Read Context
```bash
# Read the file around the commented line
# Use Read tool with the file path from the comment
```

### 3.2 Understand the Request
- Parse what the reviewer is asking for
- If the request is ambiguous, use `AskUserQuestion` with options:
  - Present your interpretation(s) of the request
  - Ask which approach the user prefers
  - Include "Other" option for custom input

### 3.3 Make the Code Change
- Use `Edit` tool to make the change
- Keep changes minimal and focused on the specific comment

### 3.4 Stage with Precision (using hunk)
For surgical, line-level staging:
```bash
# See what lines changed
hunk diff path/to/file.go

# Stage only the specific lines for this fix
hunk stage path/to/file.go:START-END
```

If `hunk` is not available, fall back to file-level staging:
```bash
git add path/to/file.go
```

### 3.5 Create Fixup Commit
Create a fixup commit targeting the original commit where the reviewer commented:

```bash
# ORIGINAL_COMMIT_SHA comes from the comment's originalCommit.oid
git commit --fixup=ORIGINAL_COMMIT_SHA
```

If no original commit SHA is available (e.g., conversation comments), use a regular commit:
```bash
git commit -m "address review: brief description of change"
```

### 3.6 Mark TODO Complete
Update the TODO status to "completed" before moving to the next item.

## Phase 4: Handle Edge Cases

### Conflicting Comments
If multiple reviewers have conflicting opinions:
1. Use `AskUserQuestion` to present both viewpoints
2. Ask the user which approach to take
3. Document the decision in the commit message

### Unable to Address
If a comment cannot be addressed (e.g., out of scope, requires discussion):
1. Keep TODO as pending
2. Note in the summary why it wasn't addressed
3. The user can reply to the reviewer manually

### Large Refactoring Requests
If a comment requests significant refactoring:
1. Use `AskUserQuestion` to confirm:
   - Should this be done as part of PR review?
   - Should it be deferred to a follow-up PR?
2. If proceeding, break into multiple smaller fixup commits

## Phase 5: Summary

After processing all comments, provide a summary:

### Addressed Comments
- List each comment that was resolved
- Show the fixup commit SHA for each

### Remaining Items
- List any comments that couldn't be addressed
- Explain why (needs discussion, out of scope, etc.)

### Next Steps
Suggest the user run:
```bash
# Squash fixup commits into their target commits
git rebase -i --autosquash <base-branch>

# Force push to update the PR
git push --force-with-lease
```

## Important Notes

1. **Always ask before large changes**: If unsure how to interpret a comment, use `AskUserQuestion` rather than guessing.

2. **One fixup per comment**: Each review comment should get its own fixup commit for clean history.

3. **Preserve conversation context**: Don't assume bot comments or status messages need action.

4. **Check for replies**: If a thread has back-and-forth, read the full thread to understand if the issue was already resolved through discussion.
