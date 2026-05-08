# apply-fixes.sh capability matrix

What the script *can* mechanically do per smell ID, vs what
`fix_kind` it emits, vs what the consuming agent has to do.

## Reading the table

- **fix_kind**: what the smell detector emits (`auto` or `manual`).
- **In plan as**: which JSON array the script puts the finding in
  (`approved_auto`, `approved_manual`, `approved_removals`,
  `approved_reshapes`, `approved_domain`).
- **Mechanical action**: what the script could in principle do
  without consulting the consuming agent.
- **Today**: what actually happens on `apply-fixes.sh --report ...`.

## Smell-level matrix

| ID | fix_kind | In plan as | Mechanical action possible | Today |
|---|---|---|---|---|
| S01 | manual | approved_manual | Delete the test (only with explicit Removal checkbox under "## Removal Candidates") | Plan-only; consumer applies |
| S02 | auto | approved_auto | Delete the tautological assertion line | Plan-only; consumer applies |
| S03 | auto | approved_auto | Delete the trivial getter/setter test | Plan-only; consumer applies |
| S04 | manual | approved_manual | None — needs a real assertion written | Plan-only; "TODO: …" comment guidance |
| S05 | manual | approved_manual | Replace `_ = X(…)` with `err := X(…); require.NoError(t, err)` (mechanical when callee is unambiguously error-returning) | Plan-only; consumer applies |
| S06 | manual | approved_manual | None — substitution depends on what the user *meant* to compare | Plan-only |
| S07 | manual | approved_manual | None — restructuring | Plan-only |
| S08 | auto | approved_auto | Delete one duplicate; collapse pair into table-driven (table-driven needs human input) | Plan-only |
| S08-VARIANT | manual | approved_manual | Suggest table-driving; not auto | Plan-only |
| S09 | manual | approved_manual | Add message strings to assertions (mechanical, low-value) | Plan-only |
| S10 | manual | approved_manual | None — needs intent | Plan-only |
| S11 | manual | approved_manual | None — needs intent | Plan-only |
| S12 | manual | approved_manual | None — write a test that kills the mutant | Plan-only |
| S-STYLE-DRIFT | auto | approved_auto | Convert minority style to majority style | Plan-only |
| D-CONCURRENCY-MISSING | manual | approved_manual or approved_domain | None — write a concurrent test | Plan-only |
| D-ERR-PATH-MISSING | manual | approved_manual or approved_domain | None — write `require.ErrorIs` test | Plan-only |
| D-CTX-CANCEL-MISSING | manual | approved_manual or approved_domain | None — write a cancellation test | Plan-only |
| D-CTX-TIMEOUT-MISSING | manual | approved_manual or approved_domain | None — write a timeout test | Plan-only |
| D-PBT-CANDIDATE | manual | approved_reshapes | None — convert to rapid PBT | Plan-only |
| D-PBT-STATE-MACHINE | manual | approved_reshapes or approved_domain | None — `rapid.StateMachine` rewrite | Plan-only |
| D-DETERMINISM-CLOCK | manual | approved_manual or approved_domain | Replace `time.Now()` with injectable clock (signature change required) | Plan-only |
| D-DETERMINISM-RAND | manual | approved_manual or approved_domain | Same as above for `rand.*` | Plan-only |
| D-DETERMINISM-ENV | manual | approved_manual or approved_domain | Replace `os.Getenv` with config injection | Plan-only |

## Observations

1. **No smell currently has a script-side mechanical applier.** Even
   `fix_kind=auto` cases (S02 tautology delete, S03 getter/setter
   delete, S08 duplicate delete, S-STYLE-DRIFT) end up as JSON in the
   plan; the consumer (Claude) then runs Edit calls. The SKILL.md
   section that says "the script applies only the checked items …
   strengthen assertions in place / add missing branch tests / reshape
   into invariants" is aspirational, not implemented.

2. **`approved_removals` is the only path with a clear mechanical
   contract** (delete the named function from the file). Even there,
   the script emits a string blob rather than calling out to `gofmt`
   / `goimports` to safely remove the function body and any imports
   that become unused.

3. **Same finding may appear in two arrays.** A D-ERR-PATH-MISSING
   gets routed into `approved_manual` (when the F-block checkbox is
   ticked under "## Findings (Apply Approved)") *or* into
   `approved_domain` (when the corresponding line under "## Domain
   Checks" is checked). If both are checked, the consumer sees the
   finding twice. The split should pick one canonical location per
   finding.

4. **`approved_removals` / `approved_reshapes` / `approved_domain`
   carry raw markdown lines, not parsed objects.** Consumer has to
   regex-extract `file:line` out of `- [x] **Remove** \`pkg/foo_test.go:42\``.
   The schema would be cleaner if they joined back to `_findings/*.json`.

## Recommended next steps for apply-fixes

A. **Implement the auto-class fixers in the script**, even minimally:
   - S02: read file, find the assertion line, delete it (or comment it
     with `// TODO: tautology — replace`).
   - S03: delete the test function (use Go's
     `golang.org/x/tools/go/ast` walk to find the FuncDecl by name and
     remove its source range).
   - S08: same — delete the duplicate function.
   These three cover 100% of `fix_kind=auto` cases.

B. **De-duplicate domain findings between `approved_manual` and
   `approved_domain`**: pick one based on smell prefix (`D-*` always
   in `approved_domain`, `S*` always in `approved_manual`).

C. **Strongly type the action plan**: emit `approved_removals` etc.
   as the same finding-object shape (joined back to the JSON), not as
   raw markdown line strings. Consumer parsing becomes a JSON walk
   instead of a regex.

D. **Honour `--scope-pkg` in the post-phase `go test`** so a focused
   refactor doesn't trigger a full-suite run.
