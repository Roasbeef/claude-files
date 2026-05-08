# 2026-05-08 — apply-fixes.sh smoke test

Walked one D-ERR-PATH-MISSING finding through the apply path on a copy
of a real report. Findings, in order of severity.

## Bugs fixed in this session

1. **`set -u` + uninitialised arrays**: `apply-fixes.sh` errored
   immediately on the first run with
   `APPROVED_REMOVE[@]: unbound variable`, because bash's strict mode
   blows up on `${arr[@]}` for arrays that were never appended to.
   Fixed: guarded all four accumulator arrays with `${arr[@]:-}` and
   skip-when-empty in the loops. Without this fix the entire Phase B
   was unusable.

## Functional gaps observed

2. **Same finding routed two ways depending on which checkbox the
   user ticks.** D-ERR-PATH-MISSING for `CoSignArkPSBT` appears both
   under `## Findings (Apply Approved)` (as F1) and under
   `## Domain Checks` (as a one-line entry). Checking the F1 box
   routes the finding into `approved_manual`; checking the Domain
   Checks line routes it into `approved_domain`. The consumer has to
   know to look in both arrays — and there is no de-dup if both
   boxes are checked.

3. **No per-smell-id templated guidance.** The action plan emits one
   shared `guidance` string ("for 'auto' findings, apply the edit
   directly…"). For a D-ERR-PATH-MISSING finding the user actually
   needs a copy-pasteable test stub: a function header, a
   `require.ErrorIs` line targeting the suggested sentinel, etc.
   Currently the calling agent has to invent that from
   `suggestion`, which is one sentence.

4. **`fix_kind=auto` is never exercised** for any of the smells in
   the original feedback round. The split is `auto: 0 / manual: 1`.
   Smells that *could* be auto (S02 tautology delete, S03
   getter/setter delete, S08 duplicate delete) were either filtered
   to zero or would only ever be auto if the user opts into removal,
   which Phase B cannot decide on its own. So in practice
   apply-fixes.sh is always emitting "write a TODO" guidance.

5. **No actual file edits.** The script's job is to emit an action
   plan; the calling agent (Claude) is supposed to apply edits via
   the Edit tool. SKILL.md says:

   > The script applies **only the checked items**:
   > - Strengthen assertions in place.
   > - Add missing branch tests.
   > - Reshape into invariants…

   That's not what happens. The script produces a JSON action plan;
   the consumer is responsible for the edits. The wording in
   SKILL.md should be updated, or the script should grow a
   real edit path for the genuinely-mechanical cases.

6. **The `--post` flag re-runs `go test` against `./...`.** That's
   correct for a full-suite pass but is the wrong default after a
   diff-scoped run on a single package — it tests far more than the
   refactor touched. Should be `--scope-pkg` aware (currently
   defaults to `./...`).

## Recommended follow-ups

- (Schema) De-duplicate findings between `approved_manual` and
  `approved_domain` — pick one home per finding, not both.
- (Templates) Per-smell `guidance` strings (or a small template per
  smell-id) so the agent doesn't have to reverse-engineer every
  finding's suggested fix from a single-sentence message.
- (Honesty) Rewrite the SKILL.md section that promises "the script
  applies only the checked items" — the script emits a plan, the
  agent applies. Or implement at least the auto-class fixes
  (delete-test, delete-tautology) directly in bash via `sed` /
  `goimports` so the promise becomes true.
- (Mechanical) Honour `--scope-pkg` in the post-phase `go test`
  invocation.
