# Generating Property-Based Tests

How to create complete, runnable property-based tests.

## Process

### 1. Analyze Target Function

- Read function signature, types, and docstrings
- Understand input types and constraints
- Identify output type and expected behavior
- Note preconditions or invariants
- Check existing example-based tests as hints

### 2. Design Input Strategies

Create appropriate generator strategies for each input parameter.

**Principles**:
- Build constraints INTO the strategy, not via `assume()` / `t.Skip()`
- Use realistic size limits to prevent slow tests
- Match real-world constraints

### 3. Identify Applicable Properties

| Property | When to Use | Test Pattern |
|----------|-------------|--------------|
| Roundtrip | encode/decode pairs | `assert decode(encode(x)) == x` |
| Idempotence | normalization, sorting | `assert f(f(x)) == f(x)` |
| Invariant | any transformation | `assert invariant(f(x))` |
| No exception | all functions (weak) | Function completes without raising |
| Type preservation | typed functions | `assert isinstance(f(x), ExpectedType)` |
| Length preservation | collections | `assert len(f(xs)) == len(xs)` |
| Element preservation | sorting, shuffling | `assert set(f(xs)) == set(xs)` |
| Ordering | sorting | `assert all(f(xs)[i] <= f(xs)[i+1] ...)` |
| Oracle | when reference exists | `assert f(x) == reference_impl(x)` |
| Commutativity | binary ops | `assert f(a, b) == f(b, a)` |

### 4. Generate Test Code

Create test functions with:
- Clear comments explaining what each property verifies
- Appropriate generator constraints for the context
- Explicit edge case sub-tests where needed

### 5. Include Edge Cases

For Go/rapid, add explicit sub-tests for critical edge cases alongside property tests:

```go
func TestMyFunc(t *testing.T) {
    // Property test for general case.
    rapid.Check(t, func(t *rapid.T) {
        // ... property test ...
    })

    // Explicit edge cases.
    edgeCases := []struct{
        name  string
        input []int
    }{
        {"empty", nil},
        {"single", []int{1}},
        {"duplicates", []int{1, 1, 1}},
        {"zero", []int{0}},
        {"negative", []int{-1}},
    }
    for _, tc := range edgeCases {
        t.Run(tc.name, func(t *testing.T) {
            // ... test edge case ...
        })
    }
}
```

For Python/Hypothesis, use `@example` decorators:
```python
@example([])           # Empty
@example([1])          # Single element
@example([1, 1, 1])    # Duplicates
@example("")           # Empty string
@example(0)            # Zero
@example(-1)           # Negative
```

## Example Test Patterns

### Roundtrip (Encode/Decode) - Go/rapid

```go
func TestMessageRoundtrip(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        original := genMessage().Draw(t, "msg")
        encoded, err := EncodeMessage(original)
        if err != nil {
            t.Fatal(err)
        }
        decoded, err := DecodeMessage(encoded)
        if err != nil {
            t.Fatal(err)
        }
        if !reflect.DeepEqual(original, decoded) {
            t.Fatalf("roundtrip failed: %+v != %+v", original, decoded)
        }
    })
}

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

### Idempotence - Go/rapid

```go
func TestNormalizeIdempotent(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        s := rapid.String().Draw(t, "s")
        once := Normalize(s)
        twice := Normalize(once)
        if once != twice {
            t.Fatalf("not idempotent: Normalize(%q)=%q, Normalize(%q)=%q",
                s, once, once, twice)
        }
    })
}
```

### Sorting Properties - Go/rapid

```go
func TestSort(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        xs := rapid.SliceOf(rapid.Int()).Draw(t, "xs")
        result := MySort(append([]int{}, xs...)) // Copy to avoid mutation issues.

        // Length preserved.
        if len(result) != len(xs) {
            t.Fatalf("length: got %d, want %d", len(result), len(xs))
        }

        // Elements preserved.
        sortedOriginal := append([]int{}, xs...)
        sort.Ints(sortedOriginal)
        sortedResult := append([]int{}, result...)
        sort.Ints(sortedResult)
        if !reflect.DeepEqual(sortedOriginal, sortedResult) {
            t.Fatal("elements not preserved")
        }

        // Ordered.
        for i := 1; i < len(result); i++ {
            if result[i-1] > result[i] {
                t.Fatalf("not sorted at %d: %d > %d", i, result[i-1], result[i])
            }
        }

        // Idempotent.
        result2 := MySort(append([]int{}, result...))
        if !reflect.DeepEqual(result, result2) {
            t.Fatal("not idempotent")
        }
    })
}
```

### Validator + Normalizer - Go/rapid

```go
func TestNormalizedIsValid(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        input := genValidInput().Draw(t, "input")
        normalized := Normalize(input)
        if !IsValid(normalized) {
            t.Fatalf("Normalize(%v) produced invalid result: %v", input, normalized)
        }
    })
}
```

### Roundtrip (Python/Hypothesis)

```python
@given(valid_messages())
def test_roundtrip(msg):
    """Encoding then decoding returns original."""
    assert decode(encode(msg)) == msg
```

### Idempotence (Python/Hypothesis)

```python
@given(st.text())
def test_normalize_idempotent(s):
    """Normalizing twice equals normalizing once."""
    assert normalize(normalize(s)) == normalize(s)
```

## Complete Example (Go/rapid)

```go
package codec_test

import (
    "reflect"
    "testing"

    "pgregory.net/rapid"

    "myapp/codec"
)

func genMessage() *rapid.Generator[codec.Message] {
    return rapid.Custom(func(t *rapid.T) codec.Message {
        return codec.Message{
            ID:       rapid.IntRange(1, 100000).Draw(t, "id"),
            Content:  rapid.StringN(0, 1000, -1).Draw(t, "content"),
            Priority: rapid.IntRange(1, 10).Draw(t, "priority"),
            Tags:     rapid.SliceOfN(rapid.StringN(1, 50, -1), 0, 20).Draw(t, "tags"),
        }
    })
}

func TestMessageCodecRoundtrip(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        msg := genMessage().Draw(t, "msg")
        encoded, err := codec.EncodeMessage(msg)
        if err != nil {
            t.Fatal(err)
        }
        decoded, err := codec.DecodeMessage(encoded)
        if err != nil {
            t.Fatal(err)
        }
        if !reflect.DeepEqual(msg, decoded) {
            t.Fatalf("roundtrip failed: %+v != %+v", msg, decoded)
        }
    })
}

func TestMessageEncodeDeterministic(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        msg := genMessage().Draw(t, "msg")
        enc1, err1 := codec.EncodeMessage(msg)
        enc2, err2 := codec.EncodeMessage(msg)
        if err1 != nil || err2 != nil {
            t.Fatalf("encode errors: %v, %v", err1, err2)
        }
        if !reflect.DeepEqual(enc1, enc2) {
            t.Fatal("encoding is not deterministic")
        }
    })
}

func TestMessageDecodeInvalidInput(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Random bytes should either decode successfully or return an error,
        // never panic.
        data := rapid.SliceOf(rapid.Byte()).Draw(t, "data")
        _, _ = codec.DecodeMessage(data)
    })
}
```

## Complete Example (Python/Hypothesis)

```python
"""Property-based tests for message_codec module."""
from hypothesis import given, strategies as st, settings, example
import pytest

from myapp.codec import encode_message, decode_message, Message, DecodeError

# Custom strategy for Message objects
messages = st.builds(
    Message,
    id=st.uuids(),
    content=st.text(max_size=1000),
    priority=st.integers(min_value=1, max_value=10),
    tags=st.lists(st.text(max_size=50), max_size=20),
)


class TestMessageCodecProperties:
    """Property-based tests for message encoding/decoding."""

    @given(messages)
    def test_roundtrip(self, msg: Message):
        """Encoding then decoding returns the original message."""
        encoded = encode_message(msg)
        decoded = decode_message(encoded)
        assert decoded == msg

    @given(messages)
    def test_encode_deterministic(self, msg: Message):
        """Same message always encodes to same bytes."""
        assert encode_message(msg) == encode_message(msg)

    @given(messages)
    def test_encoded_is_bytes(self, msg: Message):
        """Encoding produces bytes."""
        assert isinstance(encode_message(msg), bytes)

    @given(st.binary())
    def test_decode_invalid_raises_or_succeeds(self, data: bytes):
        """Random bytes either decode or raise DecodeError."""
        try:
            decode_message(data)
        except DecodeError:
            pass  # Expected for invalid input
```

## Running Tests

```bash
# Go - run all property tests
go test -v -run TestMessageCodec ./...

# Go - run with more iterations (CI)
go test -v -rapid.checks=1000 ./...

# Go - reproduce a specific failure
go test -v -run TestMessageCodecRoundtrip -rapid.seed=<seed>

# Python - run all property tests
pytest test_file.py -v

# Python - run with more examples (CI)
pytest test_file.py --hypothesis-seed=0 -v
```

## Checklist Before Finishing

- [ ] Tests are not tautological (don't reimplement the function)
- [ ] At least one strong property (not just "no crash")
- [ ] Edge cases covered with explicit sub-tests or `@example` decorators
- [ ] Strategy constraints are realistic, not over-filtered
- [ ] Iteration count appropriate for context (dev vs CI)
- [ ] Comments/docstrings explain what each property verifies
- [ ] Tests actually run and pass (or fail for expected reasons)

## Red Flags

- **Reimplementing the function**: If your assertion contains the same logic as the function under test, you've written a tautology
- **Only testing "no crash"**: This is the weakest property - always look for stronger ones first
- **Overly constrained strategies**: If you're using multiple `t.Skip()` / `assume()` calls, redesign the strategy instead
- **Missing edge cases**: No explicit edge case tests for empty, single-element, or boundary values
- **Using testing/quick in Go**: Always use rapid instead for better shrinking and generators

## When Tests Fail

See [{baseDir}/references/interpreting-failures.md]({baseDir}/references/interpreting-failures.md) for how to interpret failures and determine if they represent genuine bugs vs test errors vs ambiguous specifications.

For Go/rapid, the framework will print the shrunk minimal failing input and a seed you can use to reproduce:
```
--- FAIL: TestRoundtrip (0.02s)
    rapid: [rapid] draw msg: {ID:0 Content:"\x00" Priority:1 Tags:[]}
    rapid: [rapid] seed: 12345678
```

Use `-rapid.seed=12345678` to reproduce the exact failure.
