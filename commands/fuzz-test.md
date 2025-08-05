---
description: "Generate and run Go fuzz tests for security vulnerabilities"
argument-hint: "<package_or_function> [--duration=10m] [--security-focus]"
allowed-tools:
  - Task
  - Bash
  - Read
  - Write
  - Edit
  - MultiEdit
  - Grep
  - Glob
---

# Go Fuzzing Campaign

Create and execute Go native fuzz tests for: $ARGUMENTS

## Fuzzing Strategy

1. **Target Identification**
   - Analyze functions that parse untrusted input
   - Identify security-critical code paths
   - Focus on: parsers, decoders, validators, network handlers

2. **Fuzz Test Generation**
   Generate comprehensive fuzz tests using Go's native fuzzing:
   ```go
   func FuzzTargetFunction(f *testing.F) {
       // Add seed corpus for better coverage
       f.Add(seedInput1)
       f.Add(edgeCaseInput)
       
       f.Fuzz(func(t *testing.T, input []byte) {
           // Security assertions
           defer func() {
               if r := recover(); r != nil {
                   t.Errorf("Panic detected: %v", r)
               }
           }()
           
           // Test target function
           result := TargetFunction(input)
           
           // Security invariants
           validateNoMemoryLeak(t)
           validateNoPanic(t)
           validateNoInfiniteLoop(t)
       })
   }
   ```

3. **Security-Focused Test Patterns**
   - **Input Validation**: Test with malformed/oversized inputs
   - **Resource Exhaustion**: Check for unbounded allocations
   - **State Corruption**: Verify state consistency after errors
   - **Injection Attacks**: Test special characters and escape sequences
   - **Integer Overflows**: Use boundary values and large numbers

4. **Execution & Analysis**
   ```bash
   # Run fuzzing with coverage
   go test -fuzz=FuzzTargetFunction -fuzztime=10m
   
   # Run all fuzz tests in package
   go test -fuzz=. -fuzztime=30m ./...
   
   # Check corpus coverage
   go test -cover -run=FuzzTargetFunction
   ```

5. **Corpus Management**
   - Save interesting inputs that increase coverage
   - Maintain regression corpus from crashes
   - Share corpus across team for continuous improvement

Use the security-auditor agent to:
- Identify high-risk parsing functions
- Generate comprehensive fuzz harnesses
- Analyze crashes for security impact
- Create proof-of-concept exploits from crashes
- Recommend defensive coding patterns