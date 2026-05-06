# Reshape to Invariants: Example Tests → `rapid` Properties

This reference shows how to convert example-based tests into `pgregory.net/rapid` property-based tests when the SUT admits an invariant. Reshape is the most aggressive refinement the skill performs — every reshape proposal in the report shows the original test verbatim alongside the rewrite, and the user must approve it explicitly.

For background, see the [`property-based-testing`](../../property-based-testing/SKILL.md) skill.

## When to Reshape

Reshape is appropriate when:

1. The SUT is a **pure function** (or has clearly bounded side effects).
2. The function admits a **structural invariant** — roundtrip, idempotence, oracle, monotonicity, total order.
3. The existing test is example-based and exercises only a handful of points.
4. The cost of generating arbitrary inputs is bounded (no expensive setup per case).

Reshape is **not** appropriate when:

- The SUT has unbounded side effects (network calls, disk writes) — use a state-machine PBT instead, see below.
- The function has no clear invariant (e.g., a heuristic with no closed-form spec).
- The existing test is documenting a specific bug fix or regression — keep it as-is.

## The Five Property Templates

### 1. Roundtrip

When two functions are inverses (`Encode`/`Decode`, `Marshal`/`Unmarshal`, `Format`/`Parse`).

```go
// Existing example test
func TestEncodeTx(t *testing.T) {
    tx := Tx{Amount: 1000, To: addr1}
    b := Encode(tx)
    got, err := Decode(b)
    require.NoError(t, err)
    require.Equal(t, tx, got)
}

// Reshaped: roundtrip property
func TestEncodeTxRoundtrip(t *testing.T) {
    rapid.Check(t, func(rt *rapid.T) {
        tx := genTx(rt)
        b := Encode(tx)
        got, err := Decode(b)
        require.NoError(rt, err)
        require.Equal(rt, tx, got)
    })
}

func genTx(rt *rapid.T) Tx {
    return Tx{
        Amount: rapid.Int64Range(0, math.MaxInt64).Draw(rt, "amount"),
        To:     genAddress(rt),
        // ...
    }
}
```

The property covers the entire valid input space, not just the cases the developer remembered.

### 2. Idempotence

When applying a function twice gives the same result as once (`Normalize`, `Sort`, `Dedup`).

```go
// Reshaped
func TestNormalizeIdempotent(t *testing.T) {
    rapid.Check(t, func(rt *rapid.T) {
        s := rapid.String().Draw(rt, "s")
        once := Normalize(s)
        twice := Normalize(once)
        require.Equal(rt, once, twice)
    })
}
```

### 3. Oracle / Reference Implementation

When an independent implementation of the same spec exists (stdlib, simpler-but-slower version, paper algorithm).

```go
// Reshaped
func TestQuickSortAgainstStdlib(t *testing.T) {
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

The oracle is independent of the SUT, so any deviation is caught.

### 4. Algebraic Properties (commutativity, associativity, identity)

```go
// Commutativity
func TestAddCommutative(t *testing.T) {
    rapid.Check(t, func(rt *rapid.T) {
        a := rapid.Int().Draw(rt, "a")
        b := rapid.Int().Draw(rt, "b")
        require.Equal(rt, Add(a, b), Add(b, a))
    })
}

// Identity
func TestMergeIdentity(t *testing.T) {
    rapid.Check(t, func(rt *rapid.T) {
        s := genSet(rt)
        require.Equal(rt, s, Merge(s, EmptySet()))
    })
}
```

### 5. Order / Comparator Properties

```go
func TestCompareTotalOrder(t *testing.T) {
    rapid.Check(t, func(rt *rapid.T) {
        a := genItem(rt)
        b := genItem(rt)
        c := genItem(rt)

        // Anti-symmetry
        if Compare(a, b) > 0 {
            require.True(rt, Compare(b, a) < 0)
        }
        // Transitivity
        if Compare(a, b) <= 0 && Compare(b, c) <= 0 {
            require.LessOrEqual(rt, Compare(a, c), 0)
        }
        // Reflexivity
        require.Equal(rt, 0, Compare(a, a))
    })
}
```

## State Machines

For SUTs with side effects (in-memory data structures, in-process state machines), use `rapid.StateMachine`:

```go
// SUT: a thread-safe LRU cache.
type cacheModel struct {
    cache    *LRUCache
    expected map[string]int   // reference: simple map
    capacity int
    keys     []string
}

func (m *cacheModel) Init(rt *rapid.T) {
    m.capacity = rapid.IntRange(1, 100).Draw(rt, "capacity")
    m.cache    = NewLRUCache(m.capacity)
    m.expected = make(map[string]int)
}

func (m *cacheModel) Put(rt *rapid.T) {
    k := rapid.SampledFrom([]string{"a","b","c","d","e"}).Draw(rt, "k")
    v := rapid.Int().Draw(rt, "v")
    m.cache.Put(k, v)
    m.expected[k] = v
    // Track key for later GET.
    if !contains(m.keys, k) { m.keys = append(m.keys, k) }
}

func (m *cacheModel) Get(rt *rapid.T) {
    if len(m.keys) == 0 { return }
    k := rapid.SampledFrom(m.keys).Draw(rt, "k")
    got, ok := m.cache.Get(k)
    if want, exp := m.expected[k]; exp {
        // Reference may not match if cache has evicted; just check
        // present-in-cache implies correct value.
        if ok {
            require.Equal(rt, want, got)
        }
    }
    // (the model could track LRU eviction order to assert what *should*
    //  still be present — see strategies.md in property-based-testing)
}

func (m *cacheModel) Check(rt *rapid.T) {
    require.LessOrEqual(rt, m.cache.Size(), m.capacity)
}

func TestLRU(t *testing.T) {
    rapid.Check(t, rapid.Run[*cacheModel]())
}
```

`rapid.StateMachine` interleaves operations randomly and shrinks failures to minimal sequences. It's the right tool when the SUT is a "thing with state".

## Generator Hygiene

The quality of a property test depends on its generators. Common failures:

### Generators That Skip Important Cases

```go
// BAD: never generates 0 or negative.
amount := rapid.IntRange(1, 1000).Draw(rt, "amount")

// BETTER: include boundary values.
amount := rapid.OneOf(
    rapid.IntRange(0, 0),       // boundary
    rapid.IntRange(1, 1000),
    rapid.IntRange(-100, -1),   // negative path
).Draw(rt, "amount")
```

### Generators That Don't Match The Domain

```go
// BAD: Bitcoin amounts shouldn't be > 21M BTC.
amount := rapid.Int64().Draw(rt, "amount")

// BETTER: domain-bounded.
const maxSats = 21_000_000 * 100_000_000
amount := rapid.Int64Range(0, maxSats).Draw(rt, "amount")
```

### Generators That Are Too Random For Structured Inputs

```go
// BAD: random bytes will rarely produce a valid address.
addr := rapid.SliceOf(rapid.Byte()).Draw(rt, "addr")

// BETTER: domain-aware constructor.
addr := genValidBitcoinAddress(rt)
```

## When Reshape Goes Wrong

Reshape is a structural change to the test. After reshape:

- Re-run the test suite (`go test ./... -race -count=1`).
- If `--use-mutations` is set, optionally re-run gremlins on the touched package and confirm `test_efficacy` did not regress. Pass `--verify-mutations` to Phase B.

A reshape that lowers mutation efficacy is a regression — better tests with more cases should kill *more* mutants, not fewer.

## When Reshape Is The Wrong Move

- **The original test documents a regression or bug fix**. Keep the example as a regression test alongside the property.
- **The function has no clear invariant**. Don't force a property; example tests are honest about what's being tested.
- **The function is so simple that a property is overkill**. `func Add(a, b int) int { return a + b }` doesn't need a property test.

## Cross-Reference

For PBT generator design and strategies, see [`property-based-testing/references/strategies.md`](../../property-based-testing/references/) (the actual files exist; SKILL.md indexes them).
