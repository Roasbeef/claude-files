---
description: "Quick security-focused PR review for Bitcoin/Lightning code"
argument-hint: "<PR_NUMBER> or <owner/repo#PR_NUMBER>"
allowed-tools:
  - Task
  - Bash
  - Read
  - Write
  - Grep
  - Glob
---

# Security-Focused Code Review

Perform a rapid security assessment of PR: $ARGUMENTS

## Priority Areas:

1. **Consensus Safety**
   - Check for any consensus-breaking changes
   - Verify no modifications to validation rules
   - Ensure proper error handling in consensus code

2. **P2P Attack Vectors**
   - DoS vulnerabilities
   - Resource exhaustion risks
   - Message parsing vulnerabilities
   - Connection handling issues

3. **Financial Safety**
   - Money-losing bugs
   - HTLC timeout issues
   - Channel state inconsistencies
   - Fee calculation errors

4. **Concurrency Issues**
   - Race conditions
   - Deadlocks
   - Improper mutex usage
   - Channel operations without proper synchronization

Use both the code-reviewer and security-auditor agents in parallel to maximize coverage. Generate a security-focused review report highlighting any critical findings that could lead to fund loss or network disruption.