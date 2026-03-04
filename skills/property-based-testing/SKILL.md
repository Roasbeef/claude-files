---
name: property-based-testing
description: Provides guidance for property-based testing across multiple languages and smart contracts. Use when writing tests, reviewing code with serialization/validation/parsing patterns, designing features, or when property-based testing would provide stronger coverage than example-based tests. For Go code, uses pgregory.net/rapid as the primary PBT framework.
---

# Property-Based Testing Guide

Use this skill proactively during development when you encounter patterns where PBT provides stronger coverage than example-based tests.

## Go / rapid (Primary for Go Code)

For Go code, **always use `pgregory.net/rapid`** as the property-based testing library. Do NOT use `testing/quick` -- it is limited, lacks proper shrinking, and has poor generator support.

### Why rapid over testing/quick

- Automatic shrinking of failing cases to minimal counterexamples.
- Rich generator combinators: `IntRange`, `SliceOf`, `Map`, `Custom`, `OneOf`, `Ptr`, `SampledFrom`.
- Stateful testing via `rapid.StateMachine` for testing stateful systems.
- Integrates with standard `*testing.T` via `rapid.Check`.
- Actively maintained with good documentation.

### rapid Core Pattern

```go
func TestRoundtrip(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        msg := genMessage().Draw(t, "msg")
        encoded := Encode(msg)
        decoded, err := Decode(encoded)
        if err != nil {
            t.Fatal(err)
        }
        if !reflect.DeepEqual(msg, decoded) {
            t.Fatalf("roundtrip failed: got %v, want %v", decoded, msg)
        }
    })
}
```

### rapid Generators

| Generator | Description | Example |
|-----------|-------------|---------|
| `rapid.Int()` | Any int | `rapid.Int().Draw(t, "n")` |
| `rapid.IntRange(lo, hi)` | Bounded int | `rapid.IntRange(0, 100).Draw(t, "n")` |
| `rapid.Int64()` | Any int64 | `rapid.Int64().Draw(t, "n")` |
| `rapid.Uint64()` | Any uint64 | `rapid.Uint64().Draw(t, "n")` |
| `rapid.Float64()` | Any float64 | `rapid.Float64().Draw(t, "f")` |
| `rapid.Bool()` | true or false | `rapid.Bool().Draw(t, "b")` |
| `rapid.String()` | Any string | `rapid.String().Draw(t, "s")` |
| `rapid.StringN(minLen, maxLen, maxRunes)` | Bounded string | `rapid.StringN(1, 50, -1).Draw(t, "s")` |
| `rapid.Byte()` | Single byte | `rapid.Byte().Draw(t, "b")` |
| `rapid.SliceOf(gen)` | Slice of elements | `rapid.SliceOf(rapid.Int()).Draw(t, "xs")` |
| `rapid.SliceOfN(gen, min, max)` | Bounded slice | `rapid.SliceOfN(rapid.Int(), 1, 10).Draw(t, "xs")` |
| `rapid.MapOf(keyGen, valGen)` | Map | `rapid.MapOf(rapid.String(), rapid.Int()).Draw(t, "m")` |
| `rapid.SampledFrom(slice)` | One of values | `rapid.SampledFrom([]string{"a","b"}).Draw(t, "s")` |
| `rapid.OneOf(gens...)` | One of generators | `rapid.OneOf(rapid.Int(), rapid.Int()).Draw(t, "n")` |
| `rapid.Just(val)` | Constant value | `rapid.Just(42).Draw(t, "n")` |
| `rapid.Ptr(gen, allowNil)` | Pointer | `rapid.Ptr(rapid.Int(), true).Draw(t, "p")` |
| `rapid.Map(gen, fn)` | Transform | `rapid.Map(rapid.Int(), func(n int) uint { return uint(n&0xff) })` |
| `rapid.Custom(fn)` | Custom generator | See below |

### Custom Generators

```go
func genMessage() *rapid.Generator[Message] {
    return rapid.Custom(func(t *rapid.T) Message {
        return Message{
            ID:       rapid.IntRange(1, 10000).Draw(t, "id"),
            Content:  rapid.StringN(0, 1000, -1).Draw(t, "content"),
            Priority: rapid.IntRange(1, 10).Draw(t, "priority"),
            Tags:     rapid.SliceOfN(rapid.StringN(1, 50, -1), 0, 20).Draw(t, "tags"),
        }
    })
}
```

### Stateful Testing with rapid.StateMachine

For testing stateful systems (data structures, databases, protocol state machines):

```go
type queueMachine struct {
    queue Queue           // System under test.
    model []int           // Reference model.
}

func (m *queueMachine) Init(t *rapid.T) {
    m.queue = NewQueue()
    m.model = nil
}

func (m *queueMachine) Push(t *rapid.T) {
    v := rapid.Int().Draw(t, "v")
    m.queue.Push(v)
    m.model = append(m.model, v)
}

func (m *queueMachine) Pop(t *rapid.T) {
    if len(m.model) == 0 {
        t.Skip("empty queue")
    }
    got := m.queue.Pop()
    want := m.model[0]
    m.model = m.model[1:]
    if got != want {
        t.Fatalf("got %d, want %d", got, want)
    }
}

func (m *queueMachine) Check(t *rapid.T) {
    if m.queue.Len() != len(m.model) {
        t.Fatalf("length mismatch: got %d, want %d", m.queue.Len(), len(m.model))
    }
}

func TestQueue(t *testing.T) {
    rapid.Check(t, rapid.Run[*queueMachine]())
}
```

### Go Property Test Patterns

**Roundtrip (encode/decode):**
```go
func TestCodecRoundtrip(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        original := genMessage().Draw(t, "msg")
        encoded, err := Encode(original)
        if err != nil {
            t.Fatal(err)
        }
        decoded, err := Decode(encoded)
        if err != nil {
            t.Fatal(err)
        }
        if !reflect.DeepEqual(original, decoded) {
            t.Fatalf("roundtrip failed: %v != %v", original, decoded)
        }
    })
}
```

**Idempotence (normalization):**
```go
func TestNormalizeIdempotent(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        s := rapid.String().Draw(t, "s")
        once := Normalize(s)
        twice := Normalize(once)
        if once != twice {
            t.Fatalf("not idempotent: Normalize(%q) = %q, Normalize(%q) = %q",
                s, once, once, twice)
        }
    })
}
```

**Sorting properties:**
```go
func TestSort(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        xs := rapid.SliceOf(rapid.Int()).Draw(t, "xs")
        result := MySort(xs)

        // Length preserved.
        if len(result) != len(xs) {
            t.Fatalf("length changed: %d -> %d", len(xs), len(result))
        }

        // Ordered.
        for i := 1; i < len(result); i++ {
            if result[i-1] > result[i] {
                t.Fatalf("not sorted at index %d: %d > %d", i, result[i-1], result[i])
            }
        }

        // Idempotent.
        result2 := MySort(result)
        if !reflect.DeepEqual(result, result2) {
            t.Fatal("sort not idempotent")
        }
    })
}
```

**Oracle (reference implementation):**
```go
func TestOptimizedMatchesReference(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        input := rapid.SliceOf(rapid.Int()).Draw(t, "input")
        got := OptimizedImpl(input)
        want := ReferenceImpl(input)
        if !reflect.DeepEqual(got, want) {
            t.Fatalf("mismatch: optimized=%v, reference=%v", got, want)
        }
    })
}
```

## When to Invoke (Automatic Detection)

**Invoke this skill when you detect:**

- **Serialization pairs**: `encode`/`decode`, `serialize`/`deserialize`, `toJSON`/`fromJSON`, `pack`/`unpack`, `Marshal`/`Unmarshal`
- **Parsers**: URL parsing, config parsing, protocol parsing, string-to-structured-data
- **Normalization**: `normalize`, `sanitize`, `clean`, `canonicalize`, `format`
- **Validators**: `is_valid`, `validate`, `check_*` (especially with normalizers)
- **Data structures**: Custom collections with `add`/`remove`/`get` operations
- **Mathematical/algorithmic**: Pure functions, sorting, ordering, comparators
- **Smart contracts**: Solidity/Vyper contracts, token operations, state invariants, access control
- **State machines**: Protocol state, connection lifecycle, workflow transitions (use `rapid.StateMachine`)

**Priority by pattern:**

| Pattern | Property | Priority |
|---------|----------|----------|
| encode/decode pair | Roundtrip | HIGH |
| Pure function | Multiple | HIGH |
| State machine | Invariants via StateMachine | HIGH |
| Validator | Valid after normalize | MEDIUM |
| Sorting/ordering | Idempotence + ordering | MEDIUM |
| Normalization | Idempotence | MEDIUM |
| Builder/factory | Output invariants | LOW |
| Smart contract | State invariants | HIGH |

## When NOT to Use

Do NOT use this skill for:
- Simple CRUD operations without transformation logic
- One-off scripts or throwaway code
- Code with side effects that cannot be isolated (network calls, database writes)
- Tests where specific example cases are sufficient and edge cases are well-understood
- Integration or end-to-end testing (PBT is best for unit/component testing)

## Property Catalog (Quick Reference)

| Property | Formula | When to Use |
|----------|---------|-------------|
| **Roundtrip** | `decode(encode(x)) == x` | Serialization, conversion pairs |
| **Idempotence** | `f(f(x)) == f(x)` | Normalization, formatting, sorting |
| **Invariant** | Property holds before/after | Any transformation |
| **Commutativity** | `f(a, b) == f(b, a)` | Binary/set operations |
| **Associativity** | `f(f(a,b), c) == f(a, f(b,c))` | Combining operations |
| **Identity** | `f(x, identity) == x` | Operations with neutral element |
| **Inverse** | `f(g(x)) == x` | encrypt/decrypt, compress/decompress |
| **Oracle** | `new_impl(x) == reference(x)` | Optimization, refactoring |
| **Easy to Verify** | `is_sorted(sort(x))` | Complex algorithms |
| **No Exception** | No crash on valid input | Baseline property |

**Strength hierarchy** (weakest to strongest):
No Exception -> Type Preservation -> Invariant -> Idempotence -> Roundtrip

## Decision Tree

Based on the current task, read the appropriate section:

```
TASK: Writing new tests
  -> Read [{baseDir}/references/generating.md]({baseDir}/references/generating.md) (test generation patterns and examples)
  -> Then [{baseDir}/references/strategies.md]({baseDir}/references/strategies.md) if input generation is complex

TASK: Designing a new feature
  -> Read [{baseDir}/references/design.md]({baseDir}/references/design.md) (Property-Driven Development approach)

TASK: Code is difficult to test (mixed I/O, missing inverses)
  -> Read [{baseDir}/references/refactoring.md]({baseDir}/references/refactoring.md) (refactoring patterns for testability)

TASK: Reviewing existing PBT tests
  -> Read [{baseDir}/references/reviewing.md]({baseDir}/references/reviewing.md) (quality checklist and anti-patterns)

TASK: Test failed, need to interpret
  -> Read [{baseDir}/references/interpreting-failures.md]({baseDir}/references/interpreting-failures.md) (failure analysis and bug classification)

TASK: Need library reference
  -> Read [{baseDir}/references/libraries.md]({baseDir}/references/libraries.md) (PBT libraries by language, includes smart contract tools)
```

## How to Suggest PBT

When you detect a high-value pattern while writing tests, **offer PBT as an option**:

> "I notice `Encode`/`Decode` is a serialization pair. Property-based testing with a roundtrip property using `rapid.Check` would provide stronger coverage than example tests. Want me to use that approach?"

**If codebase already uses a PBT library** (rapid, Hypothesis, fast-check, proptest, Echidna), be more direct:

> "This codebase uses rapid. I'll write property-based tests for this serialization pair using a roundtrip property."

**If Go codebase uses `testing/quick`**, suggest migration:

> "I see this codebase uses `testing/quick`. The `rapid` library (`pgregory.net/rapid`) provides better shrinking, richer generators, and stateful testing support. Want me to use rapid for the new tests?"

**If user declines**, write good example-based tests without further prompting.

## When NOT to Use PBT

- Simple CRUD without complex validation
- UI/presentation logic
- Integration tests requiring complex external setup
- Prototyping where requirements are fluid
- User explicitly requests example-based tests only

## Red Flags

- Recommending trivial getters/setters
- Missing paired operations (encode without decode)
- Ignoring type hints (well-typed = easier to test)
- Overwhelming user with candidates (limit to top 5-10)
- Being pushy after user declines

## Rationalizations to Reject

Do not accept these shortcuts:

- **"Example tests are good enough"** - If serialization/parsing/normalization is involved, PBT finds edge cases examples miss
- **"The function is simple"** - Simple functions with complex input domains (strings, floats, nested structures) benefit most from PBT
- **"We don't have time"** - PBT tests are often shorter than comprehensive example suites
- **"It's too hard to write generators"** - rapid has excellent built-in generators and `rapid.Custom` makes custom ones trivial
- **"The test failed, so it's a bug"** - Failures require validation; see [{baseDir}/references/interpreting-failures.md]({baseDir}/references/interpreting-failures.md)
- **"No crash means it works"** - "No exception" is the weakest property; always push for stronger guarantees
- **"testing/quick is good enough"** - It lacks proper shrinking, has limited generators, and is effectively unmaintained; use rapid instead
