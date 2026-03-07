---
description: "Multi-agent code review orchestrator with Trail of Bits security analysis"
argument-hint: "<owner/repo#PR_NUMBER> or <PR_NUMBER> [--focus=security|performance|architecture|all] [--security-depth=standard|deep|full]"
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

# Multi-Agent Code Review Orchestrator

You are a review orchestrator. Your job is to dispatch specialized agents in
parallel, collect their findings, and compile a unified review report.

**Target**: $ARGUMENTS

## Phase 1: Context Gathering

Parse arguments and collect PR context before dispatching agents.

1. **Parse arguments** to extract:
   - Repository owner and name (if provided as `owner/repo#PR`)
   - PR number
   - `--focus` area: `security`, `performance`, `architecture`, or `all` (default: `all`)
   - `--security-depth`: `standard`, `deep`, or `full` (default: `deep`)

2. **Fetch PR metadata** using `gh pr view`:
   ```bash
   gh pr view {pr_number} --json number,title,author,baseRefName,headRefName,files,additions,deletions,body
   ```

3. **Get the diff** for agent consumption:
   ```bash
   gh pr diff {pr_number}
   ```

4. **Get changed file list**:
   ```bash
   gh pr view {pr_number} --json files --jq '.files[].path'
   ```

5. **Classify changed files** by scanning filenames and diff content for:
   - **Crypto/Auth**: files touching keys, signatures, hashing, authentication
   - **Consensus/Protocol**: files referencing BIPs, BOLTs, validation rules, chain logic
   - **API/Config**: files defining public interfaces, configuration schemas, RPC endpoints
   - **Value Transfer**: files handling amounts, fees, balances, UTXOs, HTLCs, channels

   Store these classifications -- they determine which conditional agents to launch.

6. **Create output directory**:
   ```bash
   mkdir -p .reviews
   ```

## Phase 2: Parallel Agent Dispatch

Launch agents using the **Task tool**. ALL agents (both Tier 1 and Tier 2)
must be launched in a **single message** with multiple Task tool calls so they
run in parallel and there is no gap for the stop hook to intercept between
dispatch and result collection.

**CRITICAL**: Issue ALL Task tool calls for ALL tiers in ONE message. Do NOT
yield control between launching agents and reading results -- the stop hook
will fire in any gap and block the orchestrator. After this single dispatch
message, immediately proceed to Phase 3 to read agent outputs.

### Tier 1: Always-Run Agents

These agents ALWAYS run (when `--security-depth` is `deep` or `full`).
If `--security-depth=standard`, only launch Agent 1 (code-reviewer).

Launch all applicable agents (Tier 1 AND Tier 2) in a **single message**:

**Agent 1: Code Quality Review** (`subagent_type: code-reviewer`)
```
Prompt: Review PR #{pr_number} in {owner}/{repo}.

Focus: {focus_area}

PR Title: {title}
PR Description: {body}
Base Branch: {base_branch}
Changed Files: {file_list}

Perform your full review methodology (Phases 0 through 7) focusing on code
quality, correctness, Go patterns, test quality, breaking changes, and
maintainability. Do NOT spawn sub-agents for security analysis -- the
orchestrator handles that separately.

Write your findings to the review file at:
.reviews/{owner}_{repo}_PR_{pr_number}_review.md

Classify each finding with severity: Critical (C), High (H), Medium (M),
Low (L), or Informational (I). Use format: {severity}-{number}
(e.g., H-1, M-2, I-3).
```

**Agent 2: Offensive Security Audit** (`subagent_type: security-auditor`)
```
Prompt: Perform a security audit of PR #{pr_number} in {owner}/{repo}.

PR diff:
{diff_content}

Changed files: {file_list}

Focus on:
- DoS vectors and resource exhaustion
- Fund loss and value transfer bugs
- Race conditions and concurrency issues
- Panic conditions reachable from external input
- Consensus implications (chain split, re-org safety)
- P2P attack vectors (eclipse, Sybil, amplification)
- Cryptographic misuse

Develop proof-of-concept exploits for any vulnerabilities found.
Classify each finding: Critical (C), High (H), Medium (M), Low (L),
Informational (I). Use format: {severity}-{number}.
```

**Agent 3: Differential Security Review** (`subagent_type: general-purpose`)
```
Prompt: You are performing a Trail of Bits-style differential security
review. Follow the methodology from the `differential-review` skill.

PR #{pr_number} in {owner}/{repo}.
Base branch: {base_branch}

Changed files: {file_list}

Execute the differential-review workflow:
1. Intake & Triage: Risk-classify each changed file
2. Changed Code Analysis: Use git blame on removed/modified lines to
   understand history and detect regressions
3. Test Coverage Analysis: Identify test gaps for modified code paths
4. Blast Radius Analysis: Count transitive callers of changed functions
   to quantify impact
5. Deep Context Analysis: Apply Five Whys to understand root cause of
   changes
6. Adversarial Analysis: Model attacker scenarios for HIGH risk changes
7. Report: Generate findings with severity classifications

Classify each finding: Critical (C), High (H), Medium (M), Low (L),
Informational (I). Use format: {severity}-{number}.
```

### Tier 2: Conditional Agents

Launch these based on Phase 1 file classifications. When `--security-depth=full`,
launch ALL Tier 2 agents unconditionally. Otherwise, only launch agents whose
trigger conditions are met.

Include all applicable Tier 2 agents in the **same message** as Tier 1 (single parallel dispatch).

**Agent 4: Deep Function Analysis** (`subagent_type: audit-context-building:function-analyzer`)
- **Trigger**: Changed files classified as Crypto/Auth, Consensus/Protocol,
  or Value Transfer; OR `--focus=security`; OR `--security-depth=full`
```
Prompt: Perform ultra-granular analysis of the critical functions modified
in PR #{pr_number} in {owner}/{repo}.

Focus on these files (the highest-risk changed files):
{critical_file_list}

Follow the audit-context-building methodology:
- Line-by-line semantic analysis of each modified function
- Apply First Principles, 5 Whys, and 5 Hows at micro scale
- Map invariants, assumptions, and trust boundaries
- Track cross-function data flows with full context propagation
- Zero speculation: every claim must cite exact line numbers

For each function produce: Purpose, Inputs/Assumptions, Outputs/Effects,
Block-by-Block Analysis, Cross-Function Dependencies, Risk Considerations.

Classify findings: Critical (C), High (H), Medium (M), Low (L),
Informational (I).
```

**Agent 5: Spec Compliance Check** (`subagent_type: spec-to-code-compliance:spec-compliance-checker`)
- **Trigger**: Changed files classified as Consensus/Protocol (references
  BIPs, BOLTs, or protocol-level logic); OR `--focus=architecture`;
  OR `--security-depth=full`
```
Prompt: Verify specification-to-code compliance for PR #{pr_number}
in {owner}/{repo}.

Changed files touching protocol code: {protocol_file_list}

Follow the spec-to-code-compliance methodology:
1. Discover spec sources (BIPs, BOLTs, design docs in the repo)
2. Extract spec intent into structured format
3. Analyze code behavior line-by-line
4. Map spec items to code with match types:
   full_match, partial_match, mismatch, missing_in_code,
   code_stronger_than_spec, code_weaker_than_spec
5. Classify divergences by severity

Anti-hallucination: if spec is silent, classify as UNDOCUMENTED.
If code adds behavior, classify as UNDOCUMENTED CODE PATH.
```

**Agent 6: API Safety & Insecure Defaults** (`subagent_type: general-purpose`)
- **Trigger**: Changed files classified as API/Config (introduces or modifies
  public interfaces, config schemas, RPC endpoints); OR `--security-depth=full`
```
Prompt: Analyze the API surfaces and configuration defaults in
PR #{pr_number} in {owner}/{repo}.

Changed API/config files: {api_file_list}

Perform two analyses:

1. SHARP EDGES (from the sharp-edges skill):
   Model three adversaries against the changed APIs:
   - Scoundrel: Malicious developer trying to exploit the API
   - Lazy Developer: Copy-pasting examples without reading docs
   - Confused Developer: Swapping parameters or misunderstanding semantics
   Check for: Algorithm Selection issues, Dangerous Defaults, Primitive vs
   Semantic API confusion, Configuration Cliffs, Silent Failures, and
   Stringly-Typed Security patterns.

2. INSECURE DEFAULTS (from the insecure-defaults skill):
   Scan for: Hardcoded fallback secrets, default credentials, weak crypto
   defaults, permissive access control (CORS *, public by default), debug
   features left enabled, fail-open vs fail-secure behavior.

Classify each finding: Critical (C), High (H), Medium (M), Low (L),
Informational (I).
```

## Phase 3: Result Compilation

After ALL agents complete, read their outputs and compile results.

### 3a. Collect Findings
For each agent, extract:
- Agent name and role
- Finding count by severity
- Individual findings with: ID, severity, title, description, file:line, fix

### 3b. Deduplicate
When multiple agents flag the same issue:
- Keep the finding with the most detail (PoC exploit > description-only)
- Note which agents agree (e.g., "Confirmed by: security-auditor, differential-review")
- If agents disagree on severity, escalate to the higher severity and note both

### 3c. Cross-Reference
Merge complementary findings into stronger combined findings:
- security-auditor PoC exploit + differential-review blast radius = stronger finding
- code-reviewer pattern violation + sharp-edges footgun analysis = richer context
- function-analyzer invariant violation + spec-compliance divergence = spec bug

## Phase 4: Unified Report Generation

Write the final report to `.reviews/{owner}_{repo}_PR_{pr_number}_review.md`.

### Report Structure:

```markdown
# Code Review: {owner}/{repo} PR #{pr_number}

**Title**: {pr_title}
**Author**: {author}
**Date**: {date}
**Base Branch**: {base_branch}
**Files Changed**: {count}
**Lines**: +{additions} -{deletions}
**Security Depth**: {standard|deep|full}
**Agents Deployed**: {count}

---

## Agent Summary

| # | Agent | Role | Findings |
|---|-------|------|----------|
| 1 | code-reviewer | Code quality & patterns | C-{n}, H-{n}, M-{n}, L-{n}, I-{n} |
| 2 | security-auditor | Offensive security | C-{n}, H-{n}, M-{n}, L-{n}, I-{n} |
| 3 | differential-review (ToB) | Diff security & blast radius | C-{n}, H-{n}, M-{n}, L-{n}, I-{n} |
| 4 | function-analyzer (ToB) | Deep function analysis | ... (if run) |
| 5 | spec-compliance (ToB) | BIP/BOLT compliance | ... (if run) |
| 6 | sharp-edges + insecure-defaults (ToB) | API safety | ... (if run) |

---

## Critical Findings
{All C-severity findings, with source agent(s) tagged}

## High Findings
{All H-severity findings}

## Medium Findings
{All M-severity findings}

## Low Findings
{All L-severity findings}

## Informational
{All I-severity findings}

---

## Specialized Analysis

### BIP/BOLT Compliance (if spec-compliance ran)
{Compliance matrix and divergence findings}

### API Safety Report (if sharp-edges ran)
{Footgun analysis and insecure default findings}

### Property-Based Testing Recommendations
{Suggested property tests based on code patterns observed}

---

## Quality Scorecard

| Aspect | Score | Notes |
|--------|-------|-------|
| Correctness | /10 | |
| Security | /10 | Combined: code-reviewer + security-auditor + ToB |
| Performance | /10 | |
| Testing | /10 | |
| Maintainability | /10 | |
| Documentation | /10 | |
| Design | /10 | |

**Overall Grade**: {F|D|C|B|A}

---

## Executive Summary

### Verdict: {REJECT | MAJOR_REWORK_REQUIRED | MINOR_FIXES_NEEDED | APPROVED_WITH_CONDITIONS | APPROVED}

### Blockers ({count})
{List of must-fix items before merge}

### Recommended Next Steps
1. {Most critical action}
2. {Second priority}
3. {Third priority}
```

## Important Notes

- **CRITICAL: Single-message dispatch.** Launch ALL agents (Tier 1 AND Tier 2)
  in ONE message with multiple Task tool calls. The stop hook fires on every
  turn boundary -- if you yield control between dispatch and result reading,
  the hook will block the orchestrator. Determine Tier 2 triggers during
  Phase 1, then issue every Task call in a single turn.
- After the dispatch message completes, immediately proceed to Phase 3 to
  read agent outputs and compile the report. Do not yield between phases.
- The code-reviewer agent handles its own review file writing. Read its
  output after it completes and incorporate into the unified report.
- When `--security-depth=standard`, skip all security agents and just run
  the code-reviewer alone. This is the fast path for low-risk PRs.
- The stop hook SHOULD fire after the final report is written -- that is the
  correct time, since the orchestrator is done and should listen for mail
  (e.g., review feedback from the user or other agents).
