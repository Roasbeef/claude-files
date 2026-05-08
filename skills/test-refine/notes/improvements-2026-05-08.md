# 2026-05-08 — feedback round from a real-world Go diff

User ran `--scope diff --use-mutations` on a 16-test-file / ~9k-line
diff in a private Go codebase. The domain-check signal was real and
actionable; the smell side had three high-impact false-positive classes
that risked the user deleting strong tests if they trusted the report.
This round addresses the verified issues and the surfacing problems
that hid the bugs.

## Verified false positives — fixed

- **S01 false positives × 4**. Two root causes:
  - `assertCalls` was missing testify's error-string family
    (`ErrorContains`, `ErrorContainsf`, `EqualError`, `EqualErrorf`,
    plus other commonly-missing methods: `Subset`, `ElementsMatch`,
    `IsType`, `Implements`, `Regexp`, etc.).
  - The detector didn't recognise user-defined helper assertions
    (e.g. `h.fooBar.assertSomething(...)`). Now any call whose name
    matches `^(assert|require|verify|expect|check|must)` + uppercase
    / underscore / end-of-string counts as an assertion.
- **S06 false positives × 2** on canonical `.String()` types whose
  String() form is the idiomatic comparison (hashes, UUID-typed
  identifiers). Added a canonical-receiver allowlist — receiver
  expression's terminal selector/ident name ending in `Hash`, `UUID`,
  `Int`, `Time`, `OutPoint`, `PubKey`, etc. exempts the comparison.
- **S09 noise** (24/30 top findings were S09). Tightened to require ≥4
  shape-duplicated asserts in a test with ≥8 total bare asserts.
  Confidence dropped to 0.4 (advisory).

## Surfacing fixes — so silent failures become visible

- **Coverage 0% silently swallowed** → header banner "Data layer was
  incomplete" surfaces when `go test -cover` produced an empty profile
  with `--coverage` requested.
- **`--use-mutations` silently dropped on diff/repo scope** →
  per-package fanout for diff scope (capped by `MUTATION_FANOUT_CAP=5`,
  override via env), hard-fail for repo scope.
- **Branch-gap renormalize buried in stderr** → header banner
  "Priority is severity-only" surfaces when every finding has gap=0.
- **Domain checks skipped on diff scope** → fanout across affected
  package dirs; this round 160 actionable domain findings appeared
  where 0 had before.

## UX fixes

- **`--base` defaulted to `main`** → auto-detect via
  `git symbolic-ref refs/remotes/origin/HEAD`, falling back to `main`
  then `master`.
- **Test bodies missing from finding blocks** → render-report.sh now
  embeds the test function body in a collapsible `<details>` block
  inside each detail entry. Truncates at column-0 `}` or 30 lines.
- **"Function under test" empty for smell findings** → derived from
  the test name via package-wide function index. detect-smells.go now
  takes production `.go` files alongside test files so the index
  contains both `Foo` (free fn) and `Receiver.Method` keys. No
  body-scan fallback — empty SUT > wrong SUT.
- **Top-table message column truncated mid-word** → dropped from the
  table entirely. The detail block has the full message + suggestion
  + body. Table is for at-a-glance scan only.
- **No "false positive" affordance** → second checkbox per finding so
  reviewer-disagreement is recordable across re-runs.

## What is intentionally not done yet

- S05 dataflow accuracy (when a callee's error type is unknown).
- Mutation efficacy verification on apply (we accept gremlins output
  as advisory — re-running gremlins post-fix is left to
  `--verify-mutations` which is implemented but not exercised here).
- Body-scan SUT fallback. Without proper type information, the
  fallback always picked the wrong call (harness setup, factory). The
  current state — empty SUT when prefix match fails — is more honest.
