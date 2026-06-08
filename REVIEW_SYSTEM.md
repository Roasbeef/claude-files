# Bitcoin/Systems Programming Code Review System

## Overview

This enhanced code review system is specifically designed for **mission-critical Bitcoin and Lightning Network software**, with focus on:
- Systems programming (Go, Rust, C++)
- P2P network protocols
- Cryptographic implementations
- Consensus-critical code

## Review Storage

All reviews are saved to `.reviews/` directory in the repository root:
- Keeps review history organized
- Easy to gitignore if desired
- Maintains review audit trail
- Format: `.reviews/{owner}_{repo}_PR_{number}_review.md`

## Enhanced Review Areas

### 1. Bitcoin Protocol Compliance
- **BIP Compliance**: Verification of relevant BIP requirements
- **Topology Restrictions**: TRUC/v3 transactions (BIP 431), ephemeral anchors
- **Package Relay**: BIP 331 package validation rules
- **Script Validation**: Resource limits (stack depth, op count, witness size)
- **Mempool Policy**: DoS protection, RBF (BIP 125), CPFP handling
- **Taproot**: BIP 340/341/342 correctness verification

### 2. Cryptographic Safety
- **Constant-Time Operations**: Side-channel resistance for crypto primitives
- **Nonce Handling**: No reuse, proper entropy sources
- **Key Derivation**: BIP 32/44/49/84/86 compliance, hardened paths
- **Signature Safety**: Malleability prevention, grinding attack resistance
- **Schnorr/MuSig2**: Taproot signature correctness
- **Key Material**: Proper zeroing, no leaks in logs or error messages
- **Random Number Quality**: crypto/rand vs math/rand verification

### 3. P2P Protocol Security
- **State Machine Safety**: Valid protocol state transitions
- **Message Parsing**: Bounds checking, buffer overflow prevention
- **Bandwidth Amplification**: Response size limits relative to requests
- **Connection Management**: DoS via connection exhaustion prevention
- **Peer Reputation**: Correct banning logic without bypass vectors
- **Message Flood Protection**: Rate limiting and backpressure
- **Protocol Confusion**: Unexpected message ordering handling

### 4. Consensus Risk Assessment
Quantified scorecard (1-10 scale) for:
- **Chain Split Risk**: Different nodes following different chains
- **Fund Loss Potential**: Maximum financial impact
- **Re-org Safety**: Behavior during blockchain reorganizations
- **Mempool Policy Deviation**: Network mempool fragmentation risk
- **Soft Fork Compatibility**: Future soft fork resilience
- **Validation Bypass Risk**: Critical validation skipping potential

### 5. Systems Programming Rigor

#### Resource Management
- **Goroutine Leaks**: Exit conditions, context cancellation
- **Memory Management**: Bounded allocations, proper cleanup
- **File Descriptors**: Leak prevention in all paths
- **Connection Pools**: Proper configuration and cleanup

#### Panic Safety
- No panics in production paths (errors preferred)
- Recover() only at goroutine boundaries
- Bounds checking for array/slice access
- Safe type assertions with ok checks
- Nil pointer dereference prevention

#### Error Handling
- Context preservation with %w wrapping
- Actionable error messages
- All error paths tested
- No silent error ignoring

#### Observability
- Structured logging (no fmt.Printf)
- No sensitive data in logs
- Comprehensive metrics (errors, latency, resources)
- Distributed tracing support
- pprof endpoints with authentication

### 6. Testing Excellence

#### Property-Based Testing
- Transaction validation invariants
- State machine invariants
- Protocol invariants
- Using Go's native fuzzing framework

#### Fuzz Testing Targets
- P2P message deserializers
- Transaction parsing (witness, non-witness)
- Script validation engine
- Block header parsing
- Invoice decoding (BOLT-11, BOLT-12)
- TLV stream parsing

#### Integration Testing
- Re-org scenarios using lnd/btcd harness
- Network partition recovery
- Mempool attack simulations
- Resource exhaustion scenarios

#### Chaos Engineering
- Disk failures (full disk, corruption, read-only)
- Network failures (packet loss, high latency, partitions)
- Resource constraints (memory limits, CPU throttling, FD exhaustion)
- Bitcoin node failures (unavailable, stale data, RPC timeouts)

#### Test Quality Metrics
- Mutation testing scores
- Branch coverage (not just line coverage)
- Concurrency testing with -race and stress tests
- Integration coverage for key flows
- Performance regression benchmarks

### 7. Production Readiness

#### Financial Safety
- Fund loss path analysis
- Fee calculation overflow/underflow prevention
- UTXO management correctness
- Dust and change output handling
- Re-org fund tracking
- Time-lock enforcement (CSV/CLTV)

#### Network Interaction
- Peer banning for malicious behavior
- DoS protection with resource limits
- Message validation completeness
- Connection and bandwidth limits
- Network partition handling

#### Data Integrity
- Database ACID properties
- Backup/recovery procedures
- Reversible schema migrations
- Corruption detection (checksums)
- State machine validity
- Crash-resistant persistence

#### Security Hardening
- Input validation coverage
- Rate limiting (per-peer and global)
- RPC/API authentication and authorization
- Secret management (no logging)
- Secure defaults
- Dependency CVE auditing

#### Deployment Safety
- Feature flags for emergency disable
- Gradual rollout capability
- Rollback procedures
- Upgrade path testing
- Backward compatibility
- Configuration validation

#### Disaster Recovery
- Tested backup procedures
- Defined RTO/RPO
- Incident response runbooks
- Data and key recovery procedures
- Lightning channel recovery (SCB)
- Force close testing

## Using the Review System

### Running a Review

```bash
# Basic review
/code-review owner/repo#123

# Security-focused review
/security-review 123

# With specific focus area
/code-review 456 --focus=security
```

### Review Output

Reviews are comprehensive markdown files including:
1. **Executive Summary**: Overall verdict and key metrics
2. **Automated Check Results**: Build, tests, linting, race detection
3. **Bitcoin Protocol Compliance**: BIP/BOLT adherence
4. **Cryptographic Safety**: Side-channel and implementation review
5. **P2P Security Analysis**: Network attack vector assessment
6. **Consensus Risk Scorecard**: Quantified risk assessment
7. **Systems Programming Audit**: Resource management, panics, errors
8. **File-by-File Analysis**: Detailed code review with fixes
9. **Testing Recommendations**: Property-based, fuzz, integration, chaos
10. **Production Readiness**: Mission-critical checklist
11. **Actionable Fixes**: Specific code changes with examples

### Review Quality Standards

Reviews follow "senior engineer" standards:
- **Direct and Honest**: No sugar coating
- **Specific**: Line numbers, exact issues, concrete fixes
- **Actionable**: Copy-pasteable code solutions
- **Production-Focused**: "What breaks at 3am?"
- **Security-Paranoid**: Assume adversarial environment
- **Maintainability-Aware**: "Can a junior debug this?"

## Specialized Agents

The review system uses multiple specialized agents in parallel:

### code-reviewer
Senior staff engineer with 15+ years Bitcoin/Lightning experience. Provides brutally honest, comprehensive code reviews focused on preventing disasters.

### security-auditor
Offensive security researcher specializing in Bitcoin/blockchain vulnerabilities. Develops proof-of-concept exploits and provides defensive strategies.

### test-engineer (via /test-forge)
Comprehensive test generation with coverage guidance, property-based testing, and advanced fuzzing.

### architecture-archaeologist (via /code-deep-dive)
Deep architectural analysis with parallel component examination and mermaid diagram generation.

## Review Severity Levels

### 🔴 BLOCKERS (Must Fix Before Merge)
- Consensus-breaking changes
- Fund loss vectors
- Security vulnerabilities (CVSS >= 7.0)
- Data corruption risks
- Breaking API changes without migration

### 🟡 CRITICAL ISSUES (Fix This Sprint)
- High-severity bugs (production impact likely)
- Missing test coverage for complex logic
- Performance regressions >10%
- Incomplete error handling
- Resource leaks

### 🟠 CODE SMELLS (Technical Debt)
- Poor abstractions
- Copy-paste code
- Missing documentation
- Suboptimal algorithms
- Style violations

## Best Practices

1. **Review Early and Often**: Use pre-PR reviews with `/pre-pr-review`
2. **Security for Critical Code**: Always use `/security-review` for consensus/crypto/p2p changes
3. **Test Quality**: Property-based and fuzz tests for parsers/validators
4. **Integration Testing**: Use lnd/btcd harness for realistic scenarios
5. **Documentation**: Include BIP/BOLT references, threat models, runbooks
6. **Parallel Agents**: Leverage multiple specialized agents for deep analysis
7. **Follow-up**: Create tasks for deferred improvements

## Configuration

### Gitignore Reviews (Optional)

Add to `.gitignore` if you don't want reviews in git:
```
.reviews/
```

### Review Templates

Create project-specific templates in `.reviews/templates/`:
- `bitcoin-core-review.md` - Bitcoin Core specific checklist
- `lightning-review.md` - Lightning Network specific checklist
- `cryptography-review.md` - Crypto implementation specific checklist

## Integration with Task System

Reviews can automatically create tasks for follow-up work:

```bash
# After review, create tasks for critical fixes
/task-add "Fix consensus risk in block validation (from PR #123 review)"
```

## Resources

- [BIP Repository](https://github.com/bitcoin/bips)
- [BOLT Specifications](https://github.com/lightning/bolts)
- [Bitcoin Core Review Club](https://bitcoincore.reviews/)
- [Go Security Best Practices](https://github.com/guardrailsio/awesome-golang-security)

## Continuous Improvement

The review system evolves based on:
- Post-mortems from production incidents
- New attack vectors discovered
- BIP/BOLT specification updates
- Lessons learned from past reviews
- Community feedback and contributions

---

**Remember**: These reviews are designed to prevent disasters before they reach production. The goal is not to make friends, but to protect user funds and maintain system integrity.
