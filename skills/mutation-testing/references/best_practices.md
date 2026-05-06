# Mutation Testing Best Practices

Patterns for using gremlins effectively on Go projects, with emphasis on Bitcoin/Lightning and other mission-critical code.

## Table of Contents

1. [Scoping Mutations](#scoping-mutations)
2. [Performance](#performance)
3. [Reading Survivors](#reading-survivors)
4. [Equivalent Mutants](#equivalent-mutants)
5. [CI/CD Integration](#cicd-integration)
6. [Mission-Critical Patterns](#mission-critical-patterns)
7. [Gremlins-Specific Gotchas](#gremlins-specific-gotchas)
8. [Common Pitfalls](#common-pitfalls)

---

## Scoping Mutations

### Start Narrow

Don't run gremlins on the entire codebase at once. Per the upstream README, gremlins is built for *smallish* Go modules; runs on large monorepos can take hours.

**Recommended ordering**:

1. **Mission-critical code**: consensus, channel state, payment, crypto, signing.
2. **Complex business logic**: state machines, fee estimators, routing.
3. **Bug-prone areas**: code with a history of post-merge fixes.
4. **Recently changed code**: PR-scoped runs.
5. **Public API surfaces**: external contracts.

**Skip mutation testing for**:

- Generated code (protobuf, mocks, gRPC stubs).
- Pure delegation / pass-through code.
- Test helpers — testing test code is rarely worthwhile.
- Trivial getters/setters.

### Per-Package Invocation

Always prefer per-package runs over `./...`:

```bash
~/.claude/skills/mutation-testing/scripts/unleash.sh \
    --pkg ./internal/wallet \
    --output .reviews/mutations/wallet.json
```

### Diff-Scoped Runs

For PR-time runs, scope to changed packages:

```bash
# Find packages with changed Go files (not test files).
changed_pkgs=$(git diff --name-only origin/main...HEAD \
    | grep '\.go$' | grep -v '_test\.go$' \
    | xargs -I{} dirname {} | sort -u)

for pkg in $changed_pkgs; do
    ~/.claude/skills/mutation-testing/scripts/unleash.sh \
        --pkg "./$pkg" \
        --silent \
        --output ".reviews/mutations/$(echo "$pkg" | tr / _).json"
done
```

---

## Performance

### Workers and CPU Pinning

```yaml
unleash:
  workers: 0      # 0 = all CPUs; lower if memory-bound
  test-cpu: 0     # 0 = no constraint per test process
  timeout-coefficient: 0  # 0 = default; raise for slow tests
```

Multiply `workers × test-cpu` to estimate peak CPU usage. On a 16-core machine with `workers=8` and `test-cpu=2`, you'll saturate the box.

### Dry-Run First

Before a long run, preview the mutant count:

```bash
gremlins unleash --dry-run --pkg ./internal/wallet
```

If the count is unreasonable (tens of thousands), narrow the scope or disable aggressive mutators.

### Test Timeout Coefficient

Gremlins applies a multiplier to the original test runtime as the per-mutant timeout. If your tests have legitimate long-running scenarios, raise `timeout-coefficient`. If they hang on infinite loops introduced by mutation, lower it.

---

## Reading Survivors

For each `LIVED` mutant, the test gap falls into one of three categories:

### Category 1: Missing Test (uncovered behavior)

```go
// Original
if balance >= threshold { return ErrInsufficient }

// Mutant: balance > threshold (LIVED)
// Gap: no test at exact threshold value.
```

**Fix**: add a boundary test.

```go
func TestBalanceExactThreshold(t *testing.T) {
    err := Check(threshold)         // exact value
    require.ErrorIs(t, err, ErrInsufficient)
}
```

### Category 2: Weak Assertion (covered but not verified)

```go
// Original
result := Calculate(a + b)

// Mutant: a - b (LIVED)
// Gap: existing test calls Calculate but doesn't assert on result.
```

**Fix**: add the assertion.

```go
func TestCalculate(t *testing.T) {
    require.Equal(t, 15, Calculate(10, 5))   // now distinguishes a+b from a-b
}
```

### Category 3: Wrong Test Data (test exists but uses values that hide the bug)

```go
// Original
if amount < 0 { return ErrNegative }

// Mutant: amount <= 0 (LIVED)
// Existing test uses amount = -10 — both original and mutant agree.
// Gap: no test at amount = 0.
```

**Fix**: add the discriminating value.

```go
func TestZero(t *testing.T) {
    require.NoError(t, Validate(0))   // distinguishes < 0 from <= 0
}
```

---

## Equivalent Mutants

Some `LIVED` mutants are semantically equivalent to the original — no test could kill them. Examples:

- Mutated value overwritten before any read.
- Mutation in unreachable code.
- Associative operator swap with commutative operands.

When you confirm equivalence:

1. Add a comment near the mutation site explaining why.
2. Optionally, maintain `EQUIVALENT_MUTANTS.md` at the repo root listing them.
3. Manually subtract from the survivor count when interpreting results.

Gremlins does **not** filter equivalents automatically. Treat the raw `test_efficacy` as a lower bound on real efficacy.

---

## CI/CD Integration

### Threshold-Gated PR Check

```yaml
# .github/workflows/mutation-test.yml
name: Mutation Testing

on:
  pull_request:
    paths: ['**.go', '!**_test.go']

jobs:
  mutation-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }

      - name: Install gremlins
        run: ~/.claude/skills/mutation-testing/scripts/install-gremlins.sh

      - name: Run mutations on changed packages
        run: |
          changed=$(git diff --name-only origin/${{ github.base_ref }}...HEAD \
            | grep '\.go$' | grep -v '_test\.go$' \
            | xargs -I{} dirname {} | sort -u)
          for pkg in $changed; do
            ~/.claude/skills/mutation-testing/scripts/unleash.sh \
                --pkg "./$pkg" \
                --silent \
                --config .gremlins.yaml \
                --output ".reviews/mutations/$(echo "$pkg" | tr / _).json"
          done

      - name: Comment on PR
        run: |
          for f in .reviews/mutations/*.json; do
            ~/.claude/skills/mutation-testing/scripts/analyze-survivors.sh \
                --input "$f" \
                --output "${f%.json}.md"
          done
          # Concatenate and post as PR comment via gh CLI.
```

### Threshold Config

Use the built-in threshold gating to fail the job on regression:

```yaml
# .gremlins.yaml
unleash:
  threshold:
    efficacy: 85
    mutant-coverage: 80
```

Gremlins exits nonzero when either threshold is unmet.

---

## Mission-Critical Patterns

### Financial Calculations

```go
func CalculateFee(amount, rate int64) int64 {
    return (amount * rate) / 10000
}
```

Required mutators: `arithmetic-base`, `conditionals-boundary`, `invert-negatives`.

Required tests:

```go
func TestCalculateFeePrecision(t *testing.T) {
    require.Equal(t, int64(1), CalculateFee(10000, 1))
    require.Equal(t, int64(0), CalculateFee(9999, 1))   // boundary
}

func TestCalculateFeeOverflow(t *testing.T) {
    require.NotPanics(t, func() {
        _ = CalculateFee(math.MaxInt64, 1)
    })
}
```

### Authorization Checks

```go
func Authorize(u *User, r *Resource) bool {
    return u.IsAdmin() && r.AllowsUser(u)
}
```

Required mutator: **`invert-logical`** — without it, `&& → ||` (an auth-bypass) won't be tested.

Required tests:

```go
func TestAuthorize(t *testing.T) {
    cases := []struct {
        admin, allowed, want bool
    }{
        {false, false, false},
        {false, true, false},   // distinguishes && from ||
        {true,  false, false},  // distinguishes && from ||
        {true,  true,  true},
    }
    for _, c := range cases {
        u := &User{admin: c.admin}
        r := &Resource{allowed: c.allowed}
        require.Equal(t, c.want, Authorize(u, r))
    }
}
```

### State Machine Transitions

Enable `invert-bitwise`, `invert-bwassign`, `remove-self-assignments`, and `invert-logical`. Test every valid transition and at least one invalid transition per state.

### Bitcoin/Lightning Specific

| Module | Mutators to enable | Why |
|---|---|---|
| Channel state machine | all | Money + state flags + transitions |
| HTLC routing | all | Money + control flow |
| Commitment tx construction | all | Money + bitwise (script flags) |
| BOLT11 invoice parsing | all | Parser correctness |
| Wire encoding | all + run with `invert-bitwise` | Byte-level correctness |
| Fee estimation | defaults + `invert-assignments` | Arithmetic dominates |
| Peer scoring | defaults | Mostly arithmetic and conditionals |

---

## Gremlins-Specific Gotchas

### Pre-1.0 API

Gremlins is at v0.x. Config flags can change between minor releases. Pin a version in `install-gremlins.sh` and bump deliberately.

### Doesn't Scale to Huge Modules

Per the upstream README, runs on very large modules can take hours. Use per-package invocation always.

### `NOT VIABLE` Excluded From Score

`NOT VIABLE` mutants (mutations that fail to compile) are excluded from `test_efficacy` and `mutations_coverage`. This is correct — they're impossible to kill — but be aware that a high `not_viable` count can mean gremlins is generating low-quality mutants for your code.

### Build Tags

If your tests rely on build tags (e.g., `//go:build integration`), pass `--tags`:

```bash
gremlins unleash --tags integration ./...
```

For tests that only run with `--integration`, also pass `--integration` to gremlins.

### Test Cache

Gremlins runs tests with `-count=1`-like behavior under the hood — test cache shouldn't interfere. If you see suspicious results, ensure your tests aren't reading external state that varies between runs.

---

## Common Pitfalls

### Pitfall 1: Treating efficacy as a goal

Aiming for 100% efficacy on a non-critical package wastes time on equivalent mutants. Set targets by code class (see SKILL.md table).

### Pitfall 2: Mutating test code

Gremlins doesn't mutate `_test.go` files by default — but if a test imports a helper package, the helper package's mutants will run. Skip helper-only packages.

### Pitfall 3: Slow / flaky tests

Mutation testing amplifies test flakiness — every mutant runs the suite. Fix flaky tests before mutation testing or you'll chase ghosts.

### Pitfall 4: Ignoring `mutations_coverage`

A 95% efficacy with 40% coverage means most code isn't tested at all — efficacy only measures what's covered. Always read both metrics.

### Pitfall 5: One-off audits

Mutation testing is most valuable in CI as a regression gate. A single audit fades; a threshold gate doesn't.

### Pitfall 6: Confusing `LIVED` with bug

A `LIVED` mutant doesn't mean there's a bug in production code — it means the test suite wouldn't catch *one specific kind of bug*. Whether that matters depends on whether the mutated code path is reachable in production with bug-causing inputs.

---

## Summary Checklist

Before:
- [ ] Pin gremlins version
- [ ] Choose target packages (mission-critical first)
- [ ] Verify baseline tests pass cleanly with no flakes
- [ ] Configure mutators per code class

During:
- [ ] Watch for high `NOT VIABLE` counts (mutator quality)
- [ ] Distinguish boundary, weak-assertion, and missing-test gaps
- [ ] Mark equivalent mutants

After:
- [ ] Add targeted tests for high-priority survivors
- [ ] Re-run to confirm
- [ ] Wire threshold gate into CI

For mission-critical code:
- [ ] All mutators enabled
- [ ] 90%+ test_efficacy
- [ ] 85%+ mutations_coverage
- [ ] All survivors reviewed (killed or documented as equivalent)
