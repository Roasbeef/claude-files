# Mutation Operators Reference

This document catalogs all mutation operators used for Go code mutation testing. Each operator is defined with examples, AST node types, and rationale.

## Operator Categories

1. [Arithmetic Operators](#arithmetic-operators)
2. [Relational Operators](#relational-operators)
3. [Logical Operators](#logical-operators)
4. [Unary Operators](#unary-operators)
5. [Assignment Operators](#assignment-operators)
6. [Statement Mutations](#statement-mutations)
7. [Constant Mutations](#constant-mutations)
8. [Boundary Mutations](#boundary-mutations)
9. [Function Call Mutations](#function-call-mutations)
10. [Control Flow Mutations](#control-flow-mutations)

---

## Arithmetic Operators

**AST Node**: `ast.BinaryExpr` with arithmetic `token.Token`

### Addition ↔ Subtraction

```go
// Original
result := a + b

// Mutations
result := a - b  // Addition to subtraction
result := a * b  // Addition to multiplication
result := a / b  // Addition to division
```

**Rationale**: Tests should verify correct arithmetic operations, especially in financial calculations.

### Multiplication ↔ Division

```go
// Original
fee := amount * rate

// Mutations
fee := amount / rate  // Multiplication to division
fee := amount + rate  // Multiplication to addition
fee := amount - rate  // Multiplication to subtraction
```

**Rationale**: Critical for fee calculations, scaling operations, and mathematical computations.

### Modulo Operator

```go
// Original
index := value % size

// Mutations
index := value / size  // Modulo to division
index := value * size  // Modulo to multiplication
```

**Rationale**: Tests boundary conditions in array indexing and cyclic operations.

---

## Relational Operators

**AST Node**: `ast.BinaryExpr` with comparison `token.Token`

### Greater Than / Less Than

```go
// Original
if balance > threshold {

// Mutations
if balance >= threshold {  // Boundary mutation
if balance < threshold {   // Relational inversion
if balance <= threshold {  // Inverted boundary
if balance == threshold {  // Exact match
if balance != threshold {  // Inequality
```

**Rationale**: Exposes boundary condition bugs and off-by-one errors.

### Equality Operators

```go
// Original
if status == Active {

// Mutations
if status != Active {  // Equality to inequality
if status > Active {   // If comparable type
if status < Active {   // If comparable type
```

**Rationale**: Tests that code correctly handles equality and inequality.

---

## Logical Operators

**AST Node**: `ast.BinaryExpr` with logical `token.Token`

### AND ↔ OR

```go
// Original
if authenticated && authorized {

// Mutations
if authenticated || authorized {  // AND to OR (security critical!)
if authenticated {                // Remove second condition
if authorized {                   // Remove first condition
```

**Rationale**: Critical for security checks, access control, and complex conditionals.

### Negation

```go
// Original
if !isEmpty {

// Mutations
if isEmpty {  // Remove negation
```

**Rationale**: Tests that boolean logic is correctly implemented.

---

## Unary Operators

**AST Node**: `ast.UnaryExpr`

### Increment / Decrement

```go
// Original
counter++

// Mutations
counter--      // Increment to decrement
counter += 2   // Wrong increment amount
// (no mutation) // Statement removal
```

**Rationale**: Tests loop termination conditions and counter logic.

### Unary Plus / Minus

```go
// Original
value := -amount

// Mutations
value := +amount  // Negation removal
value := amount   // Unary operator removal
```

**Rationale**: Tests sign handling in mathematical operations.

---

## Assignment Operators

**AST Node**: `ast.AssignStmt`

### Compound Assignment

```go
// Original
balance += deposit

// Mutations
balance -= deposit  // += to -=
balance *= deposit  // += to *=
balance = deposit   // += to =
```

**Rationale**: Tests that state updates are correctly implemented.

### Assignment Order

```go
// Original
a, b = b, a  // Swap

// Mutations
a, b = a, b  // No swap
b, a = b, a  // Reverse order
```

**Rationale**: Tests tuple assignments and swap operations.

---

## Statement Mutations

**AST Node**: Various statement types

### Return Statement Removal

```go
// Original
func Validate() error {
    if invalid {
        return ErrInvalid
    }
    return nil
}

// Mutations
func Validate() error {
    if invalid {
        // (removed return)
    }
    return nil
}
```

**Rationale**: Tests that early returns are necessary and tested.

### Return Value Mutation

```go
// Original
return true

// Mutations
return false   // Boolean flip
return !value  // If returning variable
```

**Rationale**: Tests that callers check return values correctly.

### Defer Statement Removal

```go
// Original
defer mu.Unlock()

// Mutations
// (removed defer)
mu.Unlock()  // Immediate call instead of defer
```

**Rationale**: Tests that defer timing is important and tested.

---

## Constant Mutations

**AST Node**: `ast.BasicLit`, `ast.Ident` (for predeclared constants)

### Numeric Constants

```go
// Original
const MaxRetries = 3

// Mutations
const MaxRetries = 0
const MaxRetries = 1
const MaxRetries = 4  // Off-by-one
```

**Rationale**: Tests that specific constant values matter.

### Boolean Constants

```go
// Original
enabled := true

// Mutations
enabled := false  // Boolean flip
```

**Rationale**: Tests that boolean flags are checked.

### Nil Mutations

```go
// Original
if err != nil {

// Mutations
if err == nil {  // Nil check inversion
```

**Rationale**: Tests error handling correctness.

### String Constants

```go
// Original
const Prefix = "BTC"

// Mutations
const Prefix = ""     // Empty string
const Prefix = "XXX"  // Different value
```

**Rationale**: Tests that string values are validated.

---

## Boundary Mutations

**AST Node**: Varies based on context

### Array/Slice Boundaries

```go
// Original
for i := 0; i < len(items); i++ {

// Mutations
for i := 0; i <= len(items); i++ {  // Off-by-one (will panic)
for i := 1; i < len(items); i++ {   // Skip first element
for i := 0; i < len(items)-1; i++ { // Skip last element
```

**Rationale**: Tests boundary condition handling and off-by-one errors.

### Range Conditions

```go
// Original
if value >= min && value <= max {

// Mutations
if value > min && value <= max {   // Exclude lower bound
if value >= min && value < max {   // Exclude upper bound
if value > min && value < max {    // Exclude both bounds
```

**Rationale**: Tests inclusive vs exclusive ranges.

---

## Function Call Mutations

**AST Node**: `ast.CallExpr`

### Argument Swap

```go
// Original
Transfer(from, to, amount)

// Mutations
Transfer(to, from, amount)  // Swap first two
Transfer(from, amount, to)  // Swap last two
```

**Rationale**: Tests that argument order matters and is validated.

### Argument Value Mutation

```go
// Original
Retry(maxAttempts, timeout)

// Mutations
Retry(0, timeout)           // Zero first arg
Retry(maxAttempts, 0)       // Zero second arg
Retry(maxAttempts+1, timeout) // Off-by-one
```

**Rationale**: Tests that specific argument values are meaningful.

### Function Call Removal

```go
// Original
err := Validate(input)
if err != nil {
    return err
}

// Mutations
// (removed Validate call)
var err error
if err != nil {
    return err
}
```

**Rationale**: Tests that function calls have side effects that matter.

---

## Control Flow Mutations

**AST Node**: `ast.IfStmt`, `ast.ForStmt`, `ast.SwitchStmt`

### If Condition Inversion

```go
// Original
if condition {
    doSomething()
}

// Mutations
if !condition {
    doSomething()
}
```

**Rationale**: Tests that condition polarity is correct.

### Else Branch Removal

```go
// Original
if valid {
    process()
} else {
    handleError()
}

// Mutations
if valid {
    process()
}
// (removed else)
```

**Rationale**: Tests that else branch is necessary.

### Break/Continue Removal

```go
// Original
for _, item := range items {
    if item == target {
        break
    }
}

// Mutations
for _, item := range items {
    if item == target {
        continue  // Break to continue
    }
}

// Or
for _, item := range items {
    if item == target {
        // (removed break)
    }
}
```

**Rationale**: Tests loop termination logic.

### Switch Case Fallthrough

```go
// Original
switch status {
case Active:
    start()
case Inactive:
    stop()
}

// Mutations
switch status {
case Active:
    start()
    fallthrough  // Add fallthrough
case Inactive:
    stop()
}
```

**Rationale**: Tests that cases are independent.

---

## Go-Specific Mutations

### Channel Operations

```go
// Original
ch <- value

// Mutations
<-ch           // Send to receive
value := <-ch  // Assign received value
```

**Rationale**: Tests channel direction and usage.

### Goroutine

```go
// Original
go process()

// Mutations
process()  // Remove go keyword (synchronous)
```

**Rationale**: Tests that concurrency is necessary.

### Select Statement

```go
// Original
select {
case <-done:
    return
case result := <-ch:
    process(result)
}

// Mutations
select {
case <-done:
    // (removed return)
case result := <-ch:
    process(result)
}
```

**Rationale**: Tests select case handling.

---

## Mutation Priority

Not all mutations are equally valuable. Prioritize mutations based on code importance:

### High Priority (Always Generate)
- Relational operators in conditionals (boundary bugs)
- Arithmetic in financial calculations (money bugs)
- Logical operators in security checks (auth bugs)
- Nil checks (panic prevention)
- Array bounds (panic prevention)

### Medium Priority (Generate for Non-Trivial Code)
- Function argument swaps
- Return value mutations
- Assignment operator changes
- Constant value changes

### Low Priority (Generate Selectively)
- String constant changes (unless validation critical)
- Unary operator changes (unless sign matters)
- Statement removal (unless side effects obvious)

---

## Equivalent Mutant Detection

Some mutations don't change observable behavior. Detect and skip these:

### Common Equivalent Mutants

```go
// Example 1: Unused intermediate value
result := a + b  // Mutation: a - b
result = c       // Immediate reassignment (equivalent)

// Example 2: Post-increment vs pre-increment
arr[i++]  // vs arr[++i] (not equivalent in Go, but similar)
// In standalone statement: i++ vs ++i (would be equivalent if Go supported both)

// Example 3: Associative operations
if a || b || c {  // Mutations may be equivalent due to short-circuit
```

**Strategy**: Track variable liveness and data flow to identify equivalent mutants automatically.

---

## Implementation Notes

### AST Node Type Mapping

```go
// Arithmetic: ast.BinaryExpr + token.ADD, token.SUB, etc.
// Relational: ast.BinaryExpr + token.LSS, token.GTR, etc.
// Logical: ast.BinaryExpr + token.LAND, token.LOR
// Unary: ast.UnaryExpr
// Assignment: ast.AssignStmt
// Calls: ast.CallExpr
// Return: ast.ReturnStmt
// If: ast.IfStmt
// For: ast.ForStmt, ast.RangeStmt
```

### Mutation Generation Strategy

1. Walk AST with `ast.Inspect()`
2. For each node, check if mutation operator applies
3. Generate all applicable mutations for that node
4. Store mutation descriptors with:
   - File path
   - Line and column number
   - Original code (for reporting)
   - Mutated code
   - Mutation type
   - Priority level

### Type Safety

Mutations must respect Go's type system:
- Don't mutate `+` to `/` if operands are strings
- Don't mutate `>` to `>=` if operands are non-comparable types
- Don't swap arguments if types differ

Use `go/types` package for type checking during mutation generation.

---

## Further Reading

- "PITest: Industrial-Grade Mutation Testing for Java" (Coles et al.)
- "Are Mutants a Valid Substitute for Real Faults in Software Testing?" (Just et al.)
- "An Analysis and Survey of the Development of Mutation Testing" (Jia & Harman)
