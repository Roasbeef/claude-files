---
description: Analyze a GitHub issue and create a comprehensive implementation plan
argument-hint: <issue-url-or-number> [repository]
allowed-tools: Task, Bash, Grep, Glob, Read, LS, WebFetch, WebSearch, Write
---

# GitHub Issue Analysis and Planning

You are about to analyze a GitHub issue and create a comprehensive implementation plan. **THINK ULTRA HARD** about this task - take your time to deeply understand the issue, its context, and all potential implications. Use **ULTRATHINK** mode when analyzing complex technical requirements or architectural decisions.

## Step 1: Fetch Issue Details

First, extract the issue information from the arguments provided: $ARGUMENTS

Parse the arguments to get:
- Issue number (e.g., #123 or just 123)
- Repository (if provided as owner/repo format, otherwise use current repo)

Use the `gh` CLI to fetch the issue details:
```bash
# If repository is provided: gh issue view <issue-number> --repo <repository> --json title,body,comments,labels,assignees,milestone,url
# If no repository: gh issue view <issue-number> --json title,body,comments,labels,assignees,milestone,url
```

Also fetch related information:
```bash
# Get linked PRs if any
gh api repos/{owner}/{repo}/issues/{issue-number}/timeline

# Get project status if linked to a project
gh issue status <issue-number>
```

## Step 2: Deep Analysis with Parallel Agents

**ULTRATHINK** about the issue requirements and spin up parallel agents to investigate different aspects. Use the Task tool to launch multiple specialized agents SIMULTANEOUSLY for maximum efficiency:

1. **Code Analysis Agent** (general-purpose):
   - Search the codebase for related code mentioned in the issue
   - Identify files and functions that will need modification
   - Understand existing patterns and conventions
   - Map out dependencies and potential impact areas

2. **Security Analysis Agent** (security-auditor):
   - Identify any security implications of the proposed changes
   - Check for potential vulnerabilities in the approach
   - Review authentication/authorization requirements
   - Analyze data handling and privacy concerns

3. **Architecture Review Agent** (general-purpose):
   - Evaluate architectural implications
   - Consider scalability and performance impacts
   - Review design patterns that should be followed
   - Identify potential technical debt or refactoring opportunities

4. **Testing Strategy Agent** (general-purpose):
   - Identify what tests need to be written
   - Determine testing approach (unit, integration, e2e)
   - Find existing test patterns to follow
   - Consider edge cases and error scenarios

Launch all agents in PARALLEL using a single Task tool invocation with multiple agent calls.

## Step 3: Create Comprehensive Plan Document

After gathering all information, create a markdown document named `issue-{issue-number}-plan.md` with the following structure:

```markdown
# Implementation Plan for Issue #{issue-number}

**Issue URL**: {full-github-url}
**Title**: {issue-title}
**Created**: {timestamp}
**Labels**: {labels}

## Issue Summary

{Provide a clear, concise summary of what the issue is asking for}

## Technical Analysis

### Current State
{Describe the current implementation/situation}

### Proposed Changes
{Detail what needs to be changed}

### Affected Components
{List all files, modules, and systems that will be modified}

## Implementation Strategy

### Phase 1: {Phase Name}
- [ ] Task 1
- [ ] Task 2
- [ ] ...

### Phase 2: {Phase Name}
- [ ] Task 1
- [ ] Task 2
- [ ] ...

{Continue with phases as needed}

## Technical Considerations

### Architecture Impact
{Findings from architecture review agent}

### Security Considerations
{Findings from security analysis agent}

### Performance Implications
{Any performance considerations}

### Dependencies
{External dependencies or blocking issues}

## Testing Plan

### Unit Tests
- {Specific unit tests to write}

### Integration Tests
- {Integration test scenarios}

### End-to-End Tests
- {E2E test scenarios}

### Edge Cases to Test
- {List of edge cases}

## Reviewer Checklist

**Important points for PR reviewers:**

- [ ] {Critical point 1 that reviewers should verify}
- [ ] {Critical point 2 that reviewers should verify}
- [ ] {Security consideration to review}
- [ ] {Performance aspect to validate}
- [ ] {Code quality standard to check}
- [ ] {Documentation requirement}

## Implementation Notes

### Code Conventions
{Relevant code style and patterns to follow}

### Potential Gotchas
{Known issues or tricky parts}

### Alternative Approaches Considered
{Other solutions that were evaluated}

## Estimated Effort

- **Development**: {time estimate}
- **Testing**: {time estimate}
- **Code Review**: {time estimate}
- **Total**: {total estimate}

## Success Criteria

- [ ] {Measurable success criterion 1}
- [ ] {Measurable success criterion 2}
- [ ] {Acceptance criteria from issue}

## Additional Context for Implementation Agent

{Any additional context, code snippets, or references that would help another agent implement this plan}

---
*This plan was generated with ultra-deep analysis and parallel investigation of all technical aspects.*
```

## Step 4: Final Validation

**ULTRATHINK** one more time to ensure:
1. The plan is complete and actionable
2. All security and architectural concerns are addressed
3. The testing strategy is comprehensive
4. The reviewer checklist covers critical points
5. The plan has enough context for handoff to another agent

Save the plan document and provide a summary to the user with the file location.

## Important Instructions

- **THINK ULTRA HARD** at every decision point
- Use **ULTRATHINK** mode for complex technical analysis
- Launch agents in PARALLEL for efficiency
- Be thorough but concise in the plan
- Include specific, actionable items
- Consider edge cases and failure modes
- Make the plan self-contained for handoff