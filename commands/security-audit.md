---
description: "Perform comprehensive security audit with exploit development"
argument-hint: "<path_or_repo> [--focus=p2p|rpc|crypto|all] [--exploit-dev]"
allowed-tools:
  - Task
  - Bash
  - Read
  - Write
  - Edit
  - Grep
  - Glob
  - WebSearch
---

# Security Audit Request

Perform a comprehensive penetration test and security audit of: $ARGUMENTS

## Audit Scope

1. **Offensive Testing**
   - Develop working exploits for discovered vulnerabilities
   - Test attack chains and impact
   - Measure exploit reliability

2. **Attack Vectors to Test**
   - P2P network attacks (Eclipse, Sybil, routing manipulation)
   - RPC/API exploitation (auth bypass, injection, DoS)
   - Consensus attacks (if applicable)
   - Resource exhaustion vectors
   - Timing and race conditions
   - Cryptographic weaknesses

3. **Deliverables**
   - Full security assessment report
   - Working exploit code (in ~/.claude/security/exploits/)
   - Defensive recommendations with code
   - Attack simulation results
   - Risk prioritization matrix

Use the security-auditor agent to conduct thorough offensive testing while providing actionable defensive strategies.