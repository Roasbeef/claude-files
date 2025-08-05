---
description: "Review multiple PRs in batch with priority ordering"
argument-hint: "<owner/repo> [--label=needs-review] [--author=username] [--max=5]"
allowed-tools:
  - Task
  - Bash
  - Read
  - Write
  - Grep
  - Glob
  - WebSearch
---

# Batch PR Review

Review multiple pull requests in the repository: $ARGUMENTS

## Batch Review Process:

1. **Discover PRs**
   ```bash
   # Use gh CLI to list open PRs matching criteria
   gh pr list --repo $REPO --state open --limit $MAX
   ```

2. **Prioritize Reviews**
   Order by:
   - Security-critical changes (consensus, p2p, crypto)
   - Size (smaller PRs first for quick wins)
   - Age (older PRs get priority)
   - Author reputation/history

3. **Parallel Review Execution**
   Deploy multiple code-reviewer agents to review PRs concurrently:
   - Create separate review files for each PR
   - Generate summary dashboard
   - Identify common patterns across PRs

4. **Batch Report Generation**
   Create `batch_review_summary_{date}.md` with:
   - Overview table of all PRs reviewed
   - Critical issues requiring immediate attention
   - Common issues across multiple PRs
   - Recommended merge order
   - Team patterns and suggestions

This enables efficient review of multiple PRs while maintaining quality.