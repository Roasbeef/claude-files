---
name: ci-loop
description: "Babysit CI after a push: a monitor→classify→remediate state machine that watches a run to completion, mechanically fixes trivial/build breakage, reproduces test failures locally to tell a flake from a real bug, reruns suspected flakes within a budget, hands reproduced failures to a fixer that commits a fixup and pushes, and loops until CI is green or it can justify a bail back to the user. Prefers a deterministic dynamic workflow when available; falls back to in-instance Task dispatch. Use when the user types /ci-loop or asks to watch/babysit/auto-fix CI for a pushed branch or PR until it's green."
argument-hint: "[<pr-number> | <branch> | (default: current branch)] [--base=<branch>] [--max-iters=<N>] [--flake-retries=<N>] [--profile=lite|standard|thorough] [--allow-push-to-default] [--workflow|--inline]"
disable-model-invocation: true
allowed-tools:
  - Task
  - Workflow
  - Bash
  - Read
  - Write
  - Edit
  - MultiEdit
  - Grep
  - Glob
  - TodoWrite
  - AskUserQuestion
---

# CI Loop

Babysit CI after you push. This is a **state machine**: watch the latest run to
completion, classify each failure, and take the smallest remediation that fits —
mechanically fix lint/build breakage, reproduce a failing test locally to tell a
flake from a real bug, rerun suspected flakes within a budget, or hand a
reproduced failure to a fixer agent that commits a fixup and pushes. After any
push it re-monitors the fresh run, and it ends only when **CI is green** or it
can give the user a **justified bail**. All the watching, reading, and fixing
happens in subagents, so each one burns its own context, not yours.

**Target**: $ARGUMENTS

## Why this shape (and why a state machine)

A red CI run on a branch you just pushed is the classic babysitting chore: most
failures are mechanical (a `gofmt`, a stale generated file, a lint nit) or
transient (one flaky integration test), and a few are real bugs. Doing this by
hand means tabbing back to the run page every few minutes for half an hour. This
loop automates exactly that vigil, and it is a state machine on purpose — the
states encode the judgments that are easy to get wrong:

- **Flake vs real bug is a decision, not a guess.** The loop never declares a
  failure a flake without *positive local evidence* that the test passes. The
  Reproduce state runs the test locally N times; only a clean local result (plus
  no plausible cause in the diff) earns a "flake" verdict, and even then it just
  *reruns* CI rather than ignoring the failure. When in doubt it returns
  "reproduced" and escalates to the fixer. A wrong flake call hides a real bug.
- **Trivial fixes and real fixes are different risk classes.** Lint/build
  repairs are deterministic and cheap; a code change that has to make a failing
  test pass is where a careless fix does damage. They are separate states with
  separate prompts, and the fixer is forbidden from weakening assertions or
  skipping tests to go green.
- **Bailing is a first-class outcome.** The point isn't to thrash forever. When
  the fixer can't produce a verified fix, when a flake won't stop flaking past
  its budget, or when the failure is non-actionable infra, the loop stops and
  reports *why* — it does not keep pushing speculative commits.

Because those guarantees depend on the orchestration actually running every
state in order every cycle, the **preferred execution path is a dynamic
workflow** (a deterministic JavaScript harness), not model-driven dispatch. The
workflow encodes monitor→classify→fix/reproduce/repair→loop as code that cannot
drift or skip the reproduce-before-flake check. The in-instance path below is the
fallback when the `Workflow` tool is unavailable or the user passes `--inline`.

### The states are mandatory — do not author a "just rerun it" variant

This skill runs **Monitor → Classify → (Fix | Reproduce → Repair) → loop**,
every cycle. You may adapt the failure taxonomy, the flake budget, the profile,
and the local-reproduction method. You may NOT:

- **Declare a flake without the Reproduce state.** "The test passed on rerun"
  is not evidence by itself; reproduce locally first. Blind-rerunning a real,
  deterministic failure burns CI minutes and never converges.
- **Skip Classify and fix everything the same way.** Pushing a code change to
  "fix" a broken runner or a registry outage is noise; infra failures get rerun
  or reported, not patched.
- **Make the loop unbounded.** Every path leads to green, a bail with a reason,
  or the max-iters cap. A babysitter that never reports back is worse than none.

## Phase 0: Scope, push, and brief (always done by the main loop)

Do this in the first turn, before launching the workflow.

1. **Ensure the branch is pushed and identify the target.** The loop watches CI
   for a commit that exists on the remote, so push first if needed:
   ```bash
   git branch --show-current
   git status -sb            # anything unpushed?
   git push                  # if the branch isn't on the remote / is behind
   gh pr view --json number,headRefName,url 2>/dev/null   # is there a PR?
   gh run list --branch "$(git branch --show-current)" --limit 3 \
     --json databaseId,headSha,status,conclusion,workflowName
   ```
   Record the PR number (if any) and the branch — the loop watches *the most
   recent run for the head commit*.

2. **Confirm the push target.** This loop **pushes fixup commits on its own** to
   trigger fresh CI runs. By default it pushes only to the feature branch and
   **refuses to push to `main`/`master`**. If HEAD is on a default branch, stop
   and confirm with the user before running (or pass `--allow-push-to-default`
   only when they explicitly want that).

3. **Write a short brief** to `.ci-loop/brief.md`: what the change does, any
   context the fixers can't infer from the diff (env quirks, why a test exists,
   known-flaky suites, accepted tradeoffs). This is what keeps a fixer from
   "fixing" intended behavior. The brief is re-Read from disk each phase, so a
   mid-run edit is honored on the next cycle.

4. `mkdir -p .ci-loop` and track the run with TodoWrite.

## Preferred path: dynamic workflow

When the `Workflow` tool is available and `--inline` was not passed, run the
loop as a deterministic harness. The bundled script `workflow/ci-loop.js` is a
**template** — adapt the *parameters* to the run (the PR/branch, base, profile,
flake budget), preferably by passing `args` rather than rewriting the script.

> **Template pitfall (read before editing the script):** `meta` must be a
> **pure literal**. No string concatenation, no template interpolation, no
> variables in any field. The Workflow tool rejects anything else with
> `meta must be a pure literal`, which breaks every run.

Invoke it via the `Workflow` tool, passing the Phase 0 artifacts as `args`:

```
Workflow({
  scriptPath: "<this skill dir>/workflow/ci-loop.js",
  args: {
    pr:           123,              // PR number, or omit and use branch
    branch:       "<feature branch>",
    base:         "<base branch>",  // for fixup targeting; default main
    brief:        "<contents of .ci-loop/brief.md>",
    briefPath:    ".ci-loop/brief.md",
    profile:      "standard",       // lite | standard | thorough
    flakeRetries: 2,                // per-job rerun budget before escalating
    infraRetries: 2,                // per-run rerun budget for infra failures
    maxIters:     10,               // optional; overrides the profile cap
    allowPushToDefault: false,      // never push to main unless true
  },
})
```

**Profiles** are the patience dial (`--profile`, default `standard`):

- **lite** — up to 5 monitor cycles, 3 local reproduction runs per test. A quick
  pass for a small change you expect to go green fast.
- **standard** — up to 10 cycles, 5 reproduction runs.
- **thorough** — up to 20 cycles, 10 reproduction runs. For a long, integration-
  heavy suite where flakes need more runs to rule out and you want the loop to
  keep babysitting across many CI cycles.

**Model tiering.** The bounded/mechanical states run on a cheaper tier: the
monitor (watch + report), the classifier (read logs, bucket), the reproduce
orchestration (run a test, tally), and reruns all run on **Sonnet** — that is the
floor. The two quality-critical states — the **trivial fixer** and the **test
fixer** — inherit the strong main-loop model, because a wrong code fix pushed to
CI is the expensive mistake. Reproduce is told to **bias toward "reproduced"
when unsure**, so the cheaper judge never silently dismisses a real bug as a
flake.

### The state machine the workflow runs

Each cycle is one CI observation plus one remediation step; after any push it
re-monitors the fresh run (a push re-evaluates everything, so the loop handles
one failure class per cycle and lets the next run sort out the rest):

```
            ┌────────────► Monitor (watch latest run to completion)
            │                  │
            │          green ──┴── failed
            │            │          │
            │          DONE      Classify (lint/build · test · infra)
            │                       │
            │      ┌────────────────┼──────────────────────┐
            │   trivial/build      test                   infra
            │      │                 │                       │
            │    Fix              Reproduce (run locally)   rerun?
            │   (fixup+push)        │                       ├─ budget ► rerun ─┐
            └──── push ◄──┐     ┌───┴────┐                  └─ exhausted ► BAIL │
                         │   flake     reproduced /              (infra)        │
                  ┌──────┘     │       cannot-run                               │
              cannot-fix    rerun?      │                                       │
                 BAIL       ├ budget►rerun ─► (re-monitor) ◄────────────────────┘
                            └ exhausted ► escalate as real
                                            │
                                          Repair (fixer: fix + verify locally + fixup + push)
                                            ├─ fixed ► push ► (re-monitor)
                                            └─ cannot-fix ► BAIL (report to user)
```

Terminal states the workflow returns in `finalState`:

- **green** — CI passed. Done.
- **bailed-cannot-fix** — the trivial or test fixer could not produce a verified
  fix; `bail.reason` carries the user-facing explanation.
- **bailed-infra** — a non-actionable infra failure persisted past `infraRetries`.
- **stuck** — the run is red but nothing actionable could be classified, the
  reproduce step returned no verdict, or a rerun couldn't be triggered.
- **no-ci** — no CI run was found for the head commit after the monitor retried
  (e.g. the push triggered no workflow). Check that CI is wired up for the branch.
- **monitor-error** — the monitor agent failed to return a result at all.
- **maxIters** / **budget** — the safety caps tripped before converging.

It returns a structured summary (`finalState`, `green`, `profile`, `iterations`,
`fixesApplied`, `flakesRerun`, `bail`, `history`, `tokensSpent`). The ASCII
diagram above shows the happy and bail paths; the safety-cap exits (`maxIters`,
`budget`, `stuck`, `no-ci`, `monitor-error`) are omitted there for clarity but
are all reachable. When it returns, the main loop
does Phase 6 (finalize) — autosquash and the final report — because those steps
are interactive.

Two behaviors worth knowing when you read the result:

- **Flake escalation.** A job the reproduce agent calls a "flake" is rerun, but
  only `flakeRetries` times. Past that budget the loop stops trusting the flake
  verdict and escalates the job to the fixer as a real failure — a "flake" that
  never stops flaking is treated as a bug, not ignored forever.
- **Reproduce is conservative by design.** "reproduced" and "cannot-run" both
  go to the fixer (cannot-run because we can't prove it's a flake). Only a clean
  local result with no diff-side cause earns a rerun-and-move-on.

Report `finalState` and `bail.reason` prominently. A bail is a real result to
hand back to the user, not a failure of the loop — surface it and ask how to
proceed rather than burning more cycles.

## Fallback path: in-instance Task dispatch (`--inline` or no Workflow tool)

Run the same state machine with the Task tool. Track the cycle with TodoWrite.

### Monitor
Find the latest run for the head commit (`gh run list --branch <b> --json
databaseId,headSha,status,conclusion`), then watch it
(`gh run watch <id> --interval 30 --exit-status`, or short polling if your shell
caps command time). Green → done. Failed → collect failing jobs
(`gh run view <id> --json conclusion,jobs`).

### Classify
Spawn one `general-purpose` agent over `gh run view <id> --log-failed`. Bucket
each failed job: **lint** (format/static/generated), **build** (compile),
**test** (assertion — list failing test ids), **infra** (runner/network/dep
outage), **other**. Be precise about test-assertion vs environment.

### Fix (lint/build/other)
One fixer: reproduce the check locally, apply the minimal fix, confirm the local
check passes, `git commit --fixup=<sha>`, and **push to the feature branch only**
(never `main`/`master` unless `--allow-push-to-default`). Re-monitor the new run.

### Reproduce (test failures)
One agent runs each failing test locally in isolation N times (`go test -run
'^T$' -count=N -race ./pkg/...`). Verdict per test: **reproduced** (failed
locally — real), **flake** (passed every local run *and* no diff-side cause),
**cannot-run** (CI-only env). **When unsure, return reproduced, never flake.**

### Repair (real failures) / rerun (flakes)
- Reproduced / cannot-run → spawn a fixer: root-cause it (do NOT weaken or skip
  the test), fix, verify locally until green, `git commit --fixup=<sha>`, push.
  Re-monitor. If it can't produce a verified fix → **bail with a reason**.
- Pure flakes within budget → `gh run rerun <id> --failed`, re-monitor. Past
  the flake budget → escalate to the fixer as real.
- Infra → `gh run rerun <id> --failed` within `infraRetries`, else **bail**
  (non-actionable).

Stop at `--max-iters` and report what remains.

## Phase 6: Finalize (always done by the main loop)

1. **On green** — offer to autosquash the fixups into their originals:
   ```bash
   hunk rebase autosquash --onto <base> --dry-run
   ```
   Show the plan; on approval run it and force-with-lease push the cleaned
   history. Declined → leave the fixups as-is.
2. **On a bail** — report `finalState` and `bail.reason` plainly, with what the
   fixer tried, and ask the user how to proceed. Do not dress a bail up as a
   pass.
3. **Summary** (concise, to chat): cycles run, fixes applied (with commits),
   flakes rerun, and the final CI state with a link to the run.

## Notes

- **It pushes — that's the point.** This loop pushes fixups to your feature
  branch autonomously to trigger fresh CI runs. It refuses to push to a default
  branch unless `--allow-push-to-default`. If you don't want autonomous pushes,
  use `/review-loop` (which fixes locally and never pushes) instead.
- **GitHub Actions via `gh`.** The template drives `gh run` / `gh pr checks`. For
  another CI provider, adapt the monitor/classify/rerun commands in the agents;
  the state machine is provider-agnostic.
- **Never dismiss a failure as a flake without local evidence.** This is the one
  judgment the loop is built to get right. The reproduce-before-flake rule is
  load-bearing; do not let an `--inline` run shortcut it.
- **A bail is a clean handoff, not a stop hook to satisfy.** When the fixer
  can't fix it, the most useful thing is a precise report of the failure and
  what was tried — not another speculative commit.
- **Complements `/review-loop`.** Review-loop hardens the diff *before* you push
  (adversarial review → fix locally); ci-loop babysits it *after* (watch CI →
  fix → push). Run review-loop first, then ci-loop on the pushed branch.
