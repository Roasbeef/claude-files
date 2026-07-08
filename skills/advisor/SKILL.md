---
name: advisor
description: "On-demand consultation of a higher-tier model (Opus 4.8 or Fable 5) from a cheaper executor session. Emulates the native advisor-tool pattern in Claude Code: keep the main loop on a cheap model (Sonnet 5) and call the expensive model rarely, via a focused consultation packet, for a plan/decision/'what am I missing' — not for code. Use when the executor is stuck, before committing to a plan, after repeated failed attempts at the same fix, or before a risky/irreversible change. Invoked via /advisor or when the executor recognizes a consult checkpoint."
argument-hint: "[--tier=opus|fable] [<the specific question or decision you need help with>]"
allowed-tools:
  - Task
  - SendMessage
  - Read
  - Grep
  - Glob
  - Bash
  - AskUserQuestion
---

# Advisor

Emulate the [advisor-tool pattern](https://platform.claude.com/docs/en/agents-and-tools/tool-use/advisor-tool) inside Claude Code. The native tool pairs a cheap **executor** (the request model) with a higher-intelligence **advisor** (a `model` field on the tool) and bills most tokens at the executor rate. Claude Code has no such tool, so we get the same economics a different way: **the main loop stays cheap, and this skill reaches for the expensive model rarely, through a sub-agent.**

The whole value proposition is *token asymmetry*. It only holds if two things are true:

1. **The main loop is the cheap tier** (Sonnet 5). If the session is already on Opus/Fable, every turn is paying advisor rates and this skill saves nothing — you are the advisor. Say so and stop.
2. **The packet is relevant, not raw.** A sub-agent starts fresh and **cannot share the main loop's prompt cache** (caches are model-scoped), so it pays cold context on everything you send. That argues against pasting the *whole transcript* or unrelated files — not against detail. A thin packet earns thin, generic advice; the discipline is to cut what doesn't bear on the question, then be thorough with what does.

## When to consult (the trigger discipline)

The native tool gets called ~once per task because the executor decides when. Encode that judgment — consult at these checkpoints, and otherwise *don't*:

- **Before committing to a plan** for a non-trivial task — get the shape reviewed before you spend executor tokens building the wrong thing.
- **After 2 failed attempts** at the same fix — you're likely missing something the cheap model can't see; ask "what am I missing?"
- **Before a risky or irreversible change** — a schema migration, a public-API change, a delete, anything with wide blast radius.
- **At a genuine fork** — two defensible approaches and the choice is load-bearing.

Do **not** consult for: things you can verify by reading the code, routine edits, style questions, or anything where you'd accept your own first answer. Over-consulting is the failure mode — it turns the cheap session expensive one call at a time. Aim for ~once per task.

## Procedure

### 1. Pick the tier

Parse `--tier=opus|fable` from arguments. If absent, choose by difficulty and say which you picked:

- **`opus` (Opus 4.8, \$5/\$25 per 1M in/out)** — the default. Half of Fable's cost, still a large jump over a Sonnet executor. Right for most consults.
- **`fable` (Fable 5, \$10/\$50)** — Anthropic's most capable model. Reserve for the hardest, highest-stakes questions where Opus itself might be the thing you're unsure of.

If the choice is genuinely unclear and the stakes are high, ask with `AskUserQuestion`; otherwise pick and note it in one line.

### 2. Assemble the consultation packet

Read the relevant files yourself first so you can hand over real excerpts rather than "go read the repo," then include everything that bears on the question:

- **Situation** — the full picture: what the task is, what you've tried, exactly how each attempt failed, and the constraints and invariants that matter.
- **The question** — one precise, answerable question or decision. Not "help."
- **Evidence** — every code snippet that bears on the question, as `file:line` excerpts, with enough surrounding context for the advisor to actually reason — not just the one changed line.
- **What you want back** — a plan, a decision with rationale, or a list of what you're missing. State it.

(The relevance discipline is invariant #2 above: cut irrelevant bulk, keep relevant detail — no size ceiling.)

### 3. Spawn the advisor sub-agent

Use the `Task` tool with `model` set to the chosen tier (`model: "opus"` or `model: "fable"`). Default to `subagent_type: "general-purpose"` spawned with `run_in_background: true` — this is the combination confirmed to return an addressable `agentId`, which you need if there's any chance of a follow-up. Reserve `subagent_type: "Plan"` (read-only planner) or a plain foreground spawn for a consult you're sure will be one-shot; a foreground `Plan` spawn returned no `agentId` in testing, and `SendMessage` to its description string failed outright with "not reachable." One call — this is a consult, not a fan-out. Prompt it as an advisor:

> You are a senior advisor consulted by a cheaper executor model that is doing the actual work. Below is a consultation packet. Return terse, high-leverage guidance — a plan, a decision with a one-line rationale, or the specific thing the executor is missing. **Do not write the implementation**; the executor will. Be direct, rank your points, and if the executor's framing is wrong, say so first.
>
> [packet]

**Follow-up consults on the same task.** Default to one-shot — most tasks consult once. For a task with several checkpoints (plan review, then a failed attempt, then a risky change), keep one advisor alive: send only the *delta* on each follow-up via `SendMessage` to its `agentId`.

**Completion doesn't end it, within this session.** A finished advisor is idle, not gone — `SendMessage` to its `agentId` resumes it from its transcript with everything it already knows, and you can go back and forth as many times as the task needs. That persistence doesn't survive a full session restart, though — a new session can't reach an old `agentId`, so respawn cold with a one-paragraph re-summary if you're picking the task back up later. Keep it honest anyway: the retained history is re-billed at advisor rates every turn, so let it stay lean and don't carry one across unrelated tasks.

A message to a still-working advisor lands on its next tool round rather than instantly; if it finishes first, the delta triggers an automatic resume, and the advisor may `SendMessage(to: "main")` back on its own once it's processed it — no polling required. See `docs/advisor-and-orchestrate.md` for the full mechanics.

When you're done, just stop messaging it — there's no explicit teardown for an idle advisor. (`TaskStop` on its `agentId` only finds it while a task is actively in flight; once idle it reports "no task found.") An abandoned advisor simply sits dormant, costing nothing further, until the session ends. If `SendMessage` ever finds it truly gone, spawn a fresh one with a one-paragraph re-summary.

### 4. Relay, don't dump

Summarize the advisor's guidance back into the session in a few lines — the decision and the reason, the plan steps, or the missing piece. Then act on it as the executor. Don't paste the advisor's full reply unless the user asks. The advisor is *advisory*: you own the decision, and you discard an answer that's generic or that tries to write the code for you.

## Related

- Economics, price gradient, and the native benchmarks: `docs/advisor-and-orchestrate.md`.
- The inverse pattern — an expensive planner fanning out to cheap workers — is `/orchestrate`.
