# Test Smell Catalog

This catalog defines the smells `detect-smells.go` looks for, with Go examples and detection logic. Severities feed the composite priority score.

Synthesized from [testsmells.org](https://testsmells.org/), [Test smells 20 years later (Empirical Software Engineering, 2022)](https://link.springer.com/article/10.1007/s10664-022-10207-5), and [practical assertion guidance from Diffblue](https://www.diffblue.com/resources/how-to-write-better-unit-test-assertions/).

## S01: No Assertions At All — High

A test runs the SUT but never asserts on observable behavior. JUnit-style frameworks (and `*testing.T`) report passing as long as the test body doesn't error or panic.

```go
// SMELL
func TestProcess(t *testing.T) {
    p := New()
    p.Process(input)            // <- runs, but checks nothing.
}
```

**Detection** (AST):
- Function name matches `^Test[A-Z]`.
- Body contains no calls to `t.Errorf`, `t.Error`, `t.Fatal`, `t.Fatalf`, no `require.*` or `assert.*` calls, and no manual `if got != want { t.Errorf(...) }` patterns.
- Excludes: tests that explicitly document "must not panic" with `defer recover()` AND have an explanatory comment (treated as S04 instead).

**Action**: add an assertion on what the test is trying to verify, or remove if the test is meaningless.

## S02: Tautological Assertion — High

Assertion compares a value to itself, or compares two values constructed identically. Always passes; verifies nothing.

```go
// SMELL
func TestEqual(t *testing.T) {
    require.Equal(t, want, want)        // tautology
    require.True(t, true)               // tautology
}

// Subtler smell
func TestSum(t *testing.T) {
    a, b := 1, 2
    require.Equal(t, a + b, 1 + 2)      // both sides are constants — tautology
}
```

**Detection**:
- `assert.Equal(t, X, X)` / `require.Equal(t, X, X)` where both args reference the same identifier or evaluate to the same constant expression.
- `assert.True(t, true)`, `assert.False(t, false)`.
- `if x == x` patterns.

**Action**: replace with an assertion against a known-correct value, or remove.

## S03: Getter/Setter Trivial Test — Medium

The test verifies the language's assignment semantics, not behavior.

```go
// SMELL
func TestSetGetName(t *testing.T) {
    o := New()
    o.SetName("foo")
    require.Equal(t, "foo", o.GetName())    // tests `=`, not behavior
}
```

**Detection**:
- Test body contains only one `Set*` call followed by one `Get*`/`get*` assertion on the same field.
- No business logic asserted.

**Action**: remove (these have negative value — they break on refactor and catch nothing). If the setter has validation logic, write a test for the validation instead.

## S04: Asserts No Panic Only — High

The only assertion is "didn't panic". Tells you nothing about behavior.

```go
// SMELL
func TestSafe(t *testing.T) {
    require.NotPanics(t, func() {
        ProcessTransaction(tx)              // panic is the only thing checked
    })
}
```

**Detection**:
- Test body's only assertion is `require.NotPanics`/`assert.NotPanics`, OR test body wraps SUT in `defer func() { _ = recover() }()` with no other assertion.

**Action**: add assertions on actual return values, side effects, or state. Keep the no-panic guarantee only if it's a documented contract.

## S05: Unchecked Error From SUT — High

The SUT returns `(value, error)`. The test ignores the error and asserts only on `value`.

```go
// SMELL
func TestParse(t *testing.T) {
    got, _ := Parse(input)              // error discarded
    require.Equal(t, "expected", got)
}
```

**Detection** (AST + types):
- SUT call returns `error`-typed second result.
- Test discards with `_` and continues to assert.

**Action**: either `require.NoError(t, err)` (if happy path) or assert the specific error (`require.ErrorIs(t, err, ErrTarget)`).

## S06: Sensitive Equality — Medium

Asserts equality on a `String()` rendering or a `fmt.Sprintf`. Brittle: any cosmetic change to formatting breaks the test without a real behavior change.

```go
// SMELL
require.Equal(t, "User{name=alice, age=30}", user.String())
require.Equal(t, fmt.Sprintf("%v", got), fmt.Sprintf("%v", want))
```

**Detection**:
- One side of `assert.Equal`/`require.Equal` is a call to `.String()`, `fmt.Sprint`, or `fmt.Sprintf`.

**Action**: assert on the structured value directly. Use `cmp.Diff` if the types are large.

## S07: Conditional / Skipped Assertion — Medium

The test contains an early return or skip *before* its assertion, so on the skip path the test silently passes.

```go
// SMELL
func TestPath(t *testing.T) {
    got := Compute(input)
    if got == nil {
        return                          // silently passes when computation fails
    }
    require.Equal(t, want, *got)
}
```

**Detection**:
- Control-flow path from test entry to a `return` or `t.Skip` that bypasses all subsequent assertions.

**Action**: replace early returns with explicit assertions (`require.NotNil(t, got)`).

## S08: Duplicate Test Body (Semantic) — Medium

Two or more tests with structurally identical bodies (modulo identifier renames). Often results from copy-paste during table extraction that didn't finish.

**Detection**:
- AST normalize (rename all locals to `v0, v1, …`, normalize literals into placeholders).
- Hash normalized AST.
- Group tests by hash; groups of size > 1 are duplicates.

**Action**: consolidate into a table-driven test, or delete the duplicate.

## S09: Assertion Roulette — Low

A test has many bare assertions with no failure messages. When it fails, you can't tell which assertion caused it without a stack trace.

```go
// SMELL
func TestAll(t *testing.T) {
    require.Equal(t, 1, a)
    require.Equal(t, 2, b)
    require.Equal(t, 3, c)
    require.Equal(t, 4, d)              // which one failed?
}
```

**Detection**:
- More than 3 assertions in a single test.
- None use the message argument.
- Test is not table-driven (table-driven tests are usually fine).

**Action**: convert to table-driven, or add identifying messages.

## S10: Expect-The-Expected — High

The expected value is derived from the actual computation. The assertion is structurally a tautology.

```go
// SMELL
func TestNorm(t *testing.T) {
    got := Normalize(input)
    want := strings.ToLower(input)      // <- "want" computed same way as "got"
    require.Equal(t, want, got)
}
```

**Detection**:
- Dataflow: `want` is computed by calling code that mirrors the SUT's logic, or by reusing the SUT's output.

**Action**: hardcode the expected value, or use a reference implementation that's *independent* of the SUT.

## S11: Side-Effect Not Asserted — Medium

The SUT mutates a state argument or global. The test calls it but never reads back the mutated state.

```go
// SMELL
func TestSign(t *testing.T) {
    tx := NewTx()
    Sign(tx, key)                       // mutates tx
    // (no assertion on tx.Signature)
}
```

**Detection** (heuristic):
- SUT call passes a pointer or interface argument.
- The argument's fields (or the global it mutates) are never read between the SUT call and the end of the test.

**Action**: assert on the mutated state.

## S12: Mutation-Survivor Zone — High (when gremlins data available)

The function under test has `LIVED` mutants in the gremlins JSON, but a test for it exists. Indicates the test executes the code without verifying behavior.

**Detection**:
- Cross-reference: function `F` has tests `TestF*`, AND `gremlins.json` shows mutants in `F` with `status: "LIVED"`.
- Equivalence filter: skip mutants where the mutated value is dead (overwritten before read), or in unreachable code.

**Action**: per surviving mutant, identify the test that exists for that branch and tighten the assertion to distinguish the original from the mutated value. See [`mutation_operators.md` in the mutation-testing skill](../../mutation-testing/references/mutation_operators.md) for per-mutator implied test gaps.

## Severity → Score Mapping

| Severity | Score |
|---|---|
| High | 1.0 |
| Medium | 0.6 |
| Low | 0.3 |

Used by `score.go` in the composite priority formula.

## Related Smells (Detected but Lower Priority)

The catalog above is what the AST analyzers actively flag. The literature includes additional smells that are usually noise rather than signal:

- **Magic numbers in assertions** — sometimes valid (a calculated expected value).
- **Eager test (multiple SUT calls)** — sometimes valid for state-machine tests.
- **Mystery guest (test reads from external file)** — fixtures are often necessary.

These are documented but not flagged by default. To enable, pass `--strict` to `triage.sh`.

## False Positive Discipline

Empirical research warns that test-smell detectors trigger noisily; developers often perceive warnings as overly strict ([Test smells 20 years later, 2022](https://link.springer.com/article/10.1007/s10664-022-10207-5)). To avoid this:

1. The default detector is conservative — it flags only the patterns in this catalog.
2. Every flagged finding shows the *exact* code triggering it. If it's wrong, the user rejects it in the report (no checkbox).
3. The skill prefers false negatives over false positives — better to miss a smell than to nag.
