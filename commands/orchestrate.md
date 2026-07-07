---
description: Expensive planner fans out to cheap Sonnet workers via a Workflow, then synthesizes — the orchestrator half of the advisor/orchestrator cost-arbitrage pair
argument-hint: "[--planner=opus|fable] [--worker=sonnet|haiku] <task to decompose and run>"
allowed-tools: [Workflow, Task, Read, Grep, Glob, Bash, AskUserQuestion]
---

# /orchestrate — plan expensive, execute cheap

Emulate the **orchestrator pattern** from the advisor-tool docs: a high-intelligence model (Fable 5 / Opus 4.8) *plans and decomposes*, then delegates the token-heavy execution to a fleet of cheap **Sonnet 5** workers. Most tokens are generated at the worker rate. On BrowseComp the native version hit ~96% of Fable-solo quality at ~46% of the cost.

This is the inverse of `/advisor` (cheap main loop calling an expensive advisor). Here the *expensive* reasoning is scarce (one planning pass, one synthesis pass) and the *cheap* work is abundant (N parallel workers). The `Workflow` tool maps almost 1:1 to this — it takes a per-agent `model` override — so this command is mostly about setting the tiers correctly and keeping the expensive stages thin.

## The task
$ARGUMENTS

## Tiers

Parse `--planner=` and `--worker=` from the arguments. Defaults:

- **Planner / synthesizer:** `opus` (Opus 4.8, $5/$25). Use `fable` (Fable 5, $10/$50) for genuinely hard decomposition where the plan quality is the bottleneck.
- **Workers:** `sonnet` (Sonnet 5, $3/$15). Drop to `haiku` (Haiku 4.5, $1/$5) only for mechanical, well-specified sub-tasks.

Keep the planner and synthesizer stages **small** — they are the expensive tokens. The plan should be a compact list of independent work items, not prose. Push all the reading, editing, and searching into the Sonnet workers.

## Procedure

1. **Scout inline first (cheap).** Before authoring the workflow, do the light discovery yourself in this session — list the files, find the call sites, scope the diff — so the planning stage operates on a known work-list instead of guessing the shape. This keeps the expensive planner focused.

2. **Author and run a `Workflow`** with this shape (adjust to the task):

   ```js
   export const meta = {
     name: 'orchestrate',
     description: '<one line>',
     phases: [{ title: 'Plan' }, { title: 'Execute' }, { title: 'Synthesize' }],
   }

   // Expensive, scarce: decompose into independent work items.
   phase('Plan')
   const plan = await agent(
     `Decompose this task into independent work items. Return a compact list; ` +
     `each item is self-contained and needs no cross-item context. Task: ${TASK}`,
     { model: PLANNER, effort: 'high', phase: 'Plan', schema: PLAN_SCHEMA }
   )

   // Cheap, abundant: one Sonnet worker per item, in parallel.
   phase('Execute')
   const results = await parallel(
     plan.items.map(item => () =>
       agent(`Do this work item and report what changed: ${item.spec}`,
         { model: WORKER, phase: 'Execute', schema: RESULT_SCHEMA })
     )
   )

   // Expensive, scarce: stitch the worker outputs into one coherent answer.
   phase('Synthesize')
   const answer = await agent(
     `Synthesize these worker results into a single coherent result. ` +
     `Flag conflicts and gaps. Results: ${JSON.stringify(results.filter(Boolean))}`,
     { model: PLANNER, effort: 'high', phase: 'Synthesize', schema: FINAL_SCHEMA }
   )
   return answer
   ```

   Substitute `PLANNER` / `WORKER` with the chosen tiers, define the schemas for the task, and prefer `pipeline()` over `parallel()` when items flow through multiple stages (see the Workflow tool's own guidance — pipeline by default, barrier only when a stage genuinely needs all prior results at once).

3. **Isolation:** if workers mutate files in parallel and would collide, pass `isolation: 'worktree'` on the worker `agent()` calls. Skip it otherwise — worktrees are expensive.

4. **Relay the synthesis**, not the raw worker dumps. Read the workflow result and report the outcome; the per-worker output is background.

## Guardrails

- **Don't fan out what you haven't scoped.** Discover the work-list first (step 1), then orchestrate over it. A planner guessing at structure wastes the expensive stage.
- **Keep expensive stages thin.** If the planner or synthesizer is doing token-heavy reading, you've mis-sliced it — that work belongs in a worker.
- **This is opt-in scale.** A workflow can spawn many agents. Use it when the task genuinely decomposes into parallel work; for a single-thread task, just do it, or use `/advisor` for a course-correction.

## Related

- `/advisor` — the inverse: a cheap main loop consulting an expensive model on demand.
