# Domain Checks: Concurrency, Failure-Mode, Property, Determinism

Standard test smells aren't enough for distributed-systems and Bitcoin/Lightning code. This reference describes four domain-specific dimensions the skill checks beyond the assertion-strength catalog.

These checks draw on the practices from [Jepsen](https://jepsen.io/), [FoundationDB / TigerBeetle deterministic simulation testing](https://antithesis.com/docs/resources/deterministic_simulation_testing/), and the property-based-testing tradition (`rapid`, Hypothesis, QuickCheck).

## 1. Concurrency

### What Goes Wrong

A SUT that uses goroutines, channels, or mutexes can have:

- Data races (caught only with `-race`).
- Deadlocks (caught only when locks are held in real schedules).
- Lost updates / wakeup / send (caught only under contention).

A purely sequential test of concurrent code marks coverage but proves nothing about thread-safety.

### What the Skill Looks For

For SUT functions in scope:

- Has the SUT spawned goroutines, used `chan`, `sync.Mutex`/`RWMutex`, `atomic.*`, `sync.WaitGroup`, `sync.Once`?
- If yes, does at least one test in the package:
  - Use `t.Parallel()` *and* exercise concurrent operations against the SUT?
  - Spawn ≥2 goroutines that all call the SUT?
  - Use `sync.WaitGroup` or `errgroup` to coordinate concurrent calls?

If no concurrent test exists, the skill flags `D-CONCURRENCY-MISSING` for the SUT function.

### What the Skill Suggests

A stress-test scaffold appropriate for the SUT shape:

```go
func TestSafeMapConcurrent(t *testing.T) {
    m := NewSafeMap()
    const n = 1000
    var wg sync.WaitGroup
    wg.Add(2 * n)
    for i := 0; i < n; i++ {
        go func(i int) {
            defer wg.Done()
            m.Set(fmt.Sprintf("k%d", i), i)
        }(i)
        go func(i int) {
            defer wg.Done()
            _ = m.Get(fmt.Sprintf("k%d", i))
        }(i)
    }
    wg.Wait()
}
```

For state machines with concurrent inputs, the skill prefers `rapid.StateMachine` (see `reshape-to-invariants.md`).

### Common False Positives

- SUT uses goroutines internally but exposes a synchronous API. Sequential tests are correct here — the skill flags only if the *exposed contract* is concurrent.
- Test uses a real implementation that internally serializes. Skill heuristics catch this when an explicit lock guards the test's interactions.

## 2. Failure-Mode Coverage

### What Goes Wrong

For SUTs that take `context.Context` or return `error`:

- Tests cover the happy path but not cancellation, timeout, or specific error returns.
- Tests treat errors as "exists / doesn't exist" instead of "what kind of error".
- Networking/disk SUTs aren't tested under fault injection.

In distributed systems, "the unhappy path" is the actual production path most of the time. Jepsen's contribution was making this explicit: a healthy-cluster test isn't a test ([Jepsen](https://jepsen.io/)).

### What the Skill Looks For

For each SUT function in scope:

- Does it return `error`? If yes, do tests cover the error case (not just `NoError`)?
- Does it accept `context.Context`? If yes, do tests cover cancellation (`context.WithCancel` then `cancel()`) and timeout (`context.WithTimeout` with short duration)?
- Is the SUT a network or storage operation? If yes, are there tests using a fault-injecting fake?

Findings:

| Code | Meaning |
|---|---|
| `D-ERR-PATH-MISSING` | SUT returns `error`; no test asserts on a specific error |
| `D-CTX-CANCEL-MISSING` | SUT takes `context.Context`; no cancellation test |
| `D-CTX-TIMEOUT-MISSING` | Same, no timeout test |
| `D-FAULT-INJECTION-MISSING` | Network/storage SUT, no fault-injecting test |

### What the Skill Suggests

For error paths:

```go
func TestParseErrors(t *testing.T) {
    cases := []struct {
        name    string
        input   []byte
        wantErr error
    }{
        {"empty",        nil,                ErrEmpty},
        {"truncated",    []byte{0x01, 0x02}, ErrTruncated},
        {"bad magic",    []byte("XXXX"),     ErrInvalidMagic},
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            _, err := Parse(c.input)
            require.ErrorIs(t, err, c.wantErr)
        })
    }
}
```

For cancellation:

```go
func TestFetchContextCancel(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel()
    _, err := Fetch(ctx, url)
    require.ErrorIs(t, err, context.Canceled)
}
```

For fault injection (network):

```go
func TestPeerNetworkError(t *testing.T) {
    fake := &faultyConn{
        readErr: io.ErrUnexpectedEOF,
    }
    p := NewPeer(fake)
    err := p.SendPing()
    require.ErrorIs(t, err, ErrPeerDisconnected)
}
```

### Bitcoin/Lightning Specific Failure Modes

Distributed payment systems have failure modes worth specifically testing:

- **Channel partial state**: peer disconnects mid-handshake.
- **Mempool congestion**: fee estimator returns errors.
- **Signature verification failures**: malformed sig, wrong key.
- **HTLC stuck**: timeout never fires, force-close path.
- **Replay attacks**: same nonce / preimage reused.

The skill flags absence of these for code touching `channel`, `htlc`, `commitment`, `wire`, `mempool`, `fee`.

## 3. Property / Invariant Candidates

### What Goes Wrong

Many functions have invariants stronger than any single example test can express:

- `Marshal`/`Unmarshal` — round-trip identity.
- Parsers — equivalence with reference implementation.
- Comparators — total order, transitivity, anti-symmetry.
- Normalizers — idempotence (`f(f(x)) == f(x)`).
- Hash/derivation — collision rate, avalanche.

Example-based tests cover specific values; property-based tests cover the *space*.

### What the Skill Looks For

The skill identifies PBT candidates by SUT signature:

| Pattern | Property |
|---|---|
| `Marshal(T) -> []byte` paired with `Unmarshal([]byte) -> (T, error)` | Roundtrip: `Unmarshal(Marshal(x)) == x` |
| `Parse(string) -> (T, error)` paired with `Format(T) -> string` | Roundtrip: `Format(Parse(s)) == s` (where `s` is in the format's image) |
| `Normalize(T) -> T` | Idempotence: `Normalize(Normalize(x)) == Normalize(x)` |
| `Compare(T, T) -> int` | Anti-symmetry: `sign(Compare(a, b)) == -sign(Compare(b, a))`; transitivity |
| `Hash(T) -> [N]byte` | Determinism, distribution (qualitative) |
| State machine (struct with methods that mutate fields) | `rapid.StateMachine` over the operations |

Findings: `D-PBT-CANDIDATE` with the specific property template suggested.

### What the Skill Suggests

For roundtrips, see Pattern 13 in `strengthening-patterns.md`.

For state machines, see `reshape-to-invariants.md`.

### Cross-Reference

When a candidate is detected, the skill cross-references with the [`property-based-testing`](../../property-based-testing/SKILL.md) skill's reference docs for generator templates.

## 4. Determinism / Reproducibility

### What Goes Wrong

A test that depends on `time.Now()`, unseeded `rand`, environment variables, or goroutine scheduling cannot be reproduced. When it fails on CI, you can't debug it locally without the same wall-clock state.

This is the [DST](https://antithesis.com/docs/resources/deterministic_simulation_testing/) principle, codified by FoundationDB and TigerBeetle: control all sources of nondeterminism so a failing seed reproduces exactly.

### What the Skill Looks For

In test code:

| Pattern | Finding |
|---|---|
| `time.Now()` in test or SUT | `D-DETERMINISM-CLOCK` |
| `rand.Int()`, `rand.Intn()`, etc. without explicit seed | `D-DETERMINISM-RAND` |
| `os.Getenv()` in test path | `D-DETERMINISM-ENV` |
| `runtime.GOMAXPROCS` not set in concurrent test | `D-DETERMINISM-SCHED` (advisory) |
| Test reads files outside testdata | `D-DETERMINISM-EXTERNAL` |

In production code (advisory only):

- Functions that read `time.Now()` directly without an injectable clock.
- Functions that call `crypto/rand` or `math/rand` without an injectable source.

### What the Skill Suggests

Inject a clock:

```go
type Clock interface { Now() time.Time }

type realClock struct{}
func (realClock) Now() time.Time { return time.Now() }

type Service struct { clock Clock }

func New(opts ...Option) *Service {
    s := &Service{clock: realClock{}}
    for _, o := range opts { o(s) }
    return s
}

// Test
func TestExpiry(t *testing.T) {
    fc := clock.NewMock()
    fc.Set(must(time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")))
    s := New(WithClock(fc))
    // ... advance fc.Add(time.Hour) instead of sleeping ...
}
```

For RNG:

```go
type Service struct { rng *rand.Rand }
func New(seed int64) *Service { return &Service{rng: rand.New(rand.NewSource(seed))} }
```

For tests, use `rapid` with a deterministic seed (`rapid.Check` reseeds per case but reports the seed on failure for replay).

## Combining Domain Checks

A single SUT function can hit multiple dimensions:

- A network handler with goroutines, returning errors, computing a hash → all four dimensions apply.
- A pure function — usually only PBT candidacy.

The skill emits findings independently. The composite priority score accumulates: a high-risk function with three domain findings outranks a high-risk function with one.

## Sources

- [Jepsen testing distributed systems](https://jepsen.io/)
- [Deterministic Simulation Testing — Antithesis](https://antithesis.com/docs/resources/deterministic_simulation_testing/)
- [TigerBeetle's VOPR (DST)](https://tigerbeetle.com/blog/2023-07-11-we-put-a-distributed-database-in-the-browser/)
- [Linux kernel fault injection framework](https://docs.kernel.org/fault-injection/fault-injection.html)
- [Curated list of resources on testing distributed systems](https://asatarin.github.io/testing-distributed-systems/)
