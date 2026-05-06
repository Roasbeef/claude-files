---
name: test-refine
description: Refines an existing Go test suite — removes trivial/duplicate tests, strengthens weak assertions, reshapes tests around invariants, and closes branch-coverage gaps. Uses code-guided coverage and (when available) gremlins mutation-testing survivor data rather than relying on line coverage alone. Use when test quality is uneven, after a test-generation pass, before opening a PR, or as a quality gate on critical paths (consensus, channel state, payment flows). Triggers - "refine these tests", "tests are bloated", "tighten assertions", "remove trivial tests", "audit test quality", "/test-refine".
---

# Test Refine

A test suite can be voluminous and yet weak: tests that overlap, assert nothing meaningful, run code without checking outputs, or miss the branches that actually matter. Line coverage doesn't catch this — a 100% line-covered suite can have zero real assertions. This skill operates on an **existing** test suite and refines it.

This is *orthogonal* to test generation skills (`test-forge`, `property-based-testing`):

- **`test-forge`** generates new tests.
- **`property-based-testing`** designs PBT for new or existing functions.
- **`mutation-testing`** validates whether existing tests catch behavior changes.
- **`test-refine`** (this skill) consumes the above signals and *changes the test suite* — strengthening assertions, removing dead weight, reshaping for invariants, closing branch-coverage gaps.

## When to Use

- After a `test-forge` pass — sharpen the generated tests.
- Before opening a PR — clean up the diff so reviewers see strong tests.
- Periodically on critical packages — guard against test-quality drift.
- Following a mutation-testing run with survivors — convert each LIVED mutant into a tightened test.

## Two-Phase Workflow

The skill runs **read-only triage first**, produces a markdown report, then applies fixes only after the user reviews and approves.

### Phase A — Triage (read-only)

```bash
# Default: package in cwd. JSON intermediates land in /tmp; markdown in
# .reviews/test-refinement/<date>-<scope>.md.
~/.claude/skills/test-refine/scripts/triage.sh

# Pin scope to a single file
~/.claude/skills/test-refine/scripts/triage.sh --scope file --target ./internal/wallet/wallet_test.go

# Diff-scoped (changed test files in current branch vs main)
~/.claude/skills/test-refine/scripts/triage.sh --scope diff

# Whole repo (slow)
~/.claude/skills/test-refine/scripts/triage.sh --scope repo

# With mutation testing for the strongest signal
~/.claude/skills/test-refine/scripts/triage.sh --scope package --use-mutations
```

The triage script:

1. Resolves scope.
2. Runs `go test -cover -covermode=atomic` to capture per-function statement+branch coverage.
3. Optionally calls `~/.claude/skills/mutation-testing/scripts/unleash.sh` for `LIVED` mutant data.
4. Runs the AST analyzers (`detect-smells.go`, `detect-duplicates.go`, `domain-checks.go`).
5. Scores findings via `score.go` (composite priority — see below).
6. Renders the markdown report.

### Phase B — Batch Fix (after user approves report)

The user reviews the report. Each finding has a checkbox. The user checks boxes for the fixes they approve, then:

```bash
~/.claude/skills/test-refine/scripts/apply-fixes.sh \
    --report .reviews/test-refinement/2026-05-06-wallet.md
```

The script applies **only the checked items**:

- Strengthen assertions in place.
- Add missing branch tests.
- Reshape into invariants (rewrite using `rapid` for flagged cases).
- Remove tests **only if explicitly checked**.

After fixes, it re-runs `go test ./... -race -count=1`, appends an "After" metrics section to the same report, and surfaces the diff.

> **Safety rule**: a test is never removed unless its checkbox in the report is explicitly checked. This honors the global rule "never remove/skip tests without asking" from `~/.claude/CLAUDE.md`.

## Composite Priority Score

When a triage produces dozens of findings, ranking matters. The composite score combines three signals:

```
priority = w_risk × risk_score(file_path)
         + w_severity × severity(smell_id)
         + w_gap × branch_gap(function)
```

| Component | Range | Source |
|---|---|---|
| `risk_score` | 0.2–1.0 | File-path heuristic: `consensus|channel|commit|payment|crypto|sign|verify|wallet|htlc` → 1.0; `internal/` → 0.7; `cmd/` → 0.4; `test/` helpers → 0.2 |
| `severity` | 0.3 (L) / 0.6 (M) / 1.0 (H) | Smell catalog severity (`references/smell-catalog.md`) |
| `branch_gap` | 0.0–1.0 | Uncovered branches in the target function / total branches |

Default weights: `0.5 / 0.3 / 0.2`. Override via `--weights risk=0.6,severity=0.3,gap=0.1`.

## Smell Catalog (summary)

Full catalog with Go examples in [`references/smell-catalog.md`](references/smell-catalog.md).

| ID | Smell | Severity |
|---|---|---|
| S01 | No assertions at all | High |
| S02 | Tautological assertion (`x == x`) | High |
| S03 | Getter/setter trivial test | Medium |
| S04 | Asserts no panic only | High |
| S05 | Unchecked error from SUT | High |
| S06 | Sensitive equality (asserts on `String()`) | Medium |
| S07 | Conditional/skipped assertion | Medium |
| S08 | Duplicate test body (semantic) | Medium |
| S09 | Assertion roulette (no messages) | Low |
| S10 | Expect-the-expected (`want` derived from `got`) | High |
| S11 | Side-effect not asserted | Medium |
| S12 | Mutation-survivor zone (gremlins data required) | High |

## Domain-Specific Checks

For systems / distributed / Bitcoin / networking code, "good test" goes beyond standard assertion strength. The skill also checks four dimensions detailed in [`references/domain-checks.md`](references/domain-checks.md):

- **Concurrency**: SUT uses goroutines/channels/mutexes? Test must run with `-race`-friendly patterns and exercise concurrent calls. Sequential test of concurrent code is flagged.
- **Failure-mode**: SUT returns `error` or accepts `context.Context`? Tests must cover error path, cancellation, timeout. For network/disk SUTs, missing fault-injection tests are flagged. Inspired by [Jepsen nemesis patterns](https://github.com/jepsen-io/jepsen).
- **Property/invariant**: Detects PBT candidates — serialization roundtrips, parser/encoder pairs, normalizers, comparators. Proposes conversion to `rapid` PBT (see [`references/reshape-to-invariants.md`](references/reshape-to-invariants.md)).
- **Determinism**: `time.Now()`, `rand.*` without seeds, `os.Getenv`, goroutine-ordering assumptions in tests are flagged. Suggests injectable clocks/RNGs (DST-style).

## What Counts as Trivial (and Removable)

Findings flagged for removal candidacy (still always require user approval):

- **No assertions** (S01).
- **Getter/setter-only tests** (S03) — testing language semantics, not behavior.
- **Tautological assertions** (S02).
- **Zero-mutation-killers** (S12 special case): when gremlins data shows a test exists in the package whose removal causes no change in `test_efficacy`. The test catches no mutants the rest of the suite doesn't already catch.

## Naming Convention

Test function names must not contain underscores. Use `TestEncodeTxRoundtrip`, not `TestEncodeTx_Roundtrip`. For variants of the same logical test, use `t.Run("subtest name", func(t *testing.T) {...})`. The reshape pass enforces this — any rewrite the skill proposes uses subtests, not underscored function names. See [`references/strengthening-patterns.md`](references/strengthening-patterns.md#naming-convention-important).

## Aggressive Reshape Posture

The user opted into "aggressive — reshape tests for invariants". Examples:

- A test that asserts `Marshal(x) == fixed_bytes` for a single value → reshape into a `rapid` roundtrip property: `Unmarshal(Marshal(x)) == x` for arbitrary `x`.
- A test that asserts a single state transition → reshape into a `rapid.StateMachine` covering all transitions.
- Multiple tests covering near-identical inputs → consolidate into a table-driven test with the union of cases.

Every reshape proposal in the report shows the **original test verbatim** alongside the **proposed rewrite**. If the user rejects the reshape, no change is made.

## Critical Pitfall Guards

- **Never auto-delete**. The report's checkbox grammar (`- [ ] Remove TestX`) is the only way a test is removed. Bulk approval is not a thing.
- **Reshape passing tests of mission-critical code with care**. Phase B optionally re-runs `gremlins unleash` on touched packages to verify reshape didn't *weaken* mutation score. Pass `--verify-mutations`.
- **PBT conversion for side-effect-bearing code uses state-machine PBT**. Pure-function PBT is wrong here. Such cases are flagged separately and link to `property-based-testing/references/strategies.md`.
- **Equivalent mutants are not test gaps**. When gremlins reports `LIVED`, the AST is checked for equivalence patterns (mutated value immediately overwritten, mutation in unreachable code, associative no-op) before flagging as S12.

## Output Artifacts

The committed markdown report lives at:

```
.reviews/test-refinement/<YYYY-MM-DD>-<scope-slug>.md
```

Contents:

- **Before metrics**: test count, assertion count, statement+branch coverage %, `test_efficacy` if mutation data available.
- **Top-N findings table**: rank, file:line, smell, severity, proposed action.
- **Per-function detail blocks**: weak-assertion → strong-assertion rewrites with side-by-side code.
- **Domain-check section**: concurrency / failure-mode / PBT / determinism findings.
- **Removal candidates list**: each with one-line justification and explicit checkbox.
- **PBT conversion candidates**: specific functions + property sketch using `rapid`.
- **(After fixes)** "After" metrics appended to the same file.

## Integration

- Reuses `property-based-testing` skill's references for PBT conversion templates.
- Consumes `mutation-testing` skill output (gremlins JSON) when `--use-mutations` is set.
- Invocable from `/code-review` and `/pre-pr-review` flows as a sub-step.

## Further Reading

- [`references/smell-catalog.md`](references/smell-catalog.md) — full smell catalog with Go detection logic.
- [`references/coverage-pitfalls.md`](references/coverage-pitfalls.md) — why line coverage misleads; branch / MC/DC / mutation testing context.
- [`references/strengthening-patterns.md`](references/strengthening-patterns.md) — weak-to-strong assertion rewrites.
- [`references/domain-checks.md`](references/domain-checks.md) — concurrency, failure-mode, determinism for distributed/Bitcoin code.
- [`references/reshape-to-invariants.md`](references/reshape-to-invariants.md) — converting example tests to `rapid` properties.
- [`references/workflow.md`](references/workflow.md) — phase-by-phase walkthrough with examples.
