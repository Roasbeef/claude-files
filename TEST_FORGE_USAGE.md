# Test Forge - Advanced Test Generation Guide

## Overview

Test Forge is a comprehensive test generation system that combines coverage-guided generation, property-based testing with rapid, advanced fuzzing, and iterative refinement to create thorough test suites for Go code.

## Key Features

### 1. Coverage-Guided Generation
- Analyzes existing test coverage to identify gaps
- Generates targeted tests for uncovered branches
- Tracks coverage improvements throughout the process
- Aims for 100% coverage of critical paths

### 2. Property-Based Testing with Rapid
```go
// Instead of example-based tests:
func TestAdd(t *testing.T) {
    assert.Equal(t, 4, Add(2, 2))
}

// Test Forge generates property-based tests:
func TestAddProperties(t *testing.T) {
    rapid.Check(t, func(rt *rapid.T) {
        a := rapid.Int().Draw(rt, "a")
        b := rapid.Int().Draw(rt, "b")
        
        // Commutative property
        assert.Equal(rt, Add(a, b), Add(b, a))
        
        // Identity property
        assert.Equal(rt, a, Add(a, 0))
        
        // Inverse property
        assert.Equal(rt, 0, Add(a, -a))
    })
}
```

### 3. Advanced Fuzzing Beyond Encode/Decode
```go
// Not just simple fuzzing:
func FuzzMarshal(f *testing.F) {
    f.Fuzz(func(t *testing.T, data []byte) {
        var v Value
        v.Unmarshal(data)
        v.Marshal()
    })
}

// But sophisticated state machine fuzzing:
func FuzzStateMachine(f *testing.F) {
    f.Fuzz(func(t *testing.T, ops []byte) {
        sm := NewStateMachine()
        for _, op := range ops {
            sm.Execute(Operation(op))
            assert.True(t, sm.CheckInvariants())
        }
    })
}
```

### 4. Test Harness Design
```go
// Test Forge creates reusable test infrastructure:
type PaymentTestHarness struct {
    validPayments   []Payment
    invalidPayments []Payment
    mockBank        *MockBankService
    
    // Generators for property tests
    genPayment      func(*rapid.T) Payment
    genAccount      func(*rapid.T) Account
    
    // Invariant checkers
    checkBalances   func() error
    checkAuditLog   func() error
}

// Making tests readable and maintainable
func (h *PaymentTestHarness) ProcessAndVerify(t *testing.T, p Payment) {
    err := h.Process(p)
    assert.NoError(t, h.checkBalances())
    assert.NoError(t, h.checkAuditLog())
}
```

## Usage Examples

### Basic Usage
```bash
# Test a specific function
/test-forge ProcessPayment

# Test an entire package
/test-forge ./payments

# Test with specific focus
/test-forge StateMachine "focus on concurrent operations"
```

### Example Outputs

#### 1. Property Test Generation
When you run `/test-forge BinaryTree.Insert`, Test Forge might generate:

```go
func TestBinaryTreeProperties(t *testing.T) {
    rapid.Check(t, func(rt *rapid.T) {
        tree := NewBinaryTree()
        values := rapid.SliceOf(rapid.Int()).Draw(rt, "values")
        
        // Insert all values
        for _, v := range values {
            tree.Insert(v)
        }
        
        // Property 1: BST invariant
        assert.True(rt, tree.IsBST())
        
        // Property 2: Size matches insertions
        assert.Equal(rt, len(unique(values)), tree.Size())
        
        // Property 3: All values are findable
        for _, v := range values {
            assert.True(rt, tree.Contains(v))
        }
        
        // Property 4: In-order traversal is sorted
        inOrder := tree.InOrderTraversal()
        assert.True(rt, isSorted(inOrder))
    })
}
```

#### 2. State Machine Fuzzing
For a connection pool, Test Forge might generate:

```go
func FuzzConnectionPool(f *testing.F) {
    // Seed with interesting operation sequences
    f.Add([]byte{OP_ACQUIRE, OP_RELEASE})
    f.Add([]byte{OP_ACQUIRE, OP_ACQUIRE, OP_RELEASE, OP_RELEASE})
    f.Add([]byte{OP_ACQUIRE, OP_CLOSE, OP_RELEASE})
    
    f.Fuzz(func(t *testing.T, ops []byte) {
        pool := NewConnectionPool(10)
        activeConns := make(map[int]bool)
        
        for i, op := range ops {
            switch Operation(op % 4) {
            case OP_ACQUIRE:
                conn := pool.Acquire()
                if conn != nil {
                    activeConns[conn.ID] = true
                }
            case OP_RELEASE:
                if len(activeConns) > 0 {
                    for id := range activeConns {
                        pool.Release(id)
                        delete(activeConns, id)
                        break
                    }
                }
            case OP_CLOSE:
                pool.Close()
            case OP_RESET:
                pool.Reset()
            }
            
            // Invariants that must hold after each operation
            assert.True(t, pool.ActiveCount() >= 0)
            assert.True(t, pool.ActiveCount() <= pool.MaxSize())
            assert.Equal(t, len(activeConns), pool.ActiveCount())
        }
    })
}
```

## Parallel Agent Strategy

Test Forge launches multiple specialized agents in parallel:

1. **Property Agent**: Identifies mathematical properties and invariants
2. **Edge Case Agent**: Finds boundary conditions and corner cases
3. **State Machine Agent**: Tests state transitions and consistency
4. **Negative Test Agent**: Tests error paths and invalid inputs
5. **Concurrency Agent**: Tests race conditions and parallel operations

Each agent generates its own test file, which are then combined and refined.

## Coverage Improvement Process

1. **Initial Analysis**:
   ```bash
   go test -cover ./...
   # Package: payments  Coverage: 67.3%
   ```

2. **Gap Identification**:
   - Uncovered error paths in payment validation
   - Missing tests for concurrent transfers
   - No tests for rollback scenarios

3. **Targeted Generation**:
   - Generate tests specifically for identified gaps
   - Focus on high-risk uncovered code

4. **Final Coverage**:
   ```bash
   go test -cover ./...
   # Package: payments  Coverage: 94.7%
   ```

## Best Practices

### When to Use Test Forge

✅ **Perfect for:**
- Complex business logic with many edge cases
- State machines and protocol implementations
- Code with mathematical properties to verify
- Systems requiring high reliability
- Legacy code lacking comprehensive tests

❌ **Less suitable for:**
- Simple CRUD operations
- Pure UI components
- Code heavily dependent on external services
- One-off scripts

### Tips for Best Results

1. **Provide Context**: Tell Test Forge about domain-specific invariants
   ```
   /test-forge OrderProcessor "orders must maintain ACID properties"
   ```

2. **Specify Focus Areas**: Guide the generation toward critical aspects
   ```
   /test-forge CacheImpl "focus on TTL expiration and concurrent access"
   ```

3. **Review and Refine**: Test Forge performs multiple passes, but human review improves quality

4. **Maintain Test Harnesses**: The generated harnesses are valuable - keep them updated

## Integration with CI/CD

The generated tests are designed to work with standard Go tooling:

```yaml
# .github/workflows/test.yml
- name: Run Tests with Coverage
  run: |
    go test -race -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

- name: Run Fuzz Tests
  run: |
    go test -fuzz=Fuzz -fuzztime=30s ./...

- name: Run Property Tests
  run: |
    go test -run=Property -count=100 ./...
```

## Comparison with Manual Testing

| Aspect | Manual Testing | Test Forge |
|--------|---------------|------------|
| Coverage Analysis | Manual inspection | Automated gap detection |
| Property Testing | Rarely used | Default approach |
| Test Harness | Ad-hoc creation | Systematic design |
| Fuzzing | Basic if any | Advanced state fuzzing |
| Edge Cases | Often missed | Systematically found |
| Refinement | One-time effort | Multiple passes |
| Time Investment | Hours to days | Minutes to hours |

## Limitations

- Requires understanding of code semantics (not just syntax)
- May generate redundant tests initially (refined in later passes)
- Complex mocking scenarios may need manual intervention
- Performance tests require baseline metrics

## Future Enhancements

- Mutation testing to verify test effectiveness
- Automatic test minimization
- Contract testing for interfaces
- Chaos testing for distributed systems