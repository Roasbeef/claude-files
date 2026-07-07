---
name: advisor
description: "On-demand consultation of a higher-tier model (Opus 4.8 or Fable 5) from a cheaper executor session. Emulates the native advisor-tool pattern in Claude Code: keep the main loop on a cheap model (Sonnet 5) and call the expensive model rarely, via a compact consultation packet, for a plan/decision/'what am I missing' — not for code. Use when the executor is stuck, before committing to a plan, after repeated failed attempts at the same fix, or before a risky/irreversible change. Invoked via /advisor or when the executor recognizes a consult checkpoint."
argument-hint: "[--tier=opus|fable] [<the specific question or decision you need help with>]"
allowed-tools:
  - Task
  - Read
  - Grep
  - Glob
  - Bash
  - AskUserQuestion
---

# Advisor

Emulate the [advisor-tool pattern](https://platform.claude.com/docs/en/agents-and-tools/tool-use/advisor-tool) inside Claude Code. The native tool pairs a cheap **executor** (the request model) with a higher-intelligence **advisor** (a `model` field on the tool) and bills most tokens at the executor rate. Claude Code has no such tool, so we get the same economics a different way: **the main loop stays cheap, and this skill reaches for the expensive model rarely, through a small sub-agent.**

The whole value proposition is *token asymmetry*. It only holds if two things are true:

1. **The main loop is the cheap tier** (Sonnet 5). If the session is already on Opus/Fable, every turn is paying advisor rates and this skill saves nothing — you are the advisor. Say so and stop.
2. **The consultation packet is small.** A sub-agent starts fresh and **cannot share the main loop's prompt cache** (caches are model-scoped), so the advisor pays cold context on every byte you send it. Dumping the transcript into a Fable advisor spends Fable rates on 100k tokens and destroys the arbitrage. Send ~a page, get back a paragraph.

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

- **`opus` (Opus 4.8, $5/$25)** — the default. Half of Fable's cost, still a large jump over a Sonnet executor. Right for most consults.
- **`fable` (Fable 5, $10/$50)** — Anthropic's most capable model. Reserve for the hardest, highest-stakes questions where Opus itself might be the thing you're unsure of.

If the choice is genuinely unclear and the stakes are high, ask with `AskUserQuestion`; otherwise pick and note it in one line.

### 2. Assemble the consultation packet

Keep it to roughly one page. Gather only what the advisor needs to reason — read the relevant files yourself first so you send *excerpts*, not "go read the repo":

- **Situation** — ≤10 lines: what the task is, what you've tried, where you're stuck.
- **The question** — one precise, answerable question or decision. Not "help."
- **Evidence** — the 2–3 code snippets that actually bear on the question, as `file:line` excerpts. Trim hard.
- **What you want back** — a plan, a decision with rationale, or a list of what you're missing. State it.

### 3. Spawn exactly one advisor sub-agent

Use the `Task` tool with the model override set to the chosen tier (`model: "opus"` or `model: "fable"`), `subagent_type: "Plan"` (read-only planner) for plan/decision questions, or `general-purpose` if it needs to look at more code. One call — this is a consult, not a fan-out. Prompt it as an advisor:

> You are a senior advisor consulted by a cheaper executor model that is doing the actual work. Below is a compact consultation packet. Return terse, high-leverage guidance — a plan, a decision with a one-line rationale, or the specific thing the executor is missing. **Do not write the implementation**; the executor will. Be direct, rank your points, and if the executor's framing is wrong, say so first.
>
> [packet]

If the harness's `Task` tool can't override the model per call, define a dedicated advisor sub-agent type pinned to Opus/Fable and target that instead.

### 4. Relay, don't dump

Summarize the advisor's guidance back into the session in a few lines — the decision and the reason, the plan steps, or the missing piece. Then act on it as the executor. Don't paste the advisor's full reply unless the user asks; the point is to convert expensive tokens into a cheap course-correction.

## Economics cheat-sheet

| Tier | $/1M in | $/1M out | Use as |
|---|---|---|---|
| Fable 5 | $10 | $50 | advisor (hardest calls) |
| Opus 4.8 | $5 | $25 | advisor (default), or a too-expensive main loop |
| Sonnet 5 | $3 | $15 ($2/$10 intro→2026-08-31) | the executor / main loop |
| Haiku 4.5 | $1 | $5 | trivial sub-tasks |

Native benchmark for the target you're emulating: Sonnet 5 executor + Fable 5 advisor reached ~92% of Fable-solo quality at ~63% of the cost on SWE-bench Pro, with the advisor called ~once per task. The skill only reproduces that if you honor the two invariants at the top: cheap main loop, small packet, rare calls.

## Related

- For the inverse pattern — an expensive planner fanning out to cheap workers — see `/orchestrate`.
