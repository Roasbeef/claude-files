# Mutation Testing Best Practices

This guide provides advanced patterns and best practices for effective mutation testing in Go projects.

## Table of Contents

1. [Scoping Mutations](#scoping-mutations)
2. [Performance Optimization](#performance-optimization)
3. [Test Generation Strategies](#test-generation-strategies)
4. [Handling Equivalent Mutants](#handling-equivalent-mutants)
5. [CI/CD Integration](#cicd-integration)
6. [Interpreting Results](#interpreting-results)
7. [Mission-Critical Code Patterns](#mission-critical-code-patterns)
8. [Common Pitfalls](#common-pitfalls)

---

## Scoping Mutations

### Start Narrow

Don't mutation test your entire codebase at once. Focus on high-value areas first.

**Recommended prioritization**:

1. **Mission-critical code**: Financial calculations, security checks, consensus logic
2. **Complex business logic**: State machines, workflow engines, algorithms
3. **Bug-prone areas**: Code with history of defects
4. **Recently changed code**: New features and bug fixes
5. **Public API surfaces**: External contracts that can't change

**Skip mutation testing for**:
- Generated code (protobuf, mocks)
- Simple getters/setters with no logic
- Obvious delegation code
- Test helpers (testing test code is usually low value)

### File and Package Selection

```bash
# Mutation test only changed files (for PR reviews)
git diff --name-only main | grep '\.go$' | grep -v '_test\.go$'

# Mutation test specific critical packages
./scripts/generate_mutations.go --package ./internal/consensus
./scripts/generate_mutations.go --package ./internal/wallet

# Mutation test files matching pattern
./scripts/generate_mutations.go --pattern '**/validation*.go'
```

### Line and Function Filtering

For large files, focus on specific functions:

```bash
# Mutation test specific function
./scripts/generate_mutations.go --file wallet.go --function CalculateFee

# Mutation test specific line range
./scripts/generate_mutations.go --file wallet.go --lines 100-200
```

---

## Performance Optimization

### Parallel Execution

Run independent mutations concurrently:

```bash
# Generate mutations
./scripts/generate_mutations.go --file wallet.go --output mutations.json

# Run mutations in parallel (example using xargs)
jq -r 'to_entries | .[] | "\(.key)"' mutations.json | \
  xargs -P 8 -I {} ./scripts/run_mutation_test.go --mutation mutations.json:{} --package ./internal/wallet
```

**Guideline**: Use number of CPU cores minus 1 for parallelism to avoid overwhelming the system.

### Test Selection

Only run tests that cover the mutated code:

```bash
# Use go test's coverage to identify relevant tests
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep wallet.go

# Run only tests that cover wallet.go
go test -run TestWallet ./...
```

**Strategy**: Maintain a mapping of source files to test files. Use `go list` and build tags to determine test dependencies.

### Incremental Mutation Testing

For CI/CD, only test changed code:

```bash
# Get changed files from git
CHANGED_FILES=$(git diff --name-only origin/main...HEAD | grep '\.go$' | grep -v '_test\.go$')

# Generate mutations only for changed files
for file in $CHANGED_FILES; do
  ./scripts/generate_mutations.go --file "$file" --output "mutations_$(basename "$file").json"
done
```

### Caching

Cache mutation results to avoid re-running unchanged code:

```bash
# Cache key: hash of (source file + test files + mutation descriptor)
# If all three are unchanged, reuse previous result

# Example cache structure:
# .mutation_cache/
#   wallet.go-abc123-M1.json  (cached result)
#   wallet.go-abc123-M2.json
```

---

## Test Generation Strategies

### Analyze Why Mutants Survive

For each surviving mutant, categorize the reason:

**Category 1: Missing Test**
```go
// Surviving mutant
if balance > threshold {  // Original
if balance >= threshold { // Mutant (survived)

// Reason: No test for exact threshold value
// Solution: Add boundary test
func TestBalance_ExactThreshold(t *testing.T) {
    result := CheckBalance(1000) // exact threshold
    assert.Equal(t, expected, result)
}
```

**Category 2: Weak Assertion**
```go
// Surviving mutant
result := Calculate(a + b)  // Original
result := Calculate(a - b)  // Mutant (survived)

// Existing test
func TestCalculate(t *testing.T) {
    Calculate(10, 5) // Runs but doesn't assert result!
}

// Solution: Add assertion
func TestCalculate(t *testing.T) {
    result := Calculate(10, 5)
    assert.Equal(t, 15, result) // Now catches mutant
}
```

**Category 3: Wrong Test Data**
```go
// Surviving mutant
if amount < 0 {   // Original
if amount <= 0 {  // Mutant (survived)

// Existing test
func TestNegativeAmount(t *testing.T) {
    result := Validate(-10) // Never tests zero
    assert.Error(t, result)
}

// Solution: Add edge case
func TestZeroAmount(t *testing.T) {
    result := Validate(0)
    assert.NoError(t, result) // Or Error, depending on requirements
}
```

### Systematic Test Generation

For high-priority survivors, generate tests systematically:

```go
// For boundary mutant: a > b → a >= b
// Generate three tests:
1. a < b (clearly one side)
2. a == b (exact boundary)
3. a > b (clearly other side)

// For arithmetic mutant: a + b → a - b
// Generate tests with:
1. Positive operands
2. Negative operands
3. Mixed signs
4. Zero values
5. Expected result assertion

// For logical mutant: a && b → a || b
// Generate truth table tests:
1. a=true, b=true
2. a=true, b=false
3. a=false, b=true
4. a=false, b=false
```

### Property-Based Tests for Mutations

Use property-based testing to kill mutants:

```go
// Original code
func Add(a, b int) int {
    return a + b
}

// Mutant: return a - b

// Property-based test (using rapid)
func TestAdd_Commutative(t *testing.T) {
    rapid.Check(t, func(rt *rapid.T) {
        a := rapid.Int().Draw(rt, "a")
        b := rapid.Int().Draw(rt, "b")

        // Commutative property
        assert.Equal(t, Add(a, b), Add(b, a))

        // This kills the mutant because:
        // a + b == b + a (passes)
        // a - b != b - a (fails, mutant killed)
    })
}
```

---

## Handling Equivalent Mutants

Equivalent mutants don't change observable behavior and can't be killed by tests.

### Common Equivalent Patterns

**1. Unused Variable Mutation**
```go
// Equivalent mutant
temp := a + b  // Original: a + b
temp := a - b  // Mutant: a - b
result = c     // Reassignment makes mutation irrelevant
```

**2. Dead Code**
```go
if DEBUG {  // DEBUG is always false
    log.Println("debug")  // Any mutation here is equivalent
}
```

**3. Associative Operations**
```go
// May be equivalent depending on values
if a || b || c {  // vs  if a || c || b
```

### Detecting Equivalents

**Strategy 1: Static Analysis**
```go
// Check if mutated variable is:
// 1. Never read after mutation point
// 2. Immediately overwritten
// 3. In unreachable code
```

**Strategy 2: Multiple Test Runs**
```go
// Run tests multiple times with different random seeds
// If mutant survives consistently, likely equivalent
// If occasionally killed, timing/flaky test issue
```

**Strategy 3: Manual Review**
```go
// Mark mutants as equivalent after review
// Store in .equivalent_mutants.json
// Exclude from future runs
```

### Managing Equivalent Mutants

Create an equivalents file:

```json
{
  "wallet.go": {
    "line_42_arithmetic": {
      "reason": "Variable immediately reassigned",
      "reviewed_by": "claude",
      "reviewed_at": "2025-10-29"
    }
  }
}
```

Exclude from mutation score calculation:
```
Mutation Score = killed / (total - timeouts - equivalents)
```

---

## CI/CD Integration

### Pull Request Workflow

```yaml
# .github/workflows/mutation-test.yml
name: Mutation Testing

on:
  pull_request:
    paths:
      - '**.go'
      - '!**_test.go'

jobs:
  mutation-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
        with:
          fetch-depth: 0  # Need full history for diff

      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run mutation testing on changed files
        run: |
          # Get changed Go files
          CHANGED_FILES=$(git diff --name-only origin/${{ github.base_ref }}...HEAD | \
            grep '\.go$' | grep -v '_test\.go$')

          # Run mutations
          for file in $CHANGED_FILES; do
            ~/.claude/skills/mutation-testing/scripts/generate_mutations.go \
              --file "$file" --output "mutations_$file.json"

            ~/.claude/skills/mutation-testing/scripts/run_mutation_test.go \
              --mutation "mutations_$file.json" --package ./...
          done

          # Parse results
          ~/.claude/skills/mutation-testing/scripts/parse_results.go \
            --results results/*.json --output mutation_report.json

      - name: Check mutation score threshold
        run: |
          SCORE=$(jq -r '.mutation_score' mutation_report.json)
          THRESHOLD=80

          if (( $(echo "$SCORE < $THRESHOLD" | bc -l) )); then
            echo "Mutation score $SCORE% is below threshold $THRESHOLD%"
            exit 1
          fi

      - name: Comment on PR
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const report = JSON.parse(fs.readFileSync('mutation_report.json'));

            const body = `## Mutation Testing Report

            **Mutation Score**: ${report.mutation_score}%
            **Mutants Killed**: ${report.killed}/${report.total}

            ${report.survivors.length > 0 ? '### Surviving Mutants\n' +
              report.survivors.map(s => `- ${s.file}:${s.line} - ${s.description}`).join('\n')
              : 'All mutants killed! 🎉'}
            `;

            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: body
            });
```

### Quality Gates

Define mutation score thresholds based on code criticality:

```yaml
# .mutation_config.yml
quality_gates:
  critical:
    paths: ["internal/consensus", "internal/wallet"]
    min_score: 90

  important:
    paths: ["internal/validation", "internal/protocol"]
    min_score: 80

  standard:
    paths: ["**"]
    min_score: 70
```

---

## Interpreting Results

### Mutation Score Context

Mutation score alone doesn't tell the full story. Consider:

**High score (>90%) with few tests**: Tests might be overfitted to current implementation. Add property-based tests.

**Medium score (70-80%) with many tests**: Tests likely cover main paths but miss edges. Focus on boundary tests.

**Low score (<70%)**: Significant gaps. Tests may only verify happy paths without assertions.

**100% score**: Either perfect tests or too few mutations. Verify mutation generation is thorough.

### Mutation Type Analysis

Track which mutation types survive most:

```
Boundary mutations: 20% survival → Need more edge case tests
Arithmetic mutations: 5% survival → Good coverage of calculations
Logical mutations: 40% survival → Weak conditional testing
Statement removal: 15% survival → Some dead code or weak tests
```

**Action items**:
- High boundary survival → Add boundary value tests
- High logical survival → Add truth table tests
- High arithmetic survival → Add calculation verification tests
- High statement removal survival → Remove dead code or add effect tests

### Trend Analysis

Track mutation scores over time:

```
PR #123: 75% → 82% (+7%) ✅ Improvement
PR #124: 82% → 79% (-3%) ⚠️ Regression
PR #125: 79% → 91% (+12%) ✅ Strong improvement
```

**Set up alerts**:
- Regression > 5%: Block merge
- Regression 2-5%: Warning
- Regression < 2%: Accept (noise)

---

## Mission-Critical Code Patterns

For Bitcoin/Lightning and other mission-critical systems:

### Financial Calculation Testing

```go
// Code
func CalculateFee(amount, rate int64) int64 {
    return (amount * rate) / 10000
}

// Mutation testing must catch:
// 1. Arithmetic errors: * → /, + → -, etc.
// 2. Overflow/underflow: Use extreme values
// 3. Precision loss: Division order changes
// 4. Off-by-one in denominator: 10000 → 10001

// Required tests:
func TestCalculateFee_Precision(t *testing.T) {
    // Test precision at boundaries
    assert.Equal(t, 1, CalculateFee(10000, 1))
    assert.Equal(t, 0, CalculateFee(9999, 1))
}

func TestCalculateFee_Overflow(t *testing.T) {
    // Test large values
    large := int64(1<<62)
    result := CalculateFee(large, 1)
    assert.True(t, result > 0, "Overflow should not occur")
}
```

### Security Check Testing

```go
// Code
func Authorize(user User, resource Resource) bool {
    return user.IsAdmin() && resource.AllowsUser(user)
}

// Critical mutations:
// && → || (SECURITY BUG!)
// Tests MUST catch this

// Required test:
func TestAuthorize_BothConditionsRequired(t *testing.T) {
    user := &User{admin: false}
    resource := &Resource{allowed: []User{user}}

    // Should fail because not admin
    assert.False(t, Authorize(user, resource))

    // This test kills the && → || mutant
}
```

### State Machine Testing

```go
// Code
type State int
const (
    Init State = iota
    Running
    Stopped
)

func Transition(current State, event Event) (State, error) {
    switch current {
    case Init:
        if event == Start {
            return Running, nil
        }
    case Running:
        if event == Stop {
            return Stopped, nil
        }
    }
    return current, ErrInvalidTransition
}

// Mutations to catch:
// 1. Wrong next state
// 2. Missing error return
// 3. Wrong event check

// Required: Test all valid and invalid transitions
```

---

## Common Pitfalls

### Pitfall 1: Chasing 100% Score

**Problem**: Spending excessive time on equivalent mutants or trivial code.

**Solution**: Accept 85-95% as excellent for most code. Focus on high-impact survivors.

### Pitfall 2: Ignoring Context

**Problem**: Treating all code equally regardless of risk.

**Solution**: Higher standards for critical code, lower for trivial code.

### Pitfall 3: Overfitting Tests

**Problem**: Writing tests that only pass because they match current implementation details.

**Solution**: Use property-based tests and specify behavior, not implementation.

### Pitfall 4: Slow Execution

**Problem**: Running all tests for every mutation.

**Solution**: Use coverage-guided test selection and parallel execution.

### Pitfall 5: False Confidence

**Problem**: High mutation score with weak tests that happen to kill mutants accidentally.

**Solution**: Review surviving mutants and ensure killed mutants are killed for the right reasons.

### Pitfall 6: Mutating Test Code

**Problem**: Generating mutations in test files themselves.

**Solution**: Only mutate production code, never test code.

### Pitfall 7: Ignoring Timeouts

**Problem**: Treating timeouts as killed mutants.

**Solution**: Investigate timeouts—they often reveal performance bugs or infinite loops.

---

## Advanced Techniques

### Differential Mutation Testing

Compare mutation scores between versions:

```bash
# Baseline (main branch)
git checkout main
run_mutations.sh > baseline_mutations.json

# Current (feature branch)
git checkout feature
run_mutations.sh > current_mutations.json

# Compare
diff_mutations.sh baseline_mutations.json current_mutations.json
```

### Mutation Coverage

Track which parts of code have been mutated:

```
File: wallet.go
Lines with mutations: 45/100 (45%)
Functions with mutations: 8/12 (67%)

Unmutated areas:
- Lines 1-20: Type definitions (intentionally skipped)
- Lines 80-95: Simple getters (low value)
- Lines 120-130: TODO: Generate mutations
```

### Cost-Benefit Analysis

Track time spent vs. bugs found:

```
Mutation testing time: 30 minutes per PR
Bugs found by improved tests: 3 critical bugs/month
Time saved by preventing bugs: ~10 hours/month

ROI: (10 hours saved - 2 hours spent) = +8 hours/month
```

---

## Summary Checklist

Before running mutation testing:
- [ ] Identify high-priority code to mutate
- [ ] Ensure baseline tests pass
- [ ] Set appropriate mutation score targets
- [ ] Configure parallel execution
- [ ] Set up caching for large codebases

During mutation testing:
- [ ] Monitor for timeouts and investigate
- [ ] Categorize survivors by reason
- [ ] Generate targeted tests for high-priority survivors
- [ ] Mark equivalent mutants
- [ ] Track mutation types that survive most

After mutation testing:
- [ ] Review mutation score in context
- [ ] Generate actionable test improvements
- [ ] Update equivalent mutants list
- [ ] Document findings and patterns
- [ ] Integrate into CI/CD if valuable

For mission-critical code:
- [ ] Achieve 90%+ mutation score
- [ ] Manually review all survivors
- [ ] Test boundary conditions exhaustively
- [ ] Verify security-critical mutations are killed
- [ ] Use property-based tests for key invariants
