---
description: "Perform comprehensive code review with detailed markdown report generation"
argument-hint: "<owner/repo#PR_NUMBER> or <PR_NUMBER> [--focus=security|performance|architecture|all]"
allowed-tools:
  - Task
  - Bash
  - Read
  - Write
  - Edit
  - MultiEdit
  - Grep
  - Glob
  - WebFetch
  - WebSearch
  - TodoWrite
---

# Code Review Request

I need you to perform a comprehensive code review using the code-reviewer agent. 

**Target**: $ARGUMENTS

## Instructions:

1. Parse the arguments to extract:
   - Repository owner and name (if provided as owner/repo#PR)
   - PR number
   - Focus area (if --focus flag is provided)

2. Use the code-reviewer agent to:
   - Create an incremental markdown review file
   - Perform all automated checks in parallel
   - Analyze each file systematically
   - Deploy parallel sub-agents for specialized analysis
   - Generate a comprehensive review report

3. The review should include:
   - Automated test results
   - Security vulnerability assessment
   - Performance impact analysis
   - Breaking change detection
   - Test coverage analysis
   - Actionable recommendations with code snippets

4. Save the final review as: `{repo_owner}_{repo_name}_PR_{pr_number}_review.md`

Please ensure the review is thorough, actionable, and provides clear guidance for the PR author.