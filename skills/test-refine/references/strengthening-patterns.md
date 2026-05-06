# Strengthening Patterns: Weak → Strong Assertions

Concrete rewrites of common weak Go test patterns into stronger ones. Each pattern shows the smell, the rewrite, and the rationale.

## Naming Convention (Important)

**Test function names must not contain underscores.** Use camelCase (`TestEncodeTxRoundtrip`, not `TestEncodeTx_Roundtrip`). For variants of the same logical test, use `t.Run("subtest name", func(t *testing.T) {...})` rather than appending `_Variant`. The skill's reshape pass enforces this — any rewrite it proposes uses subtests, not underscored function names.

## Pattern 1: Bare Function Call → Assertion on Return

```go
// WEAK (S01): runs the function, checks nothing.
func TestFee(t *testing.T) {
    CalculateFee(amount, rate)
}
```

```go
// STRONG: assert the actual computed result.
func TestFee(t *testing.T) {
    got := CalculateFee(10_000, 100)
    require.Equal(t, int64(100), got, "fee = amount * rate / 10_000")
}
```

**Why**: the weak version passes even if `CalculateFee` returns `0` for everything. The strong version distinguishes original from any arithmetic mutation.

## Pattern 2: NotNil → Equal

```go
// WEAK: NotNil only catches the case where the function returns nil.
got, err := Build(input)
require.NoError(t, err)
require.NotNil(t, got)
```

```go
// STRONG: assert specific structural properties.
got, err := Build(input)
require.NoError(t, err)
require.Equal(t, expectedID, got.ID)
require.Equal(t, expectedAmount, got.Amount)
require.Len(t, got.Outputs, 2)
```

**Why**: any non-nil return value passes the weak version. The strong version catches mutations to the returned struct's fields.

## Pattern 3: Error Existence → Specific Error

```go
// WEAK: any error passes.
_, err := Parse(input)
require.Error(t, err)
```

```go
// STRONG: assert the specific error type / sentinel.
_, err := Parse(input)
require.ErrorIs(t, err, ErrInvalidPrefix)
```

**Why**: `Error` passes for any failure — including a wrong error from a different code path. `ErrorIs`/`ErrorAs` pin down the specific contract.

## Pattern 4: Boundary Strengthening (around `<` vs `<=`)

```go
// WEAK: tests with values far from the boundary.
require.False(t, IsAllowed(50))
require.True(t,  IsAllowed(2000))
```

```go
// STRONG: test the exact boundary value, both sides.
const threshold = 1000
require.False(t, IsAllowed(threshold-1))
require.True(t,  IsAllowed(threshold))     // distinguishes < from <=
require.True(t,  IsAllowed(threshold+1))
```

**Why**: kills `conditionals-boundary` mutants (`>` ↔ `>=`).

## Pattern 5: Boolean Result → Truth Table

Use `t.Run` subtests rather than `_`-suffixed test names — a single `TestAuthorize` with named subtests reads better than `TestAuthorize_TruthTable`.

```go
// WEAK: only tests "everything true" path.
u := &User{admin: true, allowed: true}
require.True(t, Authorize(u, resource))
```

```go
// STRONG: full truth table (or at least the rows that distinguish && from ||).
func TestAuthorize(t *testing.T) {
    cases := []struct {
        name                 string
        admin, allowed, want bool
    }{
        {"none",     false, false, false},
        {"allowed",  false, true,  false}, // distinguishes && from ||
        {"admin",    true,  false, false}, // distinguishes && from ||
        {"both",     true,  true,  true},
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            u := &User{admin: c.admin}
            r := &Resource{allowed: c.allowed}
            require.Equal(t, c.want, Authorize(u, r))
        })
    }
}
```

**Why**: kills `invert-logical` mutants (`&&` ↔ `||`). Critical for security checks.

## Pattern 6: String Equality → Structured Equality

```go
// WEAK (S06): brittle to any formatting change.
require.Equal(t, "User{name=alice, age=30}", u.String())
```

```go
// STRONG: assert on fields directly.
require.Equal(t, "alice", u.Name)
require.Equal(t, 30, u.Age)

// Or, for large structs:
diff := cmp.Diff(want, got)
require.Empty(t, diff, "user mismatch")
```

**Why**: refactoring `String()` shouldn't break correctness tests.

## Pattern 7: Setter/Getter Round-Trip → Behavior Test

```go
// WEAK (S03): tests assignment.
o := New()
o.SetMode(ReadOnly)
require.Equal(t, ReadOnly, o.GetMode())
```

```go
// STRONG: tests behavior the field controls.
o := New()
o.SetMode(ReadOnly)
err := o.Write([]byte("data"))
require.ErrorIs(t, err, ErrReadOnly)
```

**Why**: setters/getters are tested by every test that uses the object. The strong version verifies the field actually does something.

## Pattern 8: Error Discarded → Error Asserted

```go
// WEAK (S05): error swallowed.
got, _ := Decode(input)
require.Equal(t, expected, got)
```

```go
// STRONG: error explicitly asserted (depending on intent):
got, err := Decode(input)
require.NoError(t, err, "decoding %q", input)
require.Equal(t, expected, got)
```

**Why**: silent error means `got` is the zero value, and the assertion can pass for the wrong reason.

## Pattern 9: Side Effect Not Asserted → State Read-Back

```go
// WEAK (S11): mutation not checked.
tx := NewTx(amount)
Sign(tx, key)
```

```go
// STRONG: read the mutated state.
tx := NewTx(amount)
require.NoError(t, Sign(tx, key))
require.NotNil(t, tx.Signature)
require.True(t, Verify(tx, key.Public()))   // round-trip
```

**Why**: forces the test to verify that signing actually happened correctly, not just that it didn't panic.

## Pattern 10: Loop With No Assertion → Per-Iteration Check

```go
// WEAK: loop runs but doesn't verify each iteration.
peers := AllPeers()
for _, p := range peers {
    p.Disconnect()
}
require.Empty(t, ActivePeers())          // only the final state checked
```

```go
// STRONG: assert per-iteration invariants.
peers := AllPeers()
for i, p := range peers {
    require.NoError(t, p.Disconnect())
    require.Equalf(t, len(peers)-i-1, len(ActivePeers()),
        "after disconnecting peer %d", i)
}
require.Empty(t, ActivePeers())
```

**Why**: the weak version passes even if disconnect is broken for some peers, as long as `ActivePeers()` ends empty.

## Pattern 11: Time-Dependent Test → Inject Clock

```go
// WEAK: test depends on wall clock; flaky and slow.
expiry := time.Now().Add(time.Hour)
token := Issue(expiry)
time.Sleep(time.Hour + time.Second)
require.True(t, Expired(token))
```

```go
// STRONG: inject the clock; deterministic, instant.
clock := clock.NewMock()
clock.Set(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

issued := clock.Now()
token := Issue(issued.Add(time.Hour), withClock(clock))

clock.Add(time.Hour + time.Second)
require.True(t, Expired(token, withClock(clock)))
```

**Why**: tests that sleep are slow and racy. Determinism enables reproducibility (DST principle from FoundationDB / TigerBeetle).

## Pattern 12: Concurrent Code, Sequential Test → Stress

```go
// WEAK: tests a concurrent SUT sequentially.
func TestSafeMap(t *testing.T) {
    m := NewSafeMap()
    m.Set("a", 1)
    require.Equal(t, 1, m.Get("a"))
}
```

```go
// STRONG: exercise actual concurrency.
func TestSafeMapConcurrent(t *testing.T) {
    m := NewSafeMap()
    const n = 100
    var wg sync.WaitGroup
    for i := 0; i < n; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            m.Set(fmt.Sprintf("k%d", i), i)
        }(i)
    }
    wg.Wait()

    for i := 0; i < n; i++ {
        require.Equal(t, i, m.Get(fmt.Sprintf("k%d", i)))
    }
}
```

Run with `go test -race`.

**Why**: a concurrent map is only "safe" if it's been *concurrently exercised*. Sequential tests can't catch races.

## Pattern 13: Single Value → Property

For pure functions, especially serialization and parsing, replace many example tests with one property:

```go
// WEAK: a few hardcoded roundtrips.
func TestEncodeDecode(t *testing.T) {
    cases := []Tx{tx1, tx2, tx3}
    for _, c := range cases {
        b, _ := Encode(c)
        got, _ := Decode(b)
        require.Equal(t, c, got)
    }
}
```

```go
// STRONG: rapid property over arbitrary inputs.
func TestEncodeDecodeRoundtrip(t *testing.T) {
    rapid.Check(t, func(rt *rapid.T) {
        original := genTx(rt)
        b, err := Encode(original)
        require.NoError(rt, err)
        got, err := Decode(b)
        require.NoError(rt, err)
        require.Equal(rt, original, got)
    })
}
```

See [`reshape-to-invariants.md`](reshape-to-invariants.md) for more property templates.

**Why**: the property runs hundreds of cases automatically and shrinks failures to minimal counterexamples.

## Pattern 14: Multi-Step Test Without Messages → Annotated Asserts

```go
// WEAK (S09): assertion roulette.
require.Equal(t, 1, x)
require.Equal(t, 2, y)
require.Equal(t, 3, z)
require.NoError(t, err)
```

```go
// STRONG: annotated, table-driven.
cases := []struct {
    name        string
    in          Input
    wantX, wantY, wantZ int
}{
    {"happy", happy, 1, 2, 3},
    {"empty", empty, 0, 0, 0},
}
for _, c := range cases {
    t.Run(c.name, func(t *testing.T) {
        out, err := Compute(c.in)
        require.NoError(t, err)
        require.Equal(t, c.wantX, out.X, "X")
        require.Equal(t, c.wantY, out.Y, "Y")
        require.Equal(t, c.wantZ, out.Z, "Z")
    })
}
```

**Why**: when a test fails, you immediately see which sub-case and which field.

## Pattern 15: Reference-Implementation Oracle

For complex pure functions, the strongest assertion is an *independent* reference implementation:

```go
// STRONG: oracle property.
func TestSortAgainstStdlib(t *testing.T) {
    rapid.Check(t, func(rt *rapid.T) {
        in := rapid.SliceOf(rapid.Int()).Draw(rt, "in")
        got := append([]int(nil), in...)
        QuickSort(got)
        want := append([]int(nil), in...)
        sort.Ints(want)
        require.Equal(rt, want, got)
    })
}
```

**Why**: the oracle is independent of the SUT, so the property catches *any* deviation in behavior.

## Picking The Right Strengthening

The composite priority score determines order. Inside a finding, picking the rewrite is mostly mechanical:

| Smell | Default rewrite |
|---|---|
| S01 | Pattern 1 (bare call) |
| S02 | Pattern 1 or remove |
| S03 | Pattern 7 (behavior test) or remove |
| S04 | Pattern 9 (state read-back) |
| S05 | Pattern 8 (error asserted) |
| S06 | Pattern 6 (structural equality) |
| S07 | Pattern 1 with explicit `NotNil` |
| S08 | Consolidate to table-driven |
| S09 | Pattern 14 (annotated table) |
| S10 | Pattern 15 (independent oracle) |
| S11 | Pattern 9 |
| S12 | Pattern 4/5 (whichever mutator survived) |

Domain checks (concurrency, failure-mode, determinism) feed Patterns 11–13.
