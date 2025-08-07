---
description: Generate comprehensive test suite with coverage guidance, property-based testing, and advanced fuzzing
argument-hint: <function/method/package to test> [optional: specific testing focus]
---

## Comprehensive Test Generation

I'll use the test-engineer agent to generate a sophisticated test suite for the specified code area. This will include property-based tests, advanced fuzzing, coverage-guided generation, and iterative refinement.

### Target for Testing
$ARGUMENTS

### Approach
The test-engineer agent will:
1. Analyze existing tests (if any) and run coverage analysis
2. Create or extend test harnesses for readability
3. Generate property-based tests using the rapid framework
4. Design advanced fuzz tests beyond simple encode/decode
5. Perform multiple refinement passes to improve tests
6. Track and maximize coverage throughout the process

Please use the Task tool to launch the test-engineer agent with the following prompt:

---
Generate or enhance tests for: $ARGUMENTS

## Context

First, determine whether this is:
1. **New Feature Testing**: Generate tests from scratch for new code
2. **Test Extension**: Add tests to improve existing coverage
3. **Test Enhancement**: Refactor existing tests to use better patterns

## Requirements

1. **Existing Test Analysis**:
   - Check for existing test files
   - Run `go test -cover` to identify current coverage
   - Understand existing test patterns and harnesses
   - Determine whether to extend or create new test files

2. **Coverage-Guided Approach**:
   - Target untested branches and error paths
   - Focus on high-risk uncovered code
   - Track coverage improvements throughout

3. **Test Infrastructure**:
   - Reuse existing test harnesses when available
   - Create new harnesses only when needed
   - Build generators for complex types
   - Design invariant checkers
   - Ensure tests are readable and maintainable

4. **Property-Based Testing (Primary Focus)**:
   - Use the rapid framework extensively
   - Identify and test invariants
   - Test reversible operations
   - Verify mathematical properties
   - Create custom generators for domain types

5. **Advanced Fuzzing**:
   - Go beyond simple marshal/unmarshal tests
   - Fuzz state machines with operation sequences
   - Test business logic with constraint violations
   - Design corpus with interesting edge cases

6. **Iterative Refinement**:
   After initial generation, perform passes for:
   - Simplification (remove redundancy)
   - Coverage gaps (target uncovered code)
   - Performance (add benchmarks where needed)
   - Documentation (explain complex tests)

7. **Test Organization**:
   - Separate files by test type (property_, fuzz_, unit_)
   - Use table-driven tests where appropriate
   - Include clear test names and descriptions
   - Add comments explaining what properties are tested

8. **Special Considerations**:
   - Test concurrent operations with race detector
   - Include benchmarks for performance-critical code
   - Create differential tests if multiple implementations exist
   - Use metamorphic testing for complex algorithms

## Expected Deliverables

1. **Test Files**:
   - `[name]_test.go` - Main test file with harness
   - `property_[name]_test.go` - Property-based tests
   - `fuzz_[name]_test.go` - Fuzz tests
   - `bench_[name]_test.go` - Benchmarks (if applicable)

2. **Coverage Report**:
   - Before/after coverage percentages
   - List of remaining uncovered code
   - Explanation of why some code might be untestable

3. **Test Documentation**:
   - Summary of testing approach
   - Properties and invariants tested
   - Known limitations or gaps
   - Recommendations for future testing

Focus on creating tests that not only validate correctness but also serve as documentation and catch regressions before they reach production.