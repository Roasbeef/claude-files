---
name: advisor-review
description: "Final adversarial audit of finished work by one top-tier model (Fable 5 or Opus 4.8), in a single report-only pass across three axes: verify the load-bearing invariants (especially any you could NOT run locally), simplify without losing properties, and hunt for live variants of the bug shapes just fixed. You then verify each finding and act. Use as the closing gate on a substantive or risky change before calling it done, especially when part of it (a systest, a migration, a concurrency path) can't be validated locally. Distinct from /advisor (decision consult), /review-loop (multi-agent fix loop), and /variant-analysis (single-pattern sweep). Invoke via /advisor-review or after finishing hard-to-verify work."
argument-hint: "[<what to review: branch / commit-range / the change you just made>] [--tier=fable|opus]"
allowed-tools:
  - Agent
  - SendMessage
  - Bash
  - Read
  - Grep
  - Glob
  - AskUserQuestion
---

# Advisor Review

A single top-tier model (Fable 5 or Opus 4.8) does one focused, adversarial pass
over work you have just finished, and reports. You stay the author: you verify
each finding against the code and decide what to act on. This is the closing
gate on a substantive change, run *before* you tell the user it's done.

It composes three things a same-model self-review reliably misses, into one
review:

1. **Invariant verification** — the load-bearing correctness claims, ranked by
   severity, each with a concrete failure scenario. Weighted hardest at the
   parts you *could not run* (a systest with no local lane, a migration you
   can't exercise on both engines, a concurrency ordering you reasoned about but
   didn't observe). An independent top-tier reader catches a plausible-but-wrong
   test or a silent routing drop that you, having just written it, read past.
2. **Simplification** — where the change is more complex than the property it
   protects requires; and, for tests, where it is flaky or could false-pass.
3. **Same-shape variant hunt** — other *live* instances of the exact bug shapes
   you just fixed, staying near the blast radius. The fix you just made is the
   best query you'll ever have for the next one.

## When to reach for it

- You finished a substantive or money-path change and are about to call it done.
- Part of the change can't be validated locally (systest/integration lane,
  cross-engine migration, actor/concurrency ordering, a mailbox/wire contract).
- A reviewer (bot or human) found a real bug and you want to know whether the
  same shape lives elsewhere.
- Before opening or updating a PR on a mission-critical subsystem.

Reach for it even when this session's main loop is already Fable or Opus. The
`/advisor` rule — "if you're already top-tier, you *are* the advisor" — does
not carry over here: that skill's value is the tier jump, this one's is an
independent context. A fresh reader without the mental model that wrote the
code is what breaks author bias, whatever tier you're on.

Don't reach for it on a trivial or fully-unit-covered change — that's
over-reviewing. For a fan-out fix-until-clean loop use `/review-loop`; for a
pure decision/"what am I missing" use `/advisor`; for a deep single-pattern
sweep use `/variant-analysis`.

## Mechanics

Spawn **one** subagent, top tier. One strong reviewer with a tight prompt beats
a fan-out here — you want a coherent trace, not a vote.

- **Pick the tier.** Parse `--tier=fable|opus` from the arguments. If absent:
  `fable` when part of the change couldn't be run locally or it touches a money
  path; `opus` otherwise. Say which you picked in one line.
- **Commit first.** The reviewer reads the diff from git, so uncommitted work is
  invisible to it — it would silently audit a stale version of the change.
  Commit everything under review (fixups are fine) before spawning, or
  explicitly tell it to include the working tree and untracked files.
- **Pin the base.** A two-dot diff against a branch name (`main..HEAD`) pulls in
  unrelated upstream changes whenever the base has advanced, and the reviewer
  audits code you didn't write. Compute the base once with
  `git -C <dir> merge-base <base-branch> HEAD` and hand the reviewer that SHA.
- **Spawn it addressable.** Use `Agent` with `subagent_type: general-purpose`,
  `model: fable` (or `opus`), and `run_in_background: true` — the combination
  confirmed to return an `agentId` you can `SendMessage` later. You want that
  follow-up channel for the re-verify step below; a foreground spawn returned
  no `agentId` in testing. Inside a `Workflow`, use
  `agent(prompt, { model: 'fable', ... })` instead.
- **The review is the gate — wait for it.** Don't tell the user the work is
  done while the pass is in flight. While it runs, do only work the report
  can't invalidate, and hold the "done" call until the report is back and every
  finding is dispositioned.

Give the reviewer everything it needs to work without you: the worktree
path(s), the pinned diff range, and — critically — an honest list of **what
you could not run and why**. That's where it should dig.

### The prompt template

Fill the brackets. Keep the guardrail verbatim; it's what stops the pass from
turning into a rewrite.

> This is a REPORT-ONLY pass on work I just finished. Do NOT edit files, and do
> NOT propose scope expansion, big refactors, or new features. Keep every
> suggestion small and concrete (file:line + the change). "Nothing found" is a
> valid, expected answer — do not invent problems.
>
> ## Context
> [1–3 sentences: what the change does and why. The bug it fixes / feature it
> adds. Link the issue/PR.]
>
> Branch/worktree: [path], base [<base-sha>]. Get the diff with
> `git -C [path] diff [base]..HEAD`; read surrounding code as needed.
>
> ## Task 1 — verify the load-bearing invariants
> [List the 3–6 correctness claims this change rests on. For each, ask: does it
> actually hold? Give a concrete failure scenario (inputs/state → wrong outcome)
> if not.]
> I especially could NOT run [the systest / the postgres migration / the
> concurrency path] locally because [reason]. Trace it end to end and tell me
> whether it is correct AND whether it could silently false-pass or hang.
>
> ## Task 2 — simplify
> Look only at [the new/changed files]. Where is this more complex than the
> property requires? For any new test: is it flaky, or could it pass for the
> wrong reason? Suggest the minimal trim that keeps the property.
>
> ## Task 3 — same-shape variant hunt (report only)
> We fixed these bug shapes: [Shape A: one line]; [Shape B: one line]. Look
> NARROWLY, near this change's blast radius ([the packages/files it touches]),
> for OTHER live instances of the same shapes. If a shape turns up nothing, say
> "shape X: nothing found."
>
> Report every finding, from all three tasks, in this shape:
> **[severity] file:line — one-line claim**, then the concrete failure scenario
> (inputs/state → wrong outcome) and a confidence (likely real / possible /
> probably fine). Severity: HIGH = correctness or money loss on a live path;
> MEDIUM = latent, or needs an unlikely precondition; LOW = nit. Rank most
> severe first. Keep the report tight.

### Calibrating scope

- Trivial change → skip, or Task 1 only.
- Standard substantive change → all three axes, one reviewer.
- "Be thorough" / audit / money-path → all three axes, and if the variant hunt
  comes back rich, escalate that axis into a dedicated `/variant-analysis` or a
  `/review-loop` fan-out rather than cramming it into this pass.

## Acting on the report

You are the author. The reviewer reports; you verify and commit. Never apply a
finding blind.

1. **Verify each finding against the code yourself** before touching anything. A
   top-tier reviewer is usually right, but "usually" is why you check. Confirm
   the failure scenario actually reproduces in the code.
2. **Fix confirmed high-severity bugs now**, in scope, with a regression test
   that fails without the fix. Fold them into the change (fixups or a clear new
   commit).
3. **Assess mediums honestly.** Some dissolve under scrutiny (e.g. a "race" that
   the actor's single-turn ordering already closes) — when one does, *document
   the invariant* at the site so a future refactor re-checks it, and move on.
4. **Apply the in-scope simplifications** (drop a redundant assertion, comment a
   load-bearing line the reviewer flagged as silently critical). Skip the ones
   that trade real coverage for brevity.
5. **Latent / out-of-scope findings: do not fix them here.** Record them and
   offer to file a follow-up issue — creating a public issue is outward-facing,
   so confirm with the user first. Respect the "don't expand scope" guardrail
   you gave the reviewer; expanding it yourself defeats the point.
6. **The variant hunt often hands you the next fix.** If it found a live instance
   on the same path, that's in-scope hardening — fix it, with its own test,
   since a fresh fix is a fresh query.
7. **Re-verify through the same reviewer, not a cold re-run.** A finished
   background reviewer is idle, not gone — `SendMessage` its `agentId` the fix
   delta (the new commits plus one line per finding on what you did) and ask it
   to re-check just those slices and anything the fixes touched. It re-reads
   with its full trace intact, at a fraction of a fresh pass. Respawn cold only
   in a new session, where the old `agentId` is unreachable.

## Why one top-tier pass, not self-review

Self-review re-reads with the same mental model that wrote the code, so it re-
derives the same wrong assumption and skips the same untested branch. An
independent top-tier reader with an adversarial, invariant-first prompt breaks
that symmetry: it re-traces the delivery path you assumed, questions the test you
couldn't run, and — because you just told it exactly what shape of bug you
fixed — knows precisely what to look for elsewhere. The variant axis turns a
single fix into coverage; the simplify axis keeps the fix from ossifying; the
invariant axis is the part that actually catches money-path bugs.
