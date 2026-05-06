# Gremlins Mutator Catalog

This document maps each gremlins mutator to the Go operators it targets, with examples and guidance on when to enable.

For configuration syntax, see [gremlins.dev/0.5/usage/configuration](https://gremlins.dev/0.5/usage/configuration/).

## Why Mutate?

A mutation is a small, syntactically valid change to production code that should — if the tests are doing their job — cause at least one test to fail. If no test fails, either the test suite has no assertion that depends on this code's behavior, or the change is semantically equivalent to the original.

Mutation testing is the empirical answer to "did the test actually verify anything?" — line and branch coverage cannot answer this.

## Mutator Reference

### Default-On Mutators

These run by default. Disable only with strong justification.

#### `arithmetic-base`

Swaps among `+`, `-`, `*`, `/`, `%`.

```go
// Original
fee := amount * rate / 10000

// Mutants
fee := amount + rate / 10000   // * → +
fee := amount - rate / 10000   // * → -
fee := amount / rate / 10000   // * → /
fee := amount * rate * 10000   // / → *
```

**Catches**: incorrect arithmetic in fee calculations, scaling, indexing.
**Recommended**: always on. Especially critical in financial code.

#### `conditionals-boundary`

Boundary changes: `<` ↔ `<=`, `>` ↔ `>=`.

```go
// Original
if balance >= threshold { ... }

// Mutant
if balance > threshold { ... }   // off-by-one at exact threshold
```

**Catches**: off-by-one bugs at thresholds. Tests must hit the exact boundary value to kill.
**Recommended**: always on.

#### `conditionals-negation`

Negates conditions and equality: `==` ↔ `!=`, boolean conditions inverted.

```go
// Original
if err != nil { return err }

// Mutant
if err == nil { return err }
```

**Catches**: tests that don't exercise both branches, or that have no assertion on the branch result.
**Recommended**: always on.

#### `increment-decrement`

`++` ↔ `--`, `+= 1` ↔ `-= 1` patterns.

```go
// Original
counter++

// Mutant
counter--
```

**Catches**: loop termination bugs, counter direction errors.
**Recommended**: always on.

#### `invert-negatives`

Removes or adds unary negation: `-x` ↔ `+x` (or just `x`).

```go
// Original
balance := -amount

// Mutant
balance := amount
```

**Catches**: sign-handling bugs.
**Recommended**: always on.

### Default-Off Mutators

These are aggressive — enable for critical packages, leave off for trivial code.

#### `invert-assignments`

Swaps compound-assignment operators: `+=` ↔ `-=`, `*=` ↔ `/=`, etc.

```go
// Original
balance += deposit

// Mutant
balance -= deposit
```

**Catches**: tests that don't assert on the post-assignment state.
**Recommended**: enable for state-mutating code (wallets, accumulators, counters).

#### `invert-bitwise`

Swaps `&` ↔ `|`, `^` mutations.

```go
// Original
flags := flagA | flagB

// Mutant
flags := flagA & flagB
```

**Catches**: bitmap / flag mask bugs.
**Recommended**: enable for protocol code (Bitcoin script, P2P feature bits, BOLT feature flags).

#### `invert-bwassign`

Bitwise compound-assign swaps: `&=` ↔ `|=`, `^=`.

```go
// Original
state |= READY

// Mutant
state &= READY
```

**Catches**: state flag manipulation bugs.
**Recommended**: enable for state-machine code.

#### `invert-logical`

`&&` ↔ `||`. **Security-critical** — security checks frequently rely on `&&`.

```go
// Original
if user.IsAuthenticated() && user.IsAuthorized(resource) { ... }

// Mutant
if user.IsAuthenticated() || user.IsAuthorized(resource) { ... }
//   ^ now any authenticated user is "authorized"
```

**Catches**: auth-bypass-class bugs, missing test cases that exercise individual conjuncts.
**Recommended**: **always enable for security-touching code**. Often catches the most consequential test gaps.

#### `invert-loopctrl`

`break` ↔ `continue` in loops.

```go
// Original
for _, peer := range peers {
    if peer.Banned {
        continue
    }
    process(peer)
}

// Mutant
for _, peer := range peers {
    if peer.Banned {
        break   // stops processing on first banned peer
    }
    process(peer)
}
```

**Catches**: loop control flow bugs.
**Recommended**: enable for code processing collections of independent items.

#### `remove-self-assignments`

Removes `x op= y` style updates entirely.

```go
// Original
total += line.Amount

// Mutant
// (assignment removed)
```

**Catches**: tests that don't assert on the accumulated/updated value.
**Recommended**: enable for accumulators and state machines.

## Mutator Enablement Recipes

### Default profile (most code)

```yaml
mutants:
  arithmetic-base:       { enabled: true }
  conditionals-boundary: { enabled: true }
  conditionals-negation: { enabled: true }
  increment-decrement:   { enabled: true }
  invert-negatives:      { enabled: true }
```

### Critical-path profile (consensus, wallet, channel, crypto)

Enable everything:

```yaml
mutants:
  arithmetic-base:         { enabled: true }
  conditionals-boundary:   { enabled: true }
  conditionals-negation:   { enabled: true }
  increment-decrement:     { enabled: true }
  invert-negatives:        { enabled: true }
  invert-assignments:      { enabled: true }
  invert-bitwise:          { enabled: true }
  invert-bwassign:         { enabled: true }
  invert-logical:          { enabled: true }
  invert-loopctrl:         { enabled: true }
  remove-self-assignments: { enabled: true }
```

### Security-touching code

At minimum enable `invert-logical` on top of defaults — it's the highest-value mutator for catching auth/authorization test gaps.

```yaml
mutants:
  arithmetic-base:       { enabled: true }
  conditionals-boundary: { enabled: true }
  conditionals-negation: { enabled: true }
  increment-decrement:   { enabled: true }
  invert-negatives:      { enabled: true }
  invert-logical:        { enabled: true }
```

## Mapping `LIVED` Survivors to Test Improvements

When a mutator survives, the implied missing test depends on which mutator:

| Survivor | Implied test gap |
|---|---|
| `arithmetic-base` | No assertion on calculated result |
| `conditionals-boundary` | No test at exact threshold |
| `conditionals-negation` | Branch executed but result not asserted |
| `increment-decrement` | Loop count or counter not verified |
| `invert-negatives` | Sign of result not asserted |
| `invert-assignments` | Post-update state not read back |
| `invert-bitwise` | Resulting flags not checked |
| `invert-bwassign` | State machine transition not verified |
| `invert-logical` | Need test that exercises each conjunct independently |
| `invert-loopctrl` | Order/completeness of iteration not verified |
| `remove-self-assignments` | Accumulated value not asserted |

This mapping is what `test-refine` consumes when it cross-references mutation survivors with AST-detected smells.

## Equivalent Mutants

Not every `LIVED` mutant is a real test gap. Equivalent mutants change syntax but not observable behavior. Common cases:

```go
// Original: a + b + c
// Mutated:  a + (b + c)   ← associativity; equivalent for ints

// Original: x = a; x = b
// Mutated:  x = a + 1; x = b   ← first assignment dead

// Original: if alwaysTrue { ... }
// Mutated:  if !alwaysTrue { ... }   ← if alwaysTrue is unreachable mutation, equivalent
```

Document equivalents in the project to avoid re-investigating them. Gremlins does not filter equivalents automatically.

## Further Reading

- [Gremlins documentation](https://gremlins.dev/) — authoritative reference.
- [PITest](https://pitest.org/) — Java mutation tool that inspired gremlins; its docs cover mutator semantics in depth.
- "Are Mutants a Valid Substitute for Real Faults in Software Testing?" (Just et al., FSE 2014) — empirical justification.
