# Input Strategy Reference

## Go/rapid (Primary for Go)

| Type | Generator |
|------|-----------|
| `int` | `rapid.Int()` |
| `int` (bounded) | `rapid.IntRange(0, 100)` |
| `int64` | `rapid.Int64()` |
| `uint64` | `rapid.Uint64()` |
| `float64` | `rapid.Float64()` |
| `bool` | `rapid.Bool()` |
| `string` | `rapid.String()` |
| `string` (bounded) | `rapid.StringN(minLen, maxLen, maxRunes)` |
| `byte` | `rapid.Byte()` |
| `[]byte` | `rapid.SliceOf(rapid.Byte())` |
| `[]T` | `rapid.SliceOf(genT)` |
| `[]T` (bounded) | `rapid.SliceOfN(genT, minLen, maxLen)` |
| `map[K]V` | `rapid.MapOf(keyGen, valGen)` |
| `*T` | `rapid.Ptr(genT, allowNil)` |
| One of values | `rapid.SampledFrom([]T{...})` |
| One of generators | `rapid.OneOf(gen1, gen2)` |
| Constant | `rapid.Just(value)` |
| Transform | `rapid.Map(gen, func)` |
| Custom struct | `rapid.Custom(func(t *rapid.T) T { ... })` |

### Custom Generators (rapid.Custom)

For complex types, use `rapid.Custom`:

```go
func genUser() *rapid.Generator[User] {
    return rapid.Custom(func(t *rapid.T) User {
        return User{
            Name:  rapid.StringN(1, 50, -1).Draw(t, "name"),
            Age:   rapid.IntRange(0, 150).Draw(t, "age"),
            Email: rapid.StringN(5, 100, -1).Draw(t, "email"),
            Role:  rapid.SampledFrom([]Role{Admin, User, Guest}).Draw(t, "role"),
        }
    })
}
```

### Filtered Generators (rapid.Filter)

When you need to constrain values but cannot express it directly:

```go
// Only even numbers.
evenInts := rapid.Int().Filter(func(n int) bool { return n%2 == 0 })

// Non-empty strings.
nonEmpty := rapid.String().Filter(func(s string) bool { return len(s) > 0 })
```

Note: Prefer `rapid.IntRange`/`rapid.StringN` over `Filter` when possible, as filters may discard many values.

### Recursive/Nested Generators

```go
// Generate a tree structure.
func genTree() *rapid.Generator[*TreeNode] {
    return rapid.Custom(func(t *rapid.T) *TreeNode {
        depth := rapid.IntRange(0, 5).Draw(t, "depth")
        return genTreeAtDepth(depth).Draw(t, "tree")
    })
}

func genTreeAtDepth(maxDepth int) *rapid.Generator[*TreeNode] {
    return rapid.Custom(func(t *rapid.T) *TreeNode {
        node := &TreeNode{
            Value: rapid.Int().Draw(t, "value"),
        }
        if maxDepth > 0 {
            nChildren := rapid.IntRange(0, 3).Draw(t, "nChildren")
            for i := 0; i < nChildren; i++ {
                child := genTreeAtDepth(maxDepth - 1).Draw(t, fmt.Sprintf("child_%d", i))
                node.Children = append(node.Children, child)
            }
        }
        return node
    })
}
```

### Stateful Testing Generators

For `rapid.StateMachine`, each method acts as a command with its own generators:

```go
func (m *myMachine) Insert(t *rapid.T) {
    key := rapid.StringN(1, 20, -1).Draw(t, "key")
    val := rapid.Int().Draw(t, "val")
    m.sut.Insert(key, val)
    m.model[key] = val
}
```

## Python/Hypothesis

| Type | Strategy |
|------|----------|
| `int` | `st.integers()` |
| `float` | `st.floats(allow_nan=False)` |
| `str` | `st.text()` |
| `bytes` | `st.binary()` |
| `bool` | `st.booleans()` |
| `list[T]` | `st.lists(strategy_for_T)` |
| `dict[K, V]` | `st.dictionaries(key_strategy, value_strategy)` |
| `set[T]` | `st.frozensets(strategy_for_T)` |
| `tuple[T, ...]` | `st.tuples(strategy_for_T, ...)` |
| `Optional[T]` | `st.none() \| strategy_for_T` |
| `Union[A, B]` | `st.one_of(strategy_a, strategy_b)` |
| Custom class | `st.builds(ClassName, field1=..., field2=...)` |
| Enum | `st.sampled_from(EnumClass)` |
| Constrained int | `st.integers(min_value=0, max_value=100)` |
| Email | `st.emails()` |
| UUID | `st.uuids()` |
| DateTime | `st.datetimes()` |
| Regex match | `st.from_regex(r"pattern")` |

### Composite Strategies

For complex types, use `@st.composite`:

```python
@st.composite
def valid_users(draw):
    name = draw(st.text(min_size=1, max_size=50))
    age = draw(st.integers(min_value=0, max_value=150))
    email = draw(st.emails())
    return User(name=name, age=age, email=email)
```

## JavaScript/fast-check

| Type | Strategy |
|------|----------|
| number | `fc.integer()` or `fc.float()` |
| string | `fc.string()` |
| boolean | `fc.boolean()` |
| array | `fc.array(itemArb)` |
| object | `fc.record({...})` |
| optional | `fc.option(arb)` |

### Example

```typescript
const userArb = fc.record({
  name: fc.string({ minLength: 1, maxLength: 50 }),
  age: fc.integer({ min: 0, max: 150 }),
  email: fc.emailAddress(),
});
```

## Rust/proptest

| Type | Strategy |
|------|----------|
| i32, u64, etc | `any::<i32>()` |
| String | `any::<String>()` or `"[a-z]+"` (regex) |
| Vec<T> | `prop::collection::vec(strategy, size)` |
| Option<T> | `prop::option::of(strategy)` |

### Example

```rust
proptest! {
    #[test]
    fn test_roundtrip(s in "[a-z]{1,20}") {
        let encoded = encode(&s);
        let decoded = decode(&encoded)?;
        prop_assert_eq!(s, decoded);
    }
}
```

## Best Practices

1. **Constrain early**: Build constraints into strategy, not via filters
   ```go
   // GOOD
   rapid.IntRange(1, 100)

   // BAD
   rapid.Int().Filter(func(n int) bool { return n >= 1 && n <= 100 })
   ```

   ```python
   # GOOD
   st.integers(min_value=1, max_value=100)

   # BAD
   st.integers().filter(lambda x: 1 <= x <= 100)
   ```

2. **Size limits**: Use bounded generators to prevent slow tests
   ```go
   rapid.SliceOfN(rapid.Int(), 0, 100)
   rapid.StringN(0, 1000, -1)
   ```

3. **Realistic data**: Make strategies match real-world constraints
   ```go
   // Real user ages, not arbitrary integers.
   rapid.IntRange(0, 150)
   ```

4. **Reuse generators**: Define once, use across tests
   ```go
   var genUser = rapid.Custom(func(t *rapid.T) User { ... })

   func TestOne(t *testing.T) {
       rapid.Check(t, func(t *rapid.T) {
           user := genUser.Draw(t, "user")
           // ...
       })
   }

   func TestTwo(t *testing.T) {
       rapid.Check(t, func(t *rapid.T) {
           user := genUser.Draw(t, "user")
           // ...
       })
   }
   ```
