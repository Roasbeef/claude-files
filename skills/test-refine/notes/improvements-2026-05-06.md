# `test-refine` skill — improvement notes

Generated while running the skill on a real-world 7-test-file, ~3k-line
diff in a private Go codebase. The skill produced 89 findings, of which
roughly 80 were false positives. This is the post-mortem.

## Symptoms observed

- **89 findings**, after a 5-minute Phase-A run on `--scope diff`.
- Of the top 30 (priority 0.55, all "HIGH"), **at least 26 were false
  positives** on inspection.
- The top of the table was dominated by S05 ("error return from SUT
  discarded with `_`"), every single one of which was actually a
  non-error multi-return value being discarded — e.g.
  `store, _ := newTestStore(t)` where the helper returns
  `(*Store, BatchedQuerier)`, no error.
- Two S01 ("no assertions") hits on top-level integration-test
  functions that immediately delegate to a runner helper containing
  all the assertions.
- One pair of S08 duplicate-test flags between two top-level
  integration-test entry points that intentionally share a runner
  (the standard Go pattern for parameterizing one scenario over a
  small enum).
- Phase-A produced **0 PBT candidates** despite the SUT containing a
  recursive walk over an arbitrary-depth DAG — exactly the canonical
  state-machine PBT shape.
- `Statement coverage: 0.0%` in the report header because the
  package set under `--scope diff` includes integration-test files,
  whose tests need a build tag and a live daemon — so
  `go test -cover` returns nothing usable. The report did not
  detect or warn about this.

## Specific root causes and proposed fixes

### 1. S05 detector is a syntactic-`_` heuristic, not an error-discard detector

**Code**: `scripts/detect-smells.go`, `findDiscardedErrors`.

**Bug**: it flagged any `AssignStmt` whose last LHS is `_`, regardless
of whether the RHS function returns an `error`. From the comment:

> heuristic on naming. Also catches `_ = SUT(...)` patterns where the
> only return is discarded and the call is plausibly an
> error-returning function.

The "plausibly" was doing too much work. On a Go test file with a lot
of helpers that return `(T, U)` or `(T, U, V)` (none of them errors),
nearly every line tripped this.

**Fixes** (in increasing rigor):

a. **Filter by callee name**. If the call is to a function whose
   name does not look like an error producer (`New*`, `Make*`,
   `Build*`, helper constructors), skip. This cut 80% of the
   noise on the run with one regex.

b. **Use `go/types` to actually resolve return types**. `go vet`
   already does this for `errcheck`. `detect-smells.go` runs in a
   throwaway `go.mod` with no module context, so it cannot type-check
   — it would need to be invoked with the package's real module so
   `packages.Load(...)` can produce typed ASTs. This is the right
   long-term fix.

c. **Cheap intermediate**: even without `go/types`, `findDiscardedErrors`
   has the AST. It can look at the *callee*'s `*ast.FuncDecl` if it's
   in the same package and inspect its return list for an `error`
   identifier. That covers the common case of test-helper functions
   defined in the same `_test.go` file — exactly the case that
   produced all the noise.

### 2. S01 detector did not traverse into helper calls

**Code**: `scripts/detect-smells.go`, "no assertions" check.

**Bug**: a top-level test like

```go
func TestSomeIntegrationScenario(t *testing.T) {
    runSomeScenario(t, modeA)
}
```

was flagged S01 because the body had no `require.X` calls. The
asserts lived inside the helper. This is a near-universal Go pattern
(table-driven dispatch, restart-variant dispatch, "small body
delegates to runner").

**Fix**: when the test body is a single call to another function in
the same package whose body *does* contain assertions, treat the
delegate as the assertion-bearing body. Equivalently: skip S01 when
the test body is a single delegation expression. The cost of false
negatives (a delegate that *also* has zero assertions) is low because
S01 will fire on the helper itself.

### 3. S01 detector did not look inside `t.Run` closures

**Code**: same.

**Bug**: a test consisting of N `t.Run("name", func(t *testing.T)
{ ...assertions... })` blocks was flagged S01 because the outer body
had no `require.X`.

**Fix**: when walking the test body for assertions, recurse into
`*ast.FuncLit` arguments to `t.Run`. Same idea as #2 but for
subtests.

### 4. S08 duplicate detector flagged intentional restart-variant pairs

**Code**: `scripts/detect-duplicates.go`.

**Bug**: two tests with structurally-identical bodies that differ
only in a single argument to a runner were flagged as duplicates.
This is the Go idiom for parameterizing one scenario over a tiny
enum when sharing a runner is more readable than a table.

**Fix**: when the two test bodies are both single calls to the same
runner, only differ in arguments, **and** the differing arg is a
package-internal constant of an `iota`-typed enum, demote from S08-H
to a M-severity *reshape* suggestion ("consider table-driving"),
not a duplicate-removal candidate. Better: flag as a candidate
*only if* the runner already takes a struct or slice argument.

### 5. PBT candidate detection misses state-machine shapes

**Code**: `domain-checks.go` (PBT detection logic).

**Bug**: zero PBT candidates were produced despite multiple obvious
shapes:
- A recursive walker that terminates on a chain of arbitrary depth
  (state-machine PBT for termination + frontier monotonicity).
- A retry-window dedup map (state-machine PBT for "no double
  submit").
- A multi-input-shape parser with multiple valid forms (parser /
  serializer roundtrip if there's an inverse).

The original detector only looked for serialize/deserialize pairs
(roundtrip PBT). It needs a second pass for state-machine PBT
candidates.

**Fix proposal**: in `domain-checks.go`, add detection for:
- A type with multiple `Handle*`/`On*` methods returning `error` and
  mutating internal state → suggest `rapid.StateMachine`.
- A function that takes a slice argument and recurses or loops over
  it with a sentinel-termination invariant → suggest a generator
  for the slice + an invariant assertion.

Until that lands, the report could include a "manual PBT review"
section pointing the user at `references/strategies.md` rather than
saying "_No PBT candidates detected._" which gives false confidence.

### 6. Scope `diff` did not handle build-tagged packages

**Code**: `scripts/triage.sh`, "Step 1: coverage" block.

**Bug**: when the diff includes files behind a build tag,
`go test -cover -coverprofile=...` ran without the matching tag, so
the files were excluded from the build, the profile was empty, and
the report header read `Statement coverage: 0.0%` — which was
misleading. Worse, the coverage gap also meant the `branch_gap`
factor in the priority formula was `0.0`, so integration-test
findings were uniformly priority 0.55 regardless of actual gap size.

**Fix proposals**, ordered by simplicity:

a. **Detect & warn**. If any `*_test.go` in scope has a `//go:build`
   constraint, print a warning and either (i) skip coverage for that
   package or (ii) re-run with `-tags <constraint>` parsed from the
   file. Cheap; addresses surprise.

b. **Coverage scope per file**. Group test files by their build
   constraints, run `go test -cover -tags=...` separately per group,
   merge profiles via `go tool cover` → unified report. Slightly
   more involved but produces an honest number.

c. **Skip coverage entirely on `--scope diff`** unless the user
   passes `--with-coverage`. The current default of "always run
   coverage" is expensive and produces a misleading number on
   integration-test-heavy diffs.

### 7. Priority formula collapsed when coverage was empty

**Code**: `scripts/score.go`.

**Bug**: when `branch_gap = 0.0` for every finding (because coverage
was empty — see #6), the priority formula reduced to `0.5 × risk +
0.3 × severity`. With every finding's risk falling into the same
bucket (e.g. all `0.7` for `internal/`-equivalent paths or all `1.0`
for `wallet|crypto|consensus`-matching paths), the report's "Top
Findings" table had 30 rows all at priority `0.55` — no
discrimination, no signal.

**Fix**: when coverage data is missing, *don't* default `branch_gap`
to `0.0`; either (a) use a uniform-prior `0.5` and warn, (b)
re-weight to drop the `branch_gap` term and renormalize, or (c)
sort by severity within each risk bucket as a fallback. Anything
that produces a meaningful ordering when one input signal is
unavailable.

### 8. Report did not surface the false-positive rate

**Code**: `scripts/render-report.sh`.

**Bug**: when 80% of findings are noise, the user has no fast way to
tell. The summary just said `89 findings (30 high, 35 medium, 24
low)`. Every finding looked equally credible.

**Fix proposals**:

a. **Confidence score per finding**. Each smell detector should emit
   a confidence in `[0.0, 1.0]` — high when the AST match is
   unambiguous, low when the heuristic is shaky. Use it as a fourth
   factor in the priority formula or as a separate column in the
   table. S05's confidence on a same-package helper with `error`
   return type would be 1.0; on a cross-package call it'd be 0.5;
   on a non-`error`-typed helper it'd be 0.0 (i.e. don't emit at
   all).

b. **Sample-and-verify pre-flight**. Before rendering the full
   report, run a tiny sampler: pick 5 random findings, classify
   each (is the line plausibly the smell described?). If <60% of
   the sample is plausible, the report header includes a banner:
   *"WARN: spot-check found N/5 false positives in the sample;
   manually verify before acting."* This is what the user wanted —
   fast trust signal.

### 9. The `apply-fixes.sh` step has nothing to do for these findings

**Code**: `scripts/apply-fixes.sh`.

**Observation**: every HIGH finding on this run needed either (a) a
**new test** to be added (covering an error branch the auto-triage
correctly flagged as missing) or (b) a strengthened assertion that
required understanding the test's intent. Neither is amenable to a
mechanical AST rewrite. The `apply-fixes.sh` step is essentially a
no-op for "branch-coverage gap" findings.

**Fix**: distinguish two finding categories in the schema:
- `fix:auto` — the fix is mechanical (delete the test, replace
  `require.NotNil` with `require.Equal(...)`, etc.). `apply-fixes.sh`
  handles these.
- `fix:manual` — the fix requires writing a new test or
  understanding intent. `apply-fixes.sh` writes a TODO comment in
  the test file instead, with a link back to the report.

The original schema treated every checkbox the same; checking a
manual-fix box gave the user a false expectation that the script
would do the work.

### 10. Missing: skill should detect "test in same package vs
        same-name assertion library" mismatch

**Observation** (not a bug, a future enhancement): a strong refine
pass would also flag tests that use `if got != want { t.Errorf(...) }`
in a codebase that otherwise standardizes on `require.Equal(...)`,
and vice versa. This kind of style drift creeps in over time and is
exactly the sort of thing a refine skill should catch on a review
pass.

**Suggestion**: read the user's `CLAUDE.md` (or the project's
`docs/development_guidelines.md`) for the project's preferred
assertion style and flag drift.

## Summary table — proposed prioritization

| # | Improvement | Impact | Effort |
|---|---|---|---|
| 1 | Type-aware S05 detector | Eliminates ~26 of 30 top findings | M |
| 6 | Build-tag-aware coverage on `--scope diff` | Restores branch-gap signal | M |
| 7 | Don't default `branch_gap=0.0` when coverage missing | Restores priority ordering | S |
| 2 | S01: skip when body is single delegation | Eliminates 2 false positives | S |
| 3 | S01: recurse into `t.Run` closures | Eliminates 1 false positive | S |
| 8a | Confidence score per finding | Across-the-board trust signal | M |
| 5 | State-machine PBT detection | Adds 1+ candidate | L |
| 4 | Soften S08 on restart-variant pairs | Eliminates 2 false positives | S |
| 9 | `fix:auto` vs `fix:manual` schema | Sets correct user expectation | S |
| 8b | Pre-flight sampler banner | Fast trust signal in header | M |
| 10 | Project assertion-style drift detector | New capability | L |

## Concrete one-line changes worth doing today

- In `triage.sh`, add a warning when test files in scope contain a
  `//go:build` directive and coverage is being run without matching
  `-tags`.
- In `score.go`, when `branch_gap` is unavailable for every finding,
  re-weight the formula to drop that term and renormalize.
- In `render-report.sh`, when a finding's `(file, line)` matches
  another finding for the same `(test, smell)`, collapse them into
  one row (one helper produced 5 identical rows in the same test).
