---
name: mutation-testing
description: Validates Go test suite quality through mutation testing using go-gremlins/gremlins. Mutates production code, runs the test suite against each mutant, and reports which mutants the tests fail to kill — exposing weak assertions that line coverage cannot detect. Use when evaluating test effectiveness, validating newly written tests, or improving test quality for mission-critical code (consensus, channel state, payment flows, crypto). Triggers: "mutation test", "are these tests strong", "validate test quality", "/mutation-testing".
---

# Mutation Testing

Mutation testing evaluates test quality by introducing small, deliberate bugs into production code (mutants) and checking whether the test suite fails. A test that passes on a mutant did not actually verify the behavior the mutant changed.

This skill is a thin orchestrator over [`go-gremlins/gremlins`](https://github.com/go-gremlins/gremlins) — a maintained Go mutation testing tool. The skill provides install, run, and analysis wrappers that produce machine-readable JSON for downstream tooling (notably the `test-refine` skill).

## Why Mutation Testing

A test suite can hit 100% line coverage and still be useless: tests can execute code without asserting on its results, or assert only on side-irrelevant fields. Mutation testing closes this gap by checking whether the test suite distinguishes the original code from a mutant. See `references/coverage-pitfalls.md` (in the `test-refine` skill) for the broader context.

## When to Use

- After generating tests with `test-forge` or by hand — verify they have real assertions.
- Before merging consensus / payment / crypto code — quality gate on critical paths.
- During code review — surface weak tests in the diff.
- As a signal source for `test-refine` — survivors map to weak-assertion findings.

**Target efficacy** (gremlins terminology: `test_efficacy = killed / (killed + lived)`):

| Code class | Target |
|---|---|
| Mission-critical (consensus, wallet, channel, crypto) | 90%+ |
| Core business logic | 80–90% |
| General code | 70–80% |
| Trivial/glue code | run only if cheap |

## Workflow

### 1. Install gremlins (once)

```bash
~/.claude/skills/mutation-testing/scripts/install-gremlins.sh
```

The script pins to a known-good version (override with `GREMLINS_VERSION=...`). Requires `go` on `PATH` and `$(go env GOPATH)/bin` on `PATH`.

### 2. Run mutations

```bash
# Default: cwd, JSON to .reviews/mutations/<slug>.json
~/.claude/skills/mutation-testing/scripts/unleash.sh

# Targeted package
~/.claude/skills/mutation-testing/scripts/unleash.sh \
    --pkg ./internal/wallet \
    --output .reviews/mutations/wallet.json

# With integration tests and a config file
~/.claude/skills/mutation-testing/scripts/unleash.sh \
    --pkg ./internal/channel \
    --integration \
    --config .gremlins.yaml \
    --silent
```

### 3. Analyze survivors

```bash
~/.claude/skills/mutation-testing/scripts/analyze-survivors.sh \
    --input .reviews/mutations/wallet.json \
    --output .reviews/mutations/wallet.md
```

Produces a markdown report with: efficacy/coverage summary, survivors ranked by file (consensus/channel/wallet paths bubble to the top), and mutator-type breakdown.

## Gremlins JSON Schema

`gremlins unleash --output <file>` emits a single JSON document:

```json
{
  "go_module": "github.com/example/foo",
  "test_efficacy": 82.00,
  "mutations_coverage": 80.00,
  "mutants_total": 100,
  "mutants_killed": 82,
  "mutants_lived": 8,
  "mutants_not_viable": 2,
  "mutants_not_covered": 10,
  "elapsed_time": 123.456,
  "files": [
    {
      "file_name": "wallet.go",
      "mutations": [
        { "line": 42, "column": 8, "type": "CONDITIONALS_NEGATION", "status": "KILLED" }
      ]
    }
  ]
}
```

**Mutation status values**:

| Status | Meaning | Action |
|---|---|---|
| `KILLED` | Test suite caught the mutation | Good — no action |
| `LIVED` | Tests passed despite mutation | **Survivor** — strengthen tests |
| `NOT COVERED` | Mutation in code no test exercises | Add a test for that path |
| `TIMED OUT` | Tests timed out — implicit kill | Investigate (might be perf bug) |
| `NOT VIABLE` | Mutation produced uncompilable code | Excluded from score |
| `RUNNABLE` | Dry-run only; would be tested | (only in `--dry-run`) |

**Key metrics**:
- `test_efficacy` = `killed / (killed + lived)` — quality of assertions on covered code.
- `mutations_coverage` = `(killed + lived) / (killed + lived + not_covered)` — how much code is exercised at all.

A high `mutations_coverage` with low `test_efficacy` means tests run code without verifying its behavior — the classic "100% line coverage, 0% real testing" failure mode.

## Configuration

Gremlins is configured via `.gremlins.yaml` (or `--config <path>`). Mutators ship default-on for safe operators and default-off for aggressive ones.

**Default-on mutators** (always enabled):
- `arithmetic-base` — `+ - * / %`
- `conditionals-boundary` — `< <= > >=`
- `conditionals-negation` — `== !=`, boolean conditions
- `increment-decrement` — `++ --`
- `invert-negatives` — `-x` ↔ `+x`

**Default-off mutators** — enable for critical packages:
- `invert-assignments` — `+= -= *= /=` etc. swaps
- `invert-bitwise` — `& | ^` swaps
- `invert-bwassign` — `&= |= ^=` swaps
- `invert-logical` — `&& ↔ ||` (security-critical: catches auth bypass mutations)
- `invert-loopctrl` — `break ↔ continue`
- `remove-self-assignments` — drop `x = x op y` updates

**Recommended config for consensus/wallet/payment code**:

```yaml
silent: false
unleash:
  workers: 0          # use all CPUs
  test-cpu: 0         # no per-test CPU pinning
  threshold:
    efficacy: 90      # fail if below 90%
    mutant-coverage: 85
mutants:
  arithmetic-base:        { enabled: true }
  conditionals-boundary:  { enabled: true }
  conditionals-negation:  { enabled: true }
  increment-decrement:    { enabled: true }
  invert-negatives:       { enabled: true }
  invert-assignments:     { enabled: true }
  invert-bitwise:         { enabled: true }
  invert-bwassign:        { enabled: true }
  invert-logical:         { enabled: true }   # critical for && / || in auth
  invert-loopctrl:        { enabled: true }
  remove-self-assignments:{ enabled: true }
```

See [gremlins.dev configuration docs](https://gremlins.dev/0.5/usage/configuration/) for the full schema.

## Threshold Gating (CI)

For CI, use `--silent` and set thresholds in config or via env vars:

```bash
gremlins unleash --silent --output mutations.json ./...
# Exit nonzero if efficacy < threshold.
```

The `unleash.threshold.efficacy` and `unleash.threshold.mutant-coverage` keys cause gremlins to exit nonzero when the run falls below the configured percentages — wire this into your PR check.

## Integration with Other Skills

### `test-refine`

The `test-refine` skill consumes gremlins JSON to identify weak-assertion zones (smell `S12: mutation-survivor`). When invoked with `--use-mutations`, it calls `unleash.sh` and cross-references `LIVED` mutants with the AST smell scan.

### `test-forge`

After `test-forge` generates tests, run mutation testing to validate them. `LIVED` mutants are direct evidence of weak assertions in the generated tests.

### `code-review`

Include the test_efficacy delta in PR review — regression of >5% in covered code is a strong signal of weakening test quality.

## Interpreting Results

**High efficacy (≥90%)**: Tests have strong assertions. Focus remaining work on `NOT COVERED` mutants (uncovered code paths).

**Medium (75–90%)**: Tests cover main paths. Survivors usually indicate boundary or error-path gaps.

**Low (<75%)**: Significant gaps — tests likely run code without checking outputs. Pair with `test-refine` to identify the specific smells.

**Mutator breakdown** tells you the *kind* of weakness:
- `conditionals-boundary` LIVED → missing edge tests at thresholds.
- `invert-logical` LIVED → missing truth-table coverage for `&&`/`||`.
- `arithmetic-base` LIVED → tests don't verify calculation results.
- `remove-self-assignments` LIVED → state mutations not asserted.

## Equivalent Mutants

Some `LIVED` mutants are semantically equivalent to the original — no test could kill them. Common cases:
- Mutated value immediately overwritten before being read.
- Mutation in unreachable code.
- Operator swap in associative/commutative context with no observable difference.

When you identify an equivalent mutant, document it (e.g., a comment near the mutation site, or a project-level `EQUIVALENT_MUTANTS.md`) so reviewers don't waste time on it. Gremlins doesn't filter equivalents automatically.

## Gremlins Limitations

From the upstream README: gremlins targets *smallish* Go modules (microservices). On very large modules, runs can take hours. Mitigations:

- **Per-package runs** via `--pkg ./internal/wallet`. Don't pass `./...` on a 500k-LOC monorepo.
- **Skip generated code** by using build tags or running on hand-written packages only.
- **Use `--workers`** to bound parallelism if memory is tight.
- **Use `--dry-run`** first to preview the mutation count and skip if it's too large.

## Further Reading

- [`references/mutation_operators.md`](references/mutation_operators.md) — gremlins mutator catalog with examples.
- [`references/best_practices.md`](references/best_practices.md) — patterns for boundary, security, and state-machine testing.
- [Gremlins documentation](https://gremlins.dev/) — full CLI and config reference.
- "Are Mutants a Valid Substitute for Real Faults in Software Testing?" (Just et al.) — academic foundation.
