---
description: "Create a GitHub issue from a PR comment to track follow-up work"
argument-hint: "<github-comment-url>"
allowed-tools:
  - Bash(gh:*)
  - Bash(git:*)
  - WebFetch
  - Read
  - Grep
  - Glob
  - AskUserQuestion
---

# Issue From Comment

Create a GitHub issue from a PR comment. The issue will include the original comment context, background on what's being proposed, and a brief implementation approach for a follow-up PR.

## Phase 1: Parse Comment URL

Extract the repository and comment information from the provided URL: $ARGUMENTS

GitHub comment URLs have these formats:
- Review comment: `https://github.com/owner/repo/pull/123#discussion_r1234567890`
- PR/Issue comment: `https://github.com/owner/repo/pull/123#issuecomment-1234567890`
- Issue comment: `https://github.com/owner/repo/issues/123#issuecomment-1234567890`

Parse to extract:
- `owner`: Repository owner
- `repo`: Repository name
- `pr_number` or `issue_number`: The PR or issue number
- `comment_type`: Either `discussion` (review comment) or `issuecomment`
- `comment_id`: The numeric comment ID

## Phase 2: Fetch Comment and PR Context

### Fetch the Comment

Based on comment type, use the appropriate API:

```bash
# For review comments (discussion_r*)
gh api repos/{owner}/{repo}/pulls/comments/{comment_id}

# For issue/PR comments (issuecomment-*)
gh api repos/{owner}/{repo}/issues/comments/{comment_id}
```

### Fetch PR/Issue Context

Get the PR or issue details to understand the broader context:

```bash
# Get PR details
gh pr view {pr_number} --repo {owner}/{repo} --json title,body,url,files,commits,headRefName

# Get the PR diff to understand what changed
gh pr diff {pr_number} --repo {owner}/{repo}
```

If the comment references specific code, read those files to understand the context better.

## Phase 3: Analyze the Comment

Analyze the comment to understand:

1. **What is being proposed or suggested?**
   - Is it a feature request?
   - A refactoring suggestion?
   - A bug concern?
   - An alternative approach?
   - A follow-up improvement?

2. **Why was this suggestion made?**
   - What problem does it solve?
   - What improvement does it offer?

3. **What's the scope?**
   - Is this a small change?
   - Does it require broader architectural changes?

If the comment or its intent is unclear, use `AskUserQuestion` to clarify:
- What aspect of the comment should the issue focus on?
- Is there additional context to include?

## Phase 4: Research Implementation Approach

Before creating the issue, briefly explore the codebase to understand:

1. What files/components would be affected
2. Are there existing patterns to follow
3. Any obvious blockers or dependencies

Keep this research focused - the goal is a brief, actionable proposal, not a full implementation plan.

## Phase 5: Create the Issue

Create the issue using `gh issue create`:

```bash
gh issue create --repo {owner}/{repo} --title "{title}" --body "$(cat <<'EOF'
## Background

This issue tracks a follow-up suggestion from PR #{pr_number}: {pr_title}

**Original Comment**: {comment_url}
**Author**: @{comment_author}

> {quoted_comment_body}

## Context

{Explain the PR context - what the PR was doing and why this comment was made}

## Proposed Change

{Summarize what the commenter is suggesting or requesting}

## Implementation Approach

{Brief proposal for how to approach this in a follow-up PR:
- Key files to modify
- High-level approach
- Any patterns to follow
- Potential considerations}

## Related

- PR #{pr_number}: {pr_url}
EOF
)"
```

### Link Back to Original Comment

After creating the issue, post a reply on the original comment to link the follow-up issue:

```bash
# For review comments (discussion_r*) - reply to the thread
gh api repos/{owner}/{repo}/pulls/comments/{comment_id}/replies -f body="Created #{issue_number} to track this. Thanks for the suggestion!"

# For issue/PR comments (issuecomment-*) - post a new comment referencing it
gh api repos/{owner}/{repo}/issues/{pr_number}/comments -f body="@{comment_author} Created #{issue_number} to track your suggestion above."
```

### Issue Title Guidelines

Create a concise, actionable title that:
- Starts with an action verb (Add, Improve, Refactor, Fix, etc.)
- Describes the outcome, not the source
- Does NOT mention "from comment" or "follow-up"

Examples:
- "Add validation for edge case in payment processing"
- "Refactor config loading to support hot reload"
- "Improve error messages in authentication flow"

## Phase 6: Summary

After creating the issue, provide:

1. **Issue URL**: The link to the newly created issue
2. **Summary**: Brief recap of what the issue tracks
3. **Suggested labels**: Recommend any labels that might be appropriate (enhancement, refactor, etc.)

Remind the user they can:
- Add labels: `gh issue edit {issue_number} --add-label "enhancement"`
- Assign someone: `gh issue edit {issue_number} --add-assignee "@username"`
- Link to project: `gh issue edit {issue_number} --add-project "Project Name"`

## Important Notes

1. **Preserve attribution**: Always credit the original commenter and link to the source comment.

2. **Keep it actionable**: The issue should be self-contained enough that someone unfamiliar with the PR can understand what needs to be done.

3. **Brief implementation guidance**: The approach section should be a starting point, not a comprehensive plan. Use `/issue-plan` for detailed planning later.

4. **Don't over-scope**: Focus on what the comment specifically suggests. If it's part of a larger theme, note that but keep the issue focused.

5. **Ask when uncertain**: If the comment is vague or could be interpreted multiple ways, use `AskUserQuestion` to clarify before creating the issue.
