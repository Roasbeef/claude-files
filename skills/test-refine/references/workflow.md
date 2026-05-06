# Workflow: Phase-by-Phase Walkthrough

Detailed walkthrough of the two-phase `test-refine` workflow with examples.

## Phase A: Triage

### Step 1: Resolve Scope

```bash
~/.claude/skills/test-refine/scripts/triage.sh --scope <mode> [args]
```

| Mode | Args | Behavior |
|---|---|---|
| `package` (default) | none | Current directory's package |
| `file` | `--target path/to/foo_test.go` | Single test file |
| `diff` | `--base main` | Test files changed in current branch vs base |
| `repo` | none | Every `_test.go` in the module |

Slug for the report filename derives from scope: `package-<basename>`, `file-<basename>`, `diff-<branch>`, `repo`.

### Step 2: Coverage

The script runs:

```bash
go test -cover -covermode=atomic -coverprofile=/tmp/test-refine-cov.out <pkgs>
go tool cover -func=/tmp/test-refine-cov.out > /tmp/test-refine-cov.txt
```

Coverage is parsed per function. Branch coverage is computed by counting blocks in the cover profile (Go's `-covermode=atomic` records per-block hit counts).

The skill does *not* run `go test -count=1 -race` here — that's saved for Phase B. Triage is read-only.

### Step 3: Mutation Data (Optional)

When `--use-mutations` is set:

```bash
~/.claude/skills/mutation-testing/scripts/unleash.sh \
    --pkg <scope> \
    --output /tmp/test-refine-gremlins.json \
    --silent
```

Skip if gremlins isn't installed (warns, continues). Skip if scope is too broad (whole repo) — warn user to narrow.

### Step 4: AST Analysis

Three Go programs run sequentially against the resolved file set:

1. `detect-smells` — emits findings for S01–S11.
2. `detect-duplicates` — emits S08 (duplicate test bodies).
3. `domain-checks` — emits `D-CONCURRENCY-*`, `D-ERR-*`, `D-CTX-*`, `D-FAULT-*`, `D-PBT-*`, `D-DETERMINISM-*`.

If gremlins data is present, a fourth pass cross-references LIVED mutants with smell findings and emits S12.

Each finding is a JSON object:

```json
{
  "file": "internal/wallet/wallet_test.go",
  "line": 42,
  "test_name": "TestCalculateFee",
  "smell": "S01",
  "severity": "H",
  "message": "Test runs CalculateFee but asserts nothing.",
  "function_under_test": "internal/wallet.CalculateFee",
  "proposed_action": "Add assertion on returned fee value.",
  "context": "...10 lines of code around the smell...",
  "rewrite_suggestion": "...10 lines of strengthened test..."
}
```

### Step 5: Score

`score.go` reads all findings and the coverage report. For each finding, it computes:

```
priority = 0.5 * risk_score(file)
         + 0.3 * severity(smell)
         + 0.2 * branch_gap(function_under_test)
```

Findings are sorted by priority descending.

### Step 6: Render Report

`render-report.sh` consumes findings JSON and the coverage data, fills `assets/refinement-report.md.tmpl`, and writes:

```
.reviews/test-refinement/<YYYY-MM-DD>-<scope-slug>.md
```

The report sections:

1. **Header**: scope, date, mutation-data y/n.
2. **Before metrics**: test count, coverage %, `test_efficacy` if known.
3. **Top findings table**: rank, file:line, smell, severity, action.
4. **Per-function detail blocks**: for each top finding, original code + proposed rewrite.
5. **Domain checks**: separated by dimension.
6. **Removal candidates**: list with checkboxes.
7. **PBT candidates**: list with property sketches.
8. **Apply instructions**: how to invoke Phase B.

### Step 7: User Reviews

The user reads the report. To approve a fix, they check its checkbox in the markdown file:

```markdown
- [x] **F1** — `wallet_test.go:42` — S01: add assertion on `got` (fee value).
- [ ] **F2** — `wallet_test.go:88` — S08: duplicate of TestCalculateFee_2.
```

Unchecked items are skipped in Phase B. The user can also add `<!-- skip: reason -->` comments for tracked rejections.

## Phase B: Apply Fixes

```bash
~/.claude/skills/test-refine/scripts/apply-fixes.sh \
    --report .reviews/test-refinement/2026-05-06-package-wallet.md \
    [--verify-mutations]
```

### Step 1: Parse Report

The script:

1. Reads the report.
2. Extracts checked findings (`- [x]`).
3. For each finding, reads the JSON store at `.reviews/test-refinement/_findings/<report-slug>.json` to get the full action data (rewrite text, file/line ranges).

### Step 2: Apply Fixes

For each checked finding, the appropriate action:

- **Strengthen** — `Edit` the test file to replace the weak code with the proposed rewrite.
- **Add test** — `Edit` the test file to append a new test function.
- **Reshape** — `Edit` the test file to replace the example test with the property-based version.
- **Remove** — `Edit` the test file to delete the test function. Only if the checkbox under the "Removal candidates" section was checked.
- **Inject clock/RNG** — non-trivial; the skill writes the new test code but flags the SUT change as a separate finding ("SUT needs Clock injection") for the user to apply manually.

All edits go through `Edit`. The skill never uses `Bash` to `sed` or rewrite files.

### Step 3: Verify

```bash
go test ./... -race -count=1 <scope>
```

If tests fail, the script:

1. Stops applying further fixes.
2. Reports the failure to the user with the test output.
3. Does *not* roll back automatically. The user can `git diff` and `git checkout` rejected hunks if needed.

### Step 4: (Optional) Re-run Mutation Testing

When `--verify-mutations` is set:

```bash
~/.claude/skills/mutation-testing/scripts/unleash.sh \
    --pkg <scope> \
    --output /tmp/test-refine-gremlins-after.json \
    --silent
```

The script computes `test_efficacy_delta = after - before`. If negative for the touched package, it warns:

```
WARNING: test_efficacy regressed (-2.3%). Reshape may have weakened tests.
Inspect:
  before: .reviews/test-refinement/_findings/<slug>-gremlins-before.json
  after:  .reviews/test-refinement/_findings/<slug>-gremlins-after.json
```

### Step 5: Append "After" Section

The same report file gets a new section appended:

```markdown
## After Refinement (2026-05-06 14:23)

| Metric | Before | After | Δ |
|---|---|---|---|
| Test count | 42 | 38 | -4 (4 removed) |
| Branch coverage | 78.2% | 81.5% | +3.3% |
| test_efficacy | 76.0% | 89.4% | +13.4% |

**Applied fixes**: F1, F3, F5, F8, F9, F12 (6 of 14 proposed)
**Rejected**: F2, F4, F6, F7, F10, F11, F13, F14
```

### Step 6: Surface Diff

The script prints a `git diff --stat` of changed test files and exits.

## Example: End-to-End

```bash
# 1. Triage with mutation data on the wallet package.
$ ~/.claude/skills/test-refine/scripts/triage.sh \
    --scope package \
    --pkg ./internal/wallet \
    --use-mutations
Resolved scope: package=./internal/wallet
Coverage: 78.2% (1340/1714 statements)
Gremlins: efficacy=76.0%, lived=8 mutants
Found 14 findings (3 H, 8 M, 3 L), 4 removal candidates, 2 PBT candidates.
Report: .reviews/test-refinement/2026-05-06-package-wallet.md

# 2. User opens the report, reviews, checks boxes.
$ $EDITOR .reviews/test-refinement/2026-05-06-package-wallet.md

# 3. Apply checked fixes, verify with mutations.
$ ~/.claude/skills/test-refine/scripts/apply-fixes.sh \
    --report .reviews/test-refinement/2026-05-06-package-wallet.md \
    --verify-mutations
Parsing report... 6 of 14 findings approved, 2 of 4 removals approved.
Applying F1 (strengthen)... done.
Applying F3 (strengthen)... done.
Applying F5 (reshape to PBT)... done.
Applying F8 (add error path test)... done.
Applying F9 (strengthen)... done.
Applying F12 (remove duplicate)... done.
go test ./internal/wallet -race -count=1 ... ok
Re-running mutations... efficacy=89.4% (+13.4%)
Diff: 4 files changed, 87 insertions(+), 32 deletions(-)
Report updated: .reviews/test-refinement/2026-05-06-package-wallet.md (After Refinement section)
```

## Failure Modes

### "go test fails after applying fixes"

The script stops, prints test output, and leaves the test file in the partially-edited state. The user can:

1. Inspect `git diff` to see what was applied.
2. Manually fix the failing test, or `git checkout -- <file>` to revert.
3. Re-run `apply-fixes.sh` (it skips already-applied fixes via the `--applied` cache).

### "Mutation efficacy regressed"

The script warns but continues — the user may have intentionally consolidated tests in a way that's net-better but lowers raw efficacy. Investigate before merging.

### "Report has no checked items"

`apply-fixes.sh` exits with a "no fixes approved" message. No-op.

### "Gremlins not installed and --use-mutations passed"

`triage.sh` warns and falls back to AST/coverage-only mode. The S12 findings are skipped. Run `~/.claude/skills/mutation-testing/scripts/install-gremlins.sh` and rerun.

## Tips

- **Keep scope narrow.** A whole-repo run produces a report too large to review. Iterate per package.
- **Run before opening the PR.** Reviewers shouldn't see the smells; the report should already be applied.
- **Don't auto-approve.** The "edit checkboxes by hand" step is intentional friction. Bulk approval defeats the safety guarantee.
- **Pair with mutation testing.** AST smells are heuristics; mutation survivors are evidence. The combination is much stronger than either alone.
