---
description: "Perform focused review on specific aspects of a PR"
argument-hint: "<PR_NUMBER> <aspect> [file_pattern]"
allowed-tools:
  - Task
  - Bash
  - Read
  - Grep
  - Glob
---

# Focused PR Review

Perform a targeted review of PR #$ARGUMENTS

## Supported Review Aspects:

### `tests`
- Test coverage adequacy
- Test quality and edge cases
- Missing test scenarios
- Test performance

### `api`
- API compatibility
- Breaking changes
- Documentation completeness
- Error responses

### `perf`
- Performance implications
- Benchmark results
- Memory usage
- CPU hotspots

### `concurrency`
- Thread safety
- Race conditions
- Deadlock risks
- Channel operations

### `errors`
- Error handling completeness
- Error message quality
- Panic conditions
- Recovery mechanisms

### `crypto`
- Cryptographic correctness
- Key management
- Random number usage
- Side-channel risks

## Example Usage:
- `/focused-review 1234 tests`
- `/focused-review 5678 api internal/api/*.go`
- `/focused-review 9012 perf`

Generate a focused report on just the requested aspect, allowing for quick targeted reviews when specific concerns arise.