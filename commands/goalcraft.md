---
description: Interview-driven crafting of a /goal completion-condition string (<=4000 chars)
argument-hint: <@file|#issue|text describing what "done" looks like>
allowed-tools: [AskUserQuestion, Bash, Grep, Glob, Read, LS, WebFetch]
---

# /goalcraft - Craft an effective `/goal` string

Turn a rough "I want Claude to keep working until X" into a precise, evaluator-
friendly completion condition you can paste into `/goal`. The output is a single
string of **at most 4000 characters**.

This command writes the string. It does **not** set the goal itself — the final
step hands you the string and offers to set it.

## Background: what makes a good `/goal` condition

`/goal` keeps Claude working turn after turn until a small fast model (Haiku by
default) confirms the condition holds. Critical properties to design around:

- **The evaluator only sees the conversation.** It does **not** run commands or
  read files. So the condition must be something Claude's own surfaced output
  can *demonstrate*. "Tests in `test/auth` pass" works only because Claude runs
  them and the result lands in the transcript.
- **One measurable end state.** A test result, a build exit code, a file count,
  an empty queue, a clean `git status`.
- **A stated check.** Exactly how Claude proves it: "`npm test` exits 0",
  "`go build ./...` succeeds", "`git status` is clean".
- **Constraints that matter.** What must NOT change on the way there: "no other
  test file is modified", "public API unchanged".
- **An optional bound.** Append a clause like "or stop after 20 turns" so the
  loop is guaranteed to terminate. Claude reports progress against it each turn.
- **Hard limit: 4000 characters.** Over that, the string is invalid.

A weak condition is vague ("the code is better"), un-observable by the evaluator
("the database is migrated" with no surfaced proof), or unbounded.

## Phase 1: Parse input

Parse `$ARGUMENTS`:

1. Starts with `@` -> file path (content pre-loaded via the `@` syntax). Read it
   as the description of the desired end state.
2. Matches an issue pattern (`#123`, `123`, `owner/repo#123`, a GitHub issues
   URL) -> fetch with `gh issue view <n> --json title,body,comments,labels,url`
   to understand what "done" means.
3. Otherwise -> raw text describing the goal.
4. Empty -> ask the user for a one-line description of what they want Claude to
   keep working toward.

## Phase 2: Gather concrete checks (light)

The single biggest lever on goal quality is naming a **concrete, surfaceable
check**. Spend a moment finding the real commands in this repo so the condition
references things that actually exist:

- Detect the project type and its verify commands. Look for `Makefile` targets,
  `package.json` scripts, `go.mod`, `Cargo.toml`, CI config, etc.
- Note the canonical test command (`make test`, `go test ./...`, `npm test`),
  build command, and lint command.
- If a session or the input names a specific module/path, scope to it.

Use these to propose precise checks instead of vague ones. Keep this fast — a
couple of searches, not a full exploration.

## Phase 3: Mini-interview

This is a *mini* interview — lighter than `/ideate`. Target **one, at most two**
rounds of `AskUserQuestion` (3-4 questions each). Skip any question the input or
Phase 2 already answers; never ask the obvious.

Cover these goal-specific dimensions:

1. **End state** — What single observable outcome means "done"? Offer the
   concrete candidates you found (e.g. "all tests green", "file under N lines",
   "issue queue empty"). This is the spine of the condition.
2. **Proof / check** — How should Claude demonstrate it in the transcript? Offer
   the real commands from Phase 2 (e.g. "`make test` exits 0 and output shown").
3. **Constraints** — What must not change or break along the way? (public API,
   unrelated tests, file boundaries, no new deps.)
4. **Bound** — Turn/time cap before giving up. Offer sensible defaults (e.g.
   "stop after 20 turns", "stop after 30 minutes", "no bound"). Recommend
   including one.

If the goal touches money/payments, consensus-critical code, security paths, or
data migration, add one question confirming the user wants the loop to run
unattended over that surface, and bias toward a tighter turn bound.

After answers, self-assess: do I have a measurable end state, a surfaceable
check, the constraints, and a bound? If a critical piece is still vague, ask one
more targeted question. Otherwise proceed.

## Phase 4: Draft the goal string

Compose the condition. Prefer this structure (prose is fine too, but keep every
clause observable from the transcript):

```
<measurable end state>, verified by <concrete check that Claude runs and shows>.
Constraints: <what must not change>. Stop after <N turns / T minutes> if not met.
```

Rules:

- Write it so the **evaluator can decide yes/no from Claude's surfaced output
  alone**. If a clause can't be proven in the transcript, rewrite it as one that
  can, or drop it.
- Be specific: name the exact commands, paths, and numeric thresholds.
- Fold the bound clause in unless the user declined one.
- Keep it tight. Length budget is 4000 chars, but shorter and sharper evaluates
  more reliably — aim for the smallest string that fully pins the end state.

## Phase 5: Validate length and copy to clipboard

Measure the exact character count before presenting. To both measure and copy
the string to the macOS clipboard in one step, write the string to a scratch
file (avoids any shell-quoting issues with the literal string) and pipe it:

```bash
SCRATCH="$TMPDIR/goalcraft-goal.txt"   # or the session scratchpad dir
# ...write the candidate string to "$SCRATCH" first (e.g. via the Write tool)...
wc -m < "$SCRATCH"        # exact char count
pbcopy < "$SCRATCH"       # string now on the clipboard
```

- If **> 4000**: tighten — cut restated context, collapse redundant constraints,
  drop nice-to-haves, keep the measurable end state + check + bound. Re-measure.
  Never present (or copy) a string over the limit.
- Only run `pbcopy` once the string is final and within budget.
- If `pbcopy` is unavailable (non-macOS), skip the copy and say so.

## Phase 6: Present and offer to set

Output:

1. The final goal string in a fenced ```` ```text ```` block (copy-pasteable).
2. The character count, e.g. `1,240 / 4000 chars`.
3. A confirmation that the string is **on the clipboard** — so the user can type
   `/goal ` and paste (Cmd-V) to set it.
4. A one-line note on what the evaluator will look for each turn.

Then use `AskUserQuestion` to offer:

- **Refine** — take one more pass of feedback, regenerate, and re-copy.
- **Done** — leave it; the string is on the clipboard and printed above.

## Notes

- The string is for the user to paste into `/goal`. This command copies it to
  the clipboard but never sets the goal itself — a built-in slash command must
  be typed by the user.
- Favor checks that already land in the transcript naturally (test runs, build
  output, `git status`) over ones that require Claude to remember to print proof.
- A bounded goal is almost always better than an open-ended one — recommend a
  turn or time clause unless the user explicitly wants it to run until done.
