---
name: orchestrate
description: "Expensive planner (Opus 4.8 / Fable 5) decomposes a task into independent work items, a fleet of cheap Sonnet 5 (or Haiku 4.5) workers executes them in parallel via a Workflow, and the planner synthesizes the results. Emulates the native orchestrator pattern in Claude Code: keep the expensive model scarce (one planning pass, one synthesis pass) and the cheap model abundant (N parallel workers). Prefers a deterministic dynamic workflow when available; falls back to in-instance Task dispatch. Use when the user types /orchestrate or asks to decompose, fan out, or parallelize a task across independent workers."
argument-hint: "[--planner=opus|fable] [--worker=sonnet|haiku] [--isolation] <task to decompose and run>"
disable-model-invocation: true
allowed-tools:
  - Task
  - Workflow
  - Read
  - Grep
  - Glob
  - Bash
  - AskUserQuestion
---

# Orchestrate

Emulate the **orchestrator pattern** from the [advisor-tool docs](https://platform.claude.com/docs/en/agents-and-tools/tool-use/advisor-tool): a high-intelligence model *plans and decomposes*, then delegates the token-heavy execution to a fleet of cheap workers. Most tokens are generated at the worker rate. On BrowseComp the native version hit ~96% of Fable-solo quality at ~46% of the cost.

**Target**: $ARGUMENTS

This is the inverse of `/advisor` (cheap main loop calling an expensive advisor). Here the *expensive* reasoning is scarce — one planning pass, one synthesis pass — and the *cheap* work is abundant (N parallel workers).

## Why this shape (and why a workflow)

The whole savings depend on the expensive stages actually staying thin — a planner that starts reading files, or a synthesizer that re-verifies every worker's output line by line, has silently become the expensive main loop again. A **deterministic JavaScript harness** (the `Workflow` tool) enforces that split as code: the plan phase gets a `model` override to the expensive tier and nothing else to do but decompose; the execute phase runs on the cheap tier by construction; there is no path for the harness to "decide" to skip the split the way a model-driven dispatch might drift under pressure. The in-instance path below is the fallback when the `Workflow` tool is unavailable.

### Keep the expensive stages thin — do not let Plan or Synthesize do the workers' job

You may adjust the item granularity, the tier choice, and the schemas. You may NOT:

- **Let the planner read the codebase to build the plan.** If decomposing requires deep reading, scout the shape yourself first (Phase 0) and hand the planner a known work-list, or push the reading into a worker whose job is "read X, then report the shape of Y."
- **Let the synthesizer re-do worker work.** Synthesis stitches and flags conflicts/gaps; it does not re-derive results a worker already produced.
- **Skip synthesis and dump raw worker output.** The point of the expensive final pass is one coherent answer, not N disconnected fragments.

## Phase 0: Tiers, scope, and work-list (always done by the main loop)

Do this before authoring or invoking the workflow.

1. **Parse tiers and flags** from the arguments:
   - `--planner=` — `opus` (Opus 4.8, \$5/\$25 per 1M in/out; default) or `fable` (Fable 5, \$10/\$50; reserve for genuinely hard decomposition where plan quality is the bottleneck).
   - `--worker=` — `sonnet` (Sonnet 5, \$3/\$15; default) or `haiku` (Haiku 4.5, \$1/\$5; only for mechanical, well-specified sub-tasks).
   - `--isolation` — pass if workers will mutate files in parallel and would collide; skip otherwise (worktrees are expensive).

2. **Scout inline first (cheap).** Before invoking the workflow, do the light discovery yourself in this session — list the files, find the call sites, scope the diff — so the plan phase operates on a known work-list instead of guessing the shape. If the work-list is already obvious from this scouting (e.g. "one item per file" over a known file set), skip the planner's decomposition entirely and pass `items` directly (see below) — that's an expensive planning pass you don't need to pay for.

## Preferred path: dynamic workflow

When the `Workflow` tool is available, run the bundled template at `workflow/orchestrate.js` rather than hand-authoring a script per invocation:

```
Workflow({
  scriptPath: "<this skill dir>/workflow/orchestrate.js",
  args: {
    task:      "<the task, verbatim>",
    planner:   "opus",     // or "fable"
    worker:    "sonnet",   // or "haiku"
    isolation: false,      // true if workers collide on files
    items:     null,       // or [{ id, spec }, ...] to skip the planner and go straight to Execute
  },
})
```

> **Template pitfall:** `meta` must be a **pure literal** — no string concatenation, no template interpolation, no variables in any field. The Workflow tool rejects anything else with `meta must be a pure literal`.

The template runs **Plan → Execute → Synthesize**:

- **Plan** (expensive tier) — skipped entirely if `args.items` is supplied; otherwise the planner decomposes `args.task` into a compact list of self-contained work items.
- **Execute** (cheap tier, parallel) — one worker per item, each returning a structured result.
- **Synthesize** (expensive tier) — stitches the worker results into one coherent answer, flagging conflicts and gaps.

It returns `{ items, results, answer }`. Relay `answer` to the user; the per-item `results` are background detail, not the report.

## Fallback path: in-instance Task dispatch (no `Workflow` tool)

Run the same three phases by hand with the `Task` tool:

1. **Plan** — one `Task` call on the planner tier: "Decompose this task into independent work items, each self-contained with no cross-item context needed. Task: `<task>`." Skip this call entirely if Phase 0 scouting already produced the work-list.
2. **Execute** — one `Task` call per item on the worker tier, launched together (parallel Task calls in one message, or `run_in_background: true` and collect notifications).
3. **Synthesize** — one `Task` call on the planner tier: hand it every worker result and ask for one coherent answer, conflicts and gaps flagged.

## Guardrails

- **Don't fan out what you haven't scoped.** Discover the work-list first (Phase 0), then orchestrate over it. A planner guessing at structure wastes the expensive stage.
- **This is opt-in scale.** A workflow can spawn many agents. Use it when the task genuinely decomposes into parallel work; for a single-thread task, just do it, or use `/advisor` for a course-correction.
- **Literal dollar amounts in this file are escaped (`\$5`) on purpose** — `$<digit>` is consumed by Claude Code's positional-argument substitution and silently corrupts to an empty string or a fragment of the invocation args otherwise. See `docs/advisor-and-orchestrate.md`'s Gotchas section.

## Related

- `/advisor` — the inverse: a cheap main loop consulting an expensive model on demand.
- Economics, price gradient, and the native benchmarks: `docs/advisor-and-orchestrate.md`.
