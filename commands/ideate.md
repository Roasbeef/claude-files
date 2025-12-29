---
description: Interview-driven planning from rough ideas, issues, or sketches
argument-hint: <@file|#issue|text>
allowed-tools: [AskUserQuestion, Task, Bash, Grep, Glob, Read, Write, EnterPlanMode, ExitPlanMode, LS, WebFetch]
---

# /ideate - Interview-Driven Planning

Transform rough ideas into detailed implementation plans through structured interviewing.

## Phase 1: Parse Input and Enter Plan Mode

First, parse the input arguments: $ARGUMENTS

**Input Type Detection:**
1. If input starts with `@` → file path (content already loaded via @$1 syntax)
2. If input matches issue pattern → GitHub issue:
   - `#123` or `123` (bare number)
   - `owner/repo#123`
   - Full GitHub URL containing `/issues/`
3. Otherwise → raw text input

**For GitHub Issues:**
Fetch full context using `gh` CLI:
```bash
gh issue view <issue-number> --json title,body,comments,labels,assignees,url
gh api repos/{owner}/{repo}/issues/{issue-number}/timeline
```

**For Missing Files:**
If the referenced file doesn't exist, use AskUserQuestion to offer:
- Option 1: Type the idea directly
- Option 2: Specify a different file path

**Immediately invoke EnterPlanMode** after parsing input. The entire interview happens within plan mode.

## Phase 2: Gather Codebase Context

Before starting the interview, gather context about the current codebase (if applicable):

1. Use the Explore agent or quick searches to identify:
   - Primary language(s) and frameworks
   - Architecture patterns (monolith, microservices, etc.)
   - Concurrency model (channels, mutexes, async/await)
   - Error handling conventions
   - Testing patterns

2. Store these observations mentally to inform question generation.

## Phase 3: Dynamic Interview Loop

Conduct an in-depth interview to fully understand the feature/change being planned.

### Interview Guidelines

**Question Batching:**
- Ask 3-4 questions per AskUserQuestion call
- Group related questions together
- Use clear, short headers (max 12 chars) for each question

**Question Categories to Cover:**
Track coverage across these areas, weighting by relevance to the input:

1. **Technical Implementation**
   - Data structures and storage
   - API design and interfaces
   - State management approach

2. **Architecture & Concurrency**
   - Integration with existing components
   - Lock ordering and synchronization
   - Channel vs mutex decisions
   - Goroutine lifecycle management

3. **Constraints & Requirements**
   - Compatibility requirements (versions, platforms)
   - Performance requirements
   - Non-functional requirements

4. **Edge Cases & Error Handling**
   - Failure modes and recovery
   - Partial/inconsistent states
   - Race conditions and timing issues

5. **Dependencies & Integration**
   - External APIs or services
   - Data sources
   - Team/component coordination

6. **Security Considerations** (if applicable)
   - Authentication/authorization
   - Data validation
   - Attack surface changes

**Question Quality Rules:**
- Avoid obvious questions ("What do you want to build?")
- Reference codebase patterns when relevant ("The codebase uses X pattern for Y, would you want to follow that?")
- Ask about tradeoffs, not just preferences
- Probe non-obvious implications

**Confidence Assessment:**
After each round of answers, internally assess:
- Have critical areas been covered?
- Are there contradictions to resolve?
- Is there enough specificity to write actionable tasks?

Target ~80% confidence across relevant areas before proceeding.

**Session Integration:**
If an active session exists in `~/.claude/.sessions/active/`:
- Log key decisions via `/session-log --decision`
- State survives context compaction through session system

### Risk Detection

If the discussion reveals high-risk changes:
- Money/payment handling
- Consensus-critical code
- Security-critical paths
- Data migration/loss potential

Surface a risk acknowledgment question:
```
"This involves [specific risk]. The plan will document mitigations, but I want to confirm you're aware of the stakes. Proceed with planning?"
```
User must acknowledge before plan generation.

## Phase 4: Agent Enrichment

Once confident in requirements, launch specialized agents IN PARALLEL to enrich the plan:

**1. Code Analysis Agent (Explore type):**
- Identify specific files that need modification
- Map dependencies and impact areas
- Find existing patterns to follow or extend

**2. Security Auditor (if applicable):**
- Only for features involving auth, payments, or sensitive data
- Identify security implications
- Note required validations

**3. Architecture Review (for significant changes):**
- Evaluate architectural fit
- Surface scalability considerations
- Identify potential technical debt

Collect agent findings before generating the final plan.

## Phase 5: Generate Plan Document

Write the final plan to `~/.claude/plans/ideate-{descriptive-name}.md`:

```markdown
# Implementation Plan: {Title Derived from Input}

**Generated**: {YYYY-MM-DD HH:MM}
**Input type**: {file|issue|text}
**Source**: {file path | issue URL | "direct input"}

## Summary

{2-3 sentence summary of what will be built and why}

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| {Area 1} | {Choice made} | {Why, from interview} |
| {Area 2} | {Choice made} | {Why} |
| ... | ... | ... |

## Technical Approach

{High-level technical approach, informed by codebase analysis and interview}

{Include relevant existing patterns to follow}

## Implementation Tasks

### Phase 1: {Phase Name}
- [ ] {Specific task with file path if known}
- [ ] {Task}
...

### Phase 2: {Phase Name}
- [ ] {Task}
...

{Continue phases as needed}

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| {Identified risk} | High/Med/Low | {Mitigation strategy} |
| ... | ... | ... |

## Testing Strategy

{Testing approach derived from edge case discussion}

- [ ] {Specific test scenario}
- [ ] {Test scenario}

## Files to Modify

- `path/to/file.ext` - {What changes needed}
- ...

## Notes for Implementation

{Additional context, gotchas, or considerations surfaced during interview}

{Any agent findings relevant to implementation}
```

## Phase 6: Exit Plan Mode

After writing the plan file:
1. Provide a brief summary to the user
2. Call `ExitPlanMode` to signal planning is complete
3. User can then approve and begin implementation

## Example Flows

### From File
```
User: /ideate @rough-sketch.md
→ Parse as file input, content pre-loaded
→ Enter plan mode
→ Interview based on sketch content
→ Generate detailed plan
```

### From Issue
```
User: /ideate #456
→ Fetch issue via gh cli
→ Enter plan mode
→ Interview with fresh perspective (even if issue has discussion)
→ Generate plan linked to issue
```

### From Text
```
User: /ideate "Add ability to query historical channel states"
→ Parse as raw text
→ Enter plan mode
→ Interview to flesh out requirements
→ Generate comprehensive plan
```

## Important Notes

- The interview is mandatory - there is no quick/skip mode
- Questions should be non-obvious and probe for depth
- Always reference existing codebase patterns when relevant
- Risk acknowledgment is required for high-stakes changes
- Plan must be actionable and self-contained for implementation
