---
name: review-loop
description: "Adversarial review→triage→fix loop until a cold verifier signs off. Fans out lens-specific reviewer subagents, verifies every finding against the code (killing false positives), auto-applies confirmed fixes as fixup commits, and repeats until a fresh verifier approves. Prefers a deterministic dynamic workflow when available; falls back to in-instance Task dispatch. Use when the user types /review-loop or asks to adversarially review-and-fix a change set, branch, or commit range until clean."
argument-hint: "[<commit-range> | <branch> | (default: branch vs base / uncommitted)] [--base=<branch>] [--max-iters=<N>] [--cutoff=high|medium|low] [--workflow|--inline]"
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

# Review Loop

Run an adversarial review→triage→fix loop until a fresh cold verifier signs off.
Unlike `/code-review` (report-only) this loop **verifies** every finding against
the code, **kills false positives**, **applies** the confirmed fixes as fixup
commits, and **repeats until acceptance**. All reviewing happens in subagents,
so each reviewer burns its own context, not yours.

**Target**: $ARGUMENTS

## Why this shape (and why a workflow)

This loop exists to defeat three failure modes that hit a single context window
on long, adversarial tasks:

- **Agentic laziness** — declaring a review done after partial coverage. The
  loop's fixed phases and convergence check force full coverage.
- **Self-preferential bias** — grading your own findings. The triage judge and
  the final verifier are *separate* agents that never saw your reasoning.
- **Goal drift** — losing the original constraints across turns. A design brief
  is passed verbatim to every agent.

Because those guarantees depend on the orchestration actually running every
phase every time, the **preferred execution path is a dynamic workflow** (a
deterministic JavaScript harness), not model-driven dispatch. The workflow
encodes fan-out, triage, apply, loop, and verify as code that cannot drift or
cut corners. The in-instance path below is the fallback when the `Workflow`
tool is unavailable or the user passes `--inline`.

## Phase 0: Scope, baseline, and design brief (always done by the main loop)

Do this in the first turn, before any dispatch, regardless of execution path.

1. **Resolve scope** into one concrete diff command and a stable description:
   ```bash
   git branch --show-current
   git show-ref --verify --quiet refs/heads/main && echo main || echo master
   # Range given? use it. Branch given? <base>...<branch>.
   # Else commits ahead of base? <base>..HEAD. Else uncommitted: git diff HEAD.
   git diff <range> --stat ; git log <range> --oneline
   ```
   Every finder and the verifier must review the **same** surface — record the
   exact diff command.

2. **Capture a pre-flight baseline** so pre-existing breakage is not blamed on
   the change:
   ```bash
   make build 2>&1 | tail -5 ; make test 2>&1 | tail -5 ; make lint 2>&1 | tail -5
   ```
   Note what was already red (e.g. a toolchain/lint-config issue) for the brief.

3. **Write a design brief** to `.review-loop/brief.md` — this is what makes
   triage accurate. Include: what the change does and **why** (approved intent);
   hard constraints and environment/protocol semantics reviewers can't infer
   from the diff; accepted tradeoffs and out-of-scope items; the pre-flight
   baseline.

4. **Pick lenses from the changed files** and record them in
   `.review-loop/lenses.md`. Always run the baseline adversarial panel; add
   specialized lenses when trigger files are present:

   | Lens | subagent_type | Trigger |
   |---|---|---|
   | Correctness | `code-reviewer` | always |
   | Offensive security | `security-auditor` | always |
   | Differential / blast radius | `general-purpose` + `differential-review` skill | always |
   | Concurrency | `general-purpose` | goroutines, channels, mutexes, `sync`, atomics |
   | Shell / config hardening | `general-purpose` | `*.sh`, Dockerfiles, CI YAML, hooks, settings |
   | API safety & insecure defaults | `general-purpose` + `sharp-edges`/`insecure-defaults` | public interfaces, config, RPC/proto |
   | Deep function analysis | `audit-context-building:function-analyzer` | crypto/auth, consensus, value-transfer |
   | Spec compliance | `spec-to-code-compliance:spec-compliance-checker` | BIP/BOLT/protocol/spec references |

5. `mkdir -p .review-loop` and track the run with TodoWrite (one item per phase,
   plus a per-round entry as the loop iterates).

## Preferred path: dynamic workflow

When the `Workflow` tool is available and `--inline` was not passed, run the
loop as a deterministic harness. The bundled script
`workflow/review-loop.js` is a **template** — adapt it to the run (the chosen
lens set, cutoff, and max-iters), do not assume it must run verbatim.

> **Template pitfall (read before editing the script):** `meta` must be a
> **pure literal**. No string concatenation, no template interpolation, no
> variables in any field. The Workflow tool rejects anything else with
> `meta must be a pure literal`, which breaks every run. If you adapt the
> template, keep `description`/`whenToUse` single string literals.

Invoke it via the `Workflow` tool, passing the Phase 0 artifacts as `args`:

```
Workflow({
  scriptPath: "<this skill dir>/workflow/review-loop.js",
  args: {
    diffCmd:   "<exact diff command>",
    base:      "<base branch>",
    brief:     "<contents of .review-loop/brief.md>",
    lenses:    [ /* the selected lens descriptors */ ],
    cutoff:    "medium",
    maxIters:  5,
  },
})
```

The workflow runs find→triage→apply→loop→verify and returns a structured
summary (rounds, confirmed vs rejected per round, applied fixups, deferred
follow-ups, verifier verdict). When it returns, the main loop does Phase 6
(finalize) below — autosquash offer and final green build — because those steps
are interactive and side-effectful.

If the workflow hits `maxIters` without converging, it returns what remains
rather than looping forever; surface that and ask how to proceed.

## Fallback path: in-instance Task dispatch (`--inline` or no Workflow tool)

Run the same phases with the Task tool. This is what we ran by hand; it works
but relies on the orchestrator faithfully executing each phase.

### Phase 1 — dispatch finders
Launch every selected lens in **one message** with parallel Task calls (or
`run_in_background: true` and collect notifications). Give each the same diff
surface and the design brief, with this adversarial skeleton:

```
You are an ADVERSARIAL reviewer. BREAK this change, do not grade it. Only
report findings you can argue concretely from the code.
Scope (review exactly this): <diff command>
Design brief: <.review-loop/brief.md>
Your lens: <lens + specific failure modes to hunt>
For each finding return: stable id, file:line, severity
(critical/high/medium/low/info), a concrete trigger SCENARIO, and a minimal fix
sketch. A verified "not a bug" is useful signal. Raw list, no pleasantries.
```
Write outputs to `.review-loop/round-<N>/find-<lens>.md`.

### Phase 2 — triage (never skip)
Spawn ONE `general-purpose` judge with all finder outputs + the brief + code
read access. It must **verify** each finding against the cited lines (reject
what it can't reproduce), **dedup/merge**, **kill false positives with reasons**,
and classify survivors into **fix-now** (≥ cutoff; with a repo-style fix
sketch), **follow-up** (deferrable; with an issue title), **rejected** (why).
Write to `.review-loop/round-<N>/triage.md`. If a fix-now item contradicts the
approved design, surface via `AskUserQuestion` before fixing.

### Phase 3 — apply
For each fix-now finding in severity order: implement the minimal fix matching
surrounding code; add/update tests when testable; build + relevant package
tests must pass vs the Phase 0 baseline; commit as a fixup:
```bash
git add <files> ; git commit --fixup=<target-sha>
```
Use `hunk stage` for files mixing fix-now and deferred changes. Log to
`.review-loop/round-<N>/applied.md`.

### Phase 4 — loop
Increment the round, re-run Phase 1 finders on the **new** diff. New
triage-confirmed fix-now findings → back to Phase 2/3. A clean round (zero new
fix-now) → Phase 5. Stop at `--max-iters` and report what remains.

### Phase 5 — cold acceptance verifier
Spawn ONE fresh `code-reviewer` that saw no prior round. Give it only the brief
and the full final diff (`<base>..HEAD`) and ask: APPROVE, or RE-OPEN with
concrete findings. APPROVE → Phase 6. RE-OPEN → feed findings into Phase 2
(subject to the same triage discipline).

## Phase 6: Finalize (always done by the main loop)

1. **Offer autosquash** of the fixups into their originals:
   ```bash
   hunk rebase autosquash --onto <base> --dry-run
   ```
   Show the plan; on approval run for real. If fixups interleave with other
   commits on the same lines (conflict risk), instead offer a single `review:`
   commit via soft reset. Declined → leave fixups as-is.
2. **Final verification**: build + full tests + lint, green vs the Phase 0
   baseline.
3. **Summary** (concise, to chat): rounds run, confirmed vs rejected per round,
   fixes applied (with commits), the deferred follow-up list (suggest opening
   issues), and the verifier verdict.

## Notes

- **In-instance by design.** Finders, triage, and verifier are subagents, so the
  heavy reading lives in their context, not yours. The Substrate path
  (`/s-code-review`) is the alternative when you want findings tracked in the
  review system / web UI; this trades that for lower context cost.
- **Never skip triage.** Raw adversarial finders produce plausible-but-wrong
  findings; verify-and-reject is what makes auto-apply safe.
- **Cutoff discipline.** Fix C/H/M in-loop; defer L/I to keep the loop
  converging and the diff focused. Surface deferred items, don't drop them.
