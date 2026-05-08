# 2026-05-08 — mutation testing track

Exercised `--use-mutations` end-to-end on a synthetic minimal Go module
to validate the path that the user reported was failing on their
private repo (gremlins couldn't download deps because of an
uninitialised submodule).

## What works

- `triage.sh --scope package --pkg <dir> --use-mutations` produces a
  gremlins JSON, cross-references LIVED mutants into S12 findings,
  and renders them in the report.
- For a fixture with weak tests the S12 mapping is correct: a test
  that asserts `Sub(5,3) >= 0` instead of `Sub(5,3) == 2` correctly
  surfaced both `ARITHMETIC_BASE` and `CONDITIONALS_BOUNDARY` as
  survivors.
- The diff-scope fanout from the previous round (capped by
  `MUTATION_FANOUT_CAP=5`) is in place.

## What broke / was misleading

1. **`score.go` emitted `null` when there were zero findings**, which
   crashed the render-report step with `Cannot iterate over null`.
   Fixed: emit `[]` instead.

2. **`apply-fixes.sh --post --verify-mutations` reported `Δ -33%`
   when the after-run produced zero mutants** — flatly misleading,
   since "0% efficacy of 0 mutants" is not a regression. Fixed:
   detect the zero-mutant case explicitly and print
   `comparison invalid: zero-mutant side`, with a corresponding
   warning banner appended to the report rather than a clean delta
   table.

3. **The second gremlins run on the same package frequently produces
   zero mutants** even when the first run found several. Reproduced
   with `gremlins unleash` directly (no apply-fixes wrapper). This
   appears to be a gremlins-side behaviour — possibly related to its
   build-cache invalidation or to `go test`'s test-results cache.
   Worth investigating, but not blocking — the new "comparison
   invalid" path keeps users from being misled.

## What is still aspirational in SKILL.md

- **Equivalent-mutant filter for S12**: SKILL.md says "When gremlins
  reports `LIVED`, the AST is checked for equivalence patterns
  (mutated value immediately overwritten, mutation in unreachable
  code, associative no-op) before flagging as S12." This is not
  implemented — the cross-reference is a straight `select(.status ==
  "LIVED")`. A proper equivalence pass would need to load the
  mutated source position and inspect surrounding AST. Worthwhile
  follow-up for the next round.

- **`mutants_not_covered` is silently dropped.** Gremlins reports
  three buckets: KILLED, LIVED, NOT_COVERED. The cross-ref only
  picks up LIVED. NOT_COVERED is "the test suite never exercises
  the line" — that's also a real test gap, but currently invisible
  in the report. Could be surfaced as a softer S12-COVERAGE finding
  with confidence 0.5.

## Recommended follow-ups (next round)

A. Implement the equivalent-mutant filter promised by SKILL.md.
B. Surface `mutants_not_covered` as S12-COVERAGE.
C. Investigate the second-run-yields-zero-mutants behaviour in
   gremlins; report upstream if it's a real bug.
D. Add a CI-style regression: `testdata/mutation-smoke/` with a
   weak-tests fixture and an `expected-survivors.txt` so we'd
   notice if S12 output shape changes.
