# Coverage Pitfalls: Why Line Coverage Misleads

Line coverage is the most common test-quality metric. It is also the most misleading. This reference explains the hierarchy of coverage criteria, the gap between coverage and correctness, and how `test-refine` uses code-guided coverage *without* over-relying on line %.

## The Coverage Hierarchy

| Criterion | What it measures | Limitation |
|---|---|---|
| **Statement (line)** | Each line of code executed | Doesn't verify *what* the code did — assertions can be absent |
| **Branch** | Each `true`/`false` outcome of every conditional taken | Doesn't account for *which* sub-condition drove the outcome |
| **Condition** | Each atomic boolean condition's true/false outcomes | Short-circuit evaluation can leave conditions unevaluated |
| **MC/DC** (Modified Condition/Decision Coverage) | Each condition independently shown to affect the outcome | Combinatorial; required by avionics (DO-178C Level A) and ISO 26262 ASIL D |
| **Mutation** | Each mutation killed by at least one test | Catches assertion gaps statement coverage cannot |

Each criterion strictly dominates the previous. 100% MC/DC implies 100% branch implies 100% statement. But none of them imply assertion strength — that's where mutation testing enters.

## The Assertion-Free Trap

The classic failure mode:

```go
func Calculate(amount, rate int64) int64 {
    return (amount * rate) / 10000
}

// 100% line coverage. Zero assertions.
func TestCalculate(t *testing.T) {
    Calculate(10_000, 100)              // executes every line
}
```

The function is "covered" by every coverage tool. The test catches no bugs. Line coverage alone gives you a number; mutation testing tells you the number is a lie.

This is what motivated [PITest](https://pitest.org/) and [gremlins](https://github.com/go-gremlins/gremlins) — directly probe whether tests catch behavior changes, not whether they execute lines.

## Branch Coverage and Short-Circuit Evaluation

Branch coverage looks better than statement coverage but has its own gap:

```go
if a || b {
    process()
}
```

Branch coverage is satisfied with two tests:

| Test | a | b | a || b |
|---|---|---|---|
| 1 | true | * | true |
| 2 | false | false | false |

Branch coverage is 100%. But test 1 never evaluates `b` (short-circuit), so a bug in computing `b` is invisible. This is why MC/DC requires showing each condition independently affects the outcome — but MC/DC requires `n+1` tests for `n` conditions, often combinatorially explosive.

For most Go code, branch coverage + mutation testing is the right tradeoff: branch coverage cheaply identifies untested paths, mutation testing identifies untested *behavior* on tested paths.

## How `test-refine` Uses Coverage

The skill consumes Go's standard coverage profile:

```bash
go test -cover -covermode=atomic -coverprofile=cov.out ./...
go tool cover -func=cov.out          # per-function summary
```

Two signals from this:

1. **Per-function coverage %** — feeds the `branch_gap` component of the composite priority score. Functions with low coverage in critical paths score higher.
2. **Uncovered blocks** — locations where no test exercises the code. Used to propose "Add test for X" findings.

What `test-refine` *deliberately doesn't do*:

- **Doesn't gate on a coverage threshold.** A 95% line-coverage target produces tests like the assertion-free example above. Threshold gates encourage gaming.
- **Doesn't equate coverage with quality.** The report shows coverage as one signal alongside smell counts, mutation efficacy, and domain-check findings.

## Practical Hierarchy for Go Projects

For most Go code:

1. **Statement coverage** is a reasonable baseline (60–80%) — but the skill won't push you past it.
2. **Branch coverage** is what the skill prioritizes for finding untested paths.
3. **Mutation testing** (when `--use-mutations` is set) tells you whether the tested paths are *verified*.
4. **MC/DC** is overkill for non-safety-critical code. Use it only when contractually required.

For safety- or money-critical code (consensus, channel state, payment, signing): aim for high mutation efficacy (≥90%) over high line coverage. A test suite at 70% line coverage and 95% mutation efficacy is stronger than 100% line coverage at 60% mutation efficacy.

## Coverage Pitfalls in Distributed/Networking Code

Standard coverage tools have additional blind spots for systems code:

- **Concurrent code paths.** A goroutine may run during a test without the test exercising the race the goroutine could lose. Coverage marks the goroutine's lines as executed; only `-race` and stress runs catch real bugs.
- **Failure paths.** Most error returns are reachable only when the underlying `os` / `net` layer fails. Coverage shows them as uncovered — but adding a fault-injecting wrapper is the right fix, not "ignore as untestable".
- **Non-determinism.** Coverage of `time.Now()`-dependent code varies between runs. Tests must inject a clock to be reliably covered.

These are why `test-refine` runs domain-specific checks ([`domain-checks.md`](domain-checks.md)) on top of statement coverage.

## Summary

- Line coverage is the floor, not the ceiling.
- Branch coverage finds untested paths.
- Mutation testing finds untested *behavior*.
- For critical Go code, optimize for mutation efficacy, not coverage %.
- `test-refine` uses coverage as a gap-finding signal, not a quality score.

## Sources

- [Why Code Coverage Numbers Are Lying to You — Design News](https://www.designnews.com/embedded-systems/why-code-coverage-numbers-are-lying-to-you-what-to-do-about-it)
- [Code Coverage vs Mutation Testing — Optivem Journal](https://journal.optivem.com/p/code-coverage-vs-mutation-testing)
- [What Code Coverage Metrics Really Tell You — Qt](https://www.qt.io/quality-assurance/blog/why-code-coverage-metrics-can-be-misleading-and-how-coco-code-coverage-tool-makes-them-meaningful)
- [Modified Condition/Decision Coverage (MC/DC) — Qt](https://www.qt.io/quality-assurance/coco/feature-modified-condition-decision-coverage-mcdc)
