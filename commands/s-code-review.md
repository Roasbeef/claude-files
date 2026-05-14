---
description: "Loop a Substrate review→fix→resubmit cycle until reviewer approves, with fixup commits per finding"
argument-hint: "[--type=full|security|performance|architecture] [--base=<branch>] [--max-iters=<N>]"
allowed-tools:
  - Bash
  - Read
  - Edit
  - Grep
  - Glob
  - TodoWrite
  - AskUserQuestion
  - Task
---

# Substrate Review Loop

Drive a full review→fix→resubmit loop against the Substrate review system. You
will submit the review, wait for the reviewer agent, address Critical/High/
Medium findings with precision fixup commits, and resubmit. The loop exits
only when the reviewer's iteration decision is `approve`.

**Default behavior:**
- Review type: `full` (override with `--type=...`)
- Base branch: auto-detect `main` then `master` (override with `--base=...`)
- Max iterations: 10 (override with `--max-iters=N`; safety cap, not a target)
- Severity cutoff for in-loop fixes: **critical, high, medium**
- Low / informational findings are logged but NOT fixed in-loop (surface in
  the final summary instead).

**Communication rules:**
- All status communication to the user goes through Substrate mail
  (`substrate send --to User ...`), NOT normal chat output. Normal output
  stays terse — one-line progress notes only.
- If the agent suspects a finding is a **false positive**, do NOT fix it
  silently. Mail the user with the dispute rationale, mark the finding as
  `disputed` in the iteration log, and continue.
- If something genuinely blocks the loop, mail the user — don't print to
  chat and wait.

## Phase 0: Setup

Run these checks once at the start:

```bash
# Confirm we're on a feature branch with commits ahead of base
git branch --show-current
git log --oneline @{u}..HEAD 2>/dev/null || git log --oneline origin/main..HEAD 2>/dev/null
git status --short
```

Decide the base branch:
```bash
git show-ref --verify --quiet refs/heads/main && echo main || echo master
```

Create the run log directory (populated as the loop progresses):
```bash
mkdir -p .s-code-review
```

If the working tree is dirty (uncommitted changes), STOP and mail the user:
```bash
substrate send --session-id "$CLAUDE_SESSION_ID" --to User \
  --subject "/s-code-review blocked: dirty working tree" \
  --body "Cannot start review loop — uncommitted changes present. Commit or stash first."
```

## Phase 1: Submit Initial Review

```bash
REVIEW_TYPE="${ARG_TYPE:-full}"
BASE_BRANCH="${ARG_BASE:-main}"

REVIEW_JSON=$(substrate review request \
  --session-id "$CLAUDE_SESSION_ID" \
  --type "$REVIEW_TYPE" \
  --base "$BASE_BRANCH" \
  --format json)

REVIEW_ID=$(echo "$REVIEW_JSON" | jq -r '.review_id')
echo "$REVIEW_ID" > .s-code-review/current-review-id
mkdir -p ".s-code-review/$REVIEW_ID"
```

Record the kickoff in the run log:
```bash
cat > ".s-code-review/$REVIEW_ID/SUMMARY.md" <<EOF
# Review $REVIEW_ID

- **Type**: $REVIEW_TYPE
- **Base**: $BASE_BRANCH
- **Branch**: $(git branch --show-current)
- **Started**: $(date -Iseconds)

## Iterations
EOF
```

## Phase 2: Wait for Reviewer (Hybrid)

Use the hybrid strategy: short tight polling, then long-poll on inbox.

**Stage A — tight polling (3 × 15s):**
```bash
for i in 1 2 3; do
  state=$(substrate review status "$REVIEW_ID" \
    --session-id "$CLAUDE_SESSION_ID" --format json | jq -r '.state')
  case "$state" in
    approved|changes_requested|rejected|cancelled) break ;;
  esac
  sleep 15
done
```

**Stage B — long-poll fallback** (only if Stage A timed out):
```bash
while true; do
  state=$(substrate review status "$REVIEW_ID" \
    --session-id "$CLAUDE_SESSION_ID" --format json | jq -r '.state')
  case "$state" in
    approved|changes_requested|rejected|cancelled) break ;;
  esac
  # Block up to 55s waiting for any new mail; reviewer notifications wake us.
  substrate poll --session-id "$CLAUDE_SESSION_ID" --wait=55s --quiet >/dev/null
done
```

If state becomes `rejected` or `cancelled`, mail the user with the reviewer's
summary and stop — this is a non-recoverable exit:
```bash
substrate review status "$REVIEW_ID" --session-id "$CLAUDE_SESSION_ID" \
  --format json | jq -r '.iteration_details[-1].summary' \
  | substrate send --session-id "$CLAUDE_SESSION_ID" --to User \
      --subject "Review $REVIEW_ID terminated: $state" --body-stdin
```

## Phase 3: Triage the Latest Iteration

Fetch status and issues:
```bash
STATUS=$(substrate review status "$REVIEW_ID" \
  --session-id "$CLAUDE_SESSION_ID" --format json)
ISSUES=$(substrate review issues "$REVIEW_ID" \
  --session-id "$CLAUDE_SESSION_ID" --format json)

LATEST_ITER=$(echo "$STATUS" | jq '.iterations')
LATEST_DECISION=$(echo "$STATUS" | jq -r ".iteration_details[-1].decision")
LATEST_SUMMARY=$(echo "$STATUS"  | jq -r ".iteration_details[-1].summary")
```

**Termination check** — exit the loop the moment the latest iteration
decision is `approve`:
```bash
if [ "$LATEST_DECISION" = "approve" ]; then
  # Jump to Phase 7.
  :
fi
```

Otherwise, filter to **open issues from the latest iteration** with severity
in {critical, high, medium}:
```bash
echo "$ISSUES" | jq --argjson iter "$LATEST_ITER" '
  .issues[]
  | select(.iteration_num == $iter
           and .status == "open"
           and (.severity | IN("critical","high","medium")))
'
```

Write the per-iteration log skeleton:
```bash
ITER_LOG=".s-code-review/$REVIEW_ID/iter-$LATEST_ITER.md"
cat > "$ITER_LOG" <<EOF
# Iteration $LATEST_ITER

**Decision**: $LATEST_DECISION
**Summary**: $LATEST_SUMMARY

## Findings to address
EOF
```

## Phase 4: Scrutinize Each Finding (False-Positive Filter)

For **every** must-fix finding, before changing code:

1. **Read the cited code** — use `Read` on `file_path` around `line_start`
   with enough context (±30 lines).
2. **Verify the claim** — does the reviewer's description match what the
   code actually does? Check git blame on that range to understand intent.
3. **Decide one of three labels:**
   - `valid` — the finding is correct, plan a fix.
   - `disputed` — strong evidence the reviewer misread the code or
     misapplied a rule. Examples: invariant already enforced upstream,
     pattern is idiomatic in this codebase, "missing test" claim ignores
     existing coverage at another path.
   - `out-of-scope` — finding is real but requires changes beyond this PR's
     scope (e.g., requires schema migration, cross-package refactor).

4. **For `disputed`**, mail the user with the rationale — do NOT fix:
   ```bash
   substrate send --session-id "$CLAUDE_SESSION_ID" --to User \
     --subject "Disputed finding: review $REVIEW_ID issue #<id>" \
     --body "[Context: /s-code-review loop, iteration $LATEST_ITER]

   Reviewer: <reviewer_id>
   Finding: <title>
   Location: <file>:<line>

   Why I think this is a false positive:
   <rationale, 3-6 bullets>

   Proceeding with the rest of the iteration. Reply if you want me to
   fix it anyway or push back on the reviewer."
   ```

5. **For `out-of-scope`**, mail the user but also log it for the final
   summary so it can become a follow-up.

Append each finding (valid / disputed / out-of-scope) to `iter-N.md` with
its label and rationale before moving on. The log is the audit trail the
user will read.

## Phase 5: Address Valid Findings with Fixup Commits

For each `valid` finding, follow the precision-commit recipe. Use `TodoWrite`
to track each finding as a task while you work through them.

### 5.1 — Find the originating commit
```bash
git blame -L <line>,<line> -- <file>
```

Capture the commit SHA from the blame output. That is the fixup target. If
the finding spans multiple files from different originating commits, plan
one fixup per (commit, files-from-that-commit) group — never bundle changes
that didn't all originate from the same commit, or `git rebase --autosquash`
will fail with "deleted by us".

### 5.2 — Make the change
Use `Edit` to apply the minimal fix. Stay laser-focused on the specific
finding; do NOT make tangential cleanups in the same change.

### 5.3 — Stage with `hunk` (precision)
```bash
hunk diff --json                       # confirm exact line numbers
hunk stage <file>:<startLine>-<endLine>
hunk preview                            # verify
```

Fallback if `hunk stage` fails ("patch does not apply"):
```bash
git add <file>
```

### 5.4 — Create the fixup commit
```bash
git commit --fixup=<originating-sha>
```

If `git blame` did not yield a usable target (e.g., new file from the same
PR's first commit, or the change is structural and spans many commits),
fall back to a regular focused commit:
```bash
git commit -m "address review $REVIEW_ID: <one-line description>"
```

### 5.5 — Append to the iteration log
Record the fixup in `iter-N.md`:

```markdown
### Issue #<id> [<severity>] <title>
- **File**: <file>:<line>
- **Origin commit**: <blame-sha>
- **Fixup commit**: <fixup-sha>
- **Change**: <one-line description of what was fixed>
```

Mark the corresponding `TodoWrite` task `completed` immediately — don't
batch.

## Phase 6: Resubmit

After processing every valid finding for this iteration:

```bash
substrate review resubmit "$REVIEW_ID" --session-id "$CLAUDE_SESSION_ID"
```

Update SUMMARY.md with a one-line iteration roll-up:
```bash
echo "- Iter $LATEST_ITER: <N> fixed, <M> disputed, <K> out-of-scope" \
  >> ".s-code-review/$REVIEW_ID/SUMMARY.md"
```

Then **loop back to Phase 2** (wait for the next iteration).

**Safety cap:** if `LATEST_ITER >= max-iters` (default 10) and we still
haven't seen `approve`, mail the user and stop. This is a circuit breaker,
not a normal exit — explain in the mail why convergence failed (e.g.,
reviewer keeps surfacing new findings, mutual misunderstanding on a class
of issues, etc.).

## Phase 7: Approval & Final Summary

When `LATEST_DECISION == "approve"`:

1. **Finalize SUMMARY.md** with the approval iteration and total counts:
   ```bash
   cat >> ".s-code-review/$REVIEW_ID/SUMMARY.md" <<EOF

   ## Final
   - **Approved at iteration**: $LATEST_ITER
   - **Total fixups**: $(git log --oneline @{u}..HEAD | grep -c '^[a-f0-9]* fixup!')
   - **Disputed (not fixed)**: <count from logs>
   - **Out-of-scope (follow-up)**: <count from logs>
   - **Approver summary**: $LATEST_SUMMARY
   EOF
   ```

2. **Mail the user** with the verdict and a pointer to the log:
   ```bash
   substrate send --session-id "$CLAUDE_SESSION_ID" --to User \
     --subject "Review $REVIEW_ID APPROVED after $LATEST_ITER iterations" \
     --body "$(cat .s-code-review/$REVIEW_ID/SUMMARY.md)"
   ```

3. **Send the diff** so the user sees what changed in the web UI:
   ```bash
   substrate send-diff --session-id "$CLAUDE_SESSION_ID" --to User \
     --base "$BASE_BRANCH" \
     --subject "Review $REVIEW_ID — final diff"
   ```

4. **Suggest the autosquash** (do NOT run it automatically — the user owns
   history rewrites):
   ```
   Next step you may want to run:
     hunk rebase autosquash --onto <base>
     git push --force-with-lease
   ```

## Important Notes

- **Severity cutoff is firm.** Critical/High/Medium are fixed in-loop. Low
  and Informational are logged in the iteration file but NOT acted on.
  Surface them in the final SUMMARY.md so the user can decide what's worth
  a follow-up.
- **False positives are real.** Reviewer agents will occasionally misread
  code. Always read the cited lines yourself before fixing. When in doubt,
  mark `disputed` and mail the user — never silently "fix" something you
  don't agree with.
- **One fixup per originating commit.** Use `git blame` to determine the
  target. Never bundle changes from multiple originating commits into one
  fixup — autosquash will break on "deleted by us" conflicts.
- **Iteration log is the deliverable.** The user can `cat
  .s-code-review/<review-id>/SUMMARY.md` to see exactly what was fixed,
  disputed, or deferred across every round. Keep it tidy.
- **Mail-first communication.** Status updates, blockers, and disputed
  findings go to the user via `substrate send`. Normal chat output stays
  brief — single-line progress notes only.
- **The Substrate Stop hook is your friend.** Between iterations, if the
  reviewer takes a while, the Stop hook's long-poll keeps the agent alive
  while you wait. Don't try to outsmart it with custom sleep loops beyond
  the hybrid pattern in Phase 2.
