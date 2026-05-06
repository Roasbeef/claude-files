# Mutation Testing Report

**Date**: {{DATE}}
**Module**: {{GO_MODULE}}
**Scope**: {{SCOPE}}
**Tool**: gremlins {{GREMLINS_VERSION}}

---

## Summary

| Metric | Value |
|--------|-------|
| Test efficacy | **{{TEST_EFFICACY}}%** |
| Mutations coverage | {{MUTATIONS_COVERAGE}}% |
| Mutants killed | {{MUTANTS_KILLED}} |
| Mutants LIVED | {{MUTANTS_LIVED}} |
| Mutants not covered | {{MUTANTS_NOT_COVERED}} |
| Mutants not viable | {{MUTANTS_NOT_VIABLE}} |
| Total mutants | {{MUTANTS_TOTAL}} |
| Elapsed | {{ELAPSED_TIME}}s |

`test_efficacy = killed / (killed + lived)` — fraction of covered mutants that the test suite caught.
`mutations_coverage = (killed + lived) / (killed + lived + not_covered)` — fraction of code the test suite exercises.

---

## Quality Assessment

{{QUALITY_BADGE}}

{{QUALITY_MESSAGE}}

Targets:

| Code class | Target efficacy |
|---|---|
| Mission-critical | 90%+ |
| Core business logic | 80–90% |
| General | 70–80% |

---

## Surviving Mutants (LIVED)

The following mutants were not killed by the test suite. Each represents either a missing test, a weak assertion, or an equivalent mutant.

{{#if SURVIVORS}}
| File | Line:Col | Mutator | Implied gap |
|---|---|---|---|
{{#each SURVIVORS}}
| {{FILE}} | {{LINE}}:{{COLUMN}} | `{{TYPE}}` | {{IMPLIED_GAP}} |
{{/each}}

### Detail

{{#each SURVIVORS}}
#### {{FILE}}:{{LINE}}:{{COLUMN}} — `{{TYPE}}`

**Original** (line {{LINE}}):
```go
{{ORIGINAL_CONTEXT}}
```

**Mutated**:
```go
{{MUTATED_CONTEXT}}
```

**Why this matters**: {{WHY_MATTERS}}

**Recommended action**: {{RECOMMENDED_TEST}}

---

{{/each}}
{{else}}
All covered mutants killed. No survivors.
{{/if}}

---

## Mutator Breakdown

| Mutator | Killed | LIVED | Not covered | Efficacy |
|---|---|---|---|---|
{{#each MUTATOR_STATS}}
| `{{TYPE}}` | {{KILLED}} | {{LIVED}} | {{NOT_COVERED}} | {{EFFICACY}}% |
{{/each}}

Use this to identify which *kinds* of mutations the test suite struggles with — see `references/mutation_operators.md` for the implied test gap per mutator.

---

## File Breakdown

| File | Killed | LIVED | Not covered | Efficacy |
|---|---|---|---|---|
{{#each FILE_STATS}}
| {{FILE}} | {{KILLED}} | {{LIVED}} | {{NOT_COVERED}} | {{EFFICACY}}% |
{{/each}}

---

## Not Covered

{{MUTANTS_NOT_COVERED}} mutations were in code paths no test exercises. These are *coverage* gaps (line/branch), not assertion gaps.

{{#if NOT_COVERED_LIST}}
| File | Line:Col | Mutator |
|---|---|---|
{{#each NOT_COVERED_LIST}}
| {{FILE}} | {{LINE}}:{{COLUMN}} | `{{TYPE}}` |
{{/each}}
{{/if}}

---

## Recommendations

### Immediate

{{#if LOW_EFFICACY}}
- Test efficacy is below target. Focus on the surviving mutants in critical paths first.
- Strengthen assertions: every test should fail if the function under test computes a different value.
{{/if}}

{{#if BOUNDARY_SURVIVORS}}
- {{BOUNDARY_SURVIVORS}} `conditionals-boundary` mutants survived. Add tests at exact threshold values (`<` vs `<=`, `>` vs `>=`).
{{/if}}

{{#if LOGICAL_SURVIVORS}}
- {{LOGICAL_SURVIVORS}} `invert-logical` mutants survived. Add truth-table tests that exercise each `&&`/`||` operand independently.
{{/if}}

{{#if ARITH_SURVIVORS}}
- {{ARITH_SURVIVORS}} `arithmetic-base` mutants survived. Tests are calling functions but not asserting on the computed result.
{{/if}}

### Long-term

- Add property-based tests (`rapid`) for invariants like commutativity, idempotence, roundtrip.
- Wire a `threshold` gate into CI to prevent regressions.
- Review surviving mutants periodically — equivalent mutants accumulate.

---

## Configuration

```yaml
{{CONFIG_DUMP}}
```

---

## Next Steps

1. Triage `LIVED` mutants: real gap, equivalent mutant, or false positive.
2. Add or strengthen tests for real gaps.
3. Re-run gremlins on the same scope to confirm killed.
4. Update `EQUIVALENT_MUTANTS.md` for documented equivalents.
5. Set or update threshold in `.gremlins.yaml`.

---

Generated from gremlins JSON at `{{INPUT_PATH}}`. See [`SKILL.md`](../SKILL.md) and [`references/best_practices.md`](../references/best_practices.md).
