# Cost-arbitrage agents: `/advisor` and `/orchestrate`

Two skills that reproduce the [advisor-tool](https://platform.claude.com/docs/en/agents-and-tools/tool-use/advisor-tool) economics inside Claude Code, where there is no advisor tool. Both rest on one idea: **the model that generates the most tokens should be the cheap one, and the expensive model should be touched rarely.**

They are mirror images.

| | `/advisor` | `/orchestrate` |
|---|---|---|
| Main loop | **cheap** (Sonnet 5) | expensive (Opus/Fable) |
| Expensive model | called on demand, ~once/task | the planner + synthesizer, scarce by design |
| Cheap model | the whole session | the worker fleet, abundant |
| Shape | cheap executor → expensive advisor | expensive planner → cheap workers |
| Native benchmark | ~92% of Fable-solo at ~63% cost (SWE-bench Pro) | ~96% of Fable-solo at ~46% cost (BrowseComp) |

The price gradient that makes either worth doing (per 1M tokens, in/out):

| Tier | In | Out |
|---|---|---|
| Fable 5 | $10 | $50 |
| Opus 4.8 | $5 | $25 |
| Sonnet 5 | $3 | $15 ($2/$10 intro → 2026-08-31) |
| Haiku 4.5 | $1 | $5 |

---

## `/advisor` — cheap loop, expensive consult

### The setup that makes it pay off

`/advisor` only saves anything if the **session's main loop is Sonnet 5**. Run the working session on Sonnet, and let `/advisor` reach up to Opus 4.8 or Fable 5 at a few checkpoints. If the session is already on Opus/Fable, every turn is paying advisor rates and the skill saves nothing — you *are* the advisor.

Set the session model with `/model` (or launch Claude Code with Sonnet) before starting real work you expect to consult on.

### When to reach for it

The native tool gets called ~once per task because the executor decides when. Reproduce that judgment. Consult at:

- **Before committing to a plan** for a non-trivial task — cheap to get the shape reviewed before you spend Sonnet tokens building the wrong thing.
- **After 2 failed attempts** at the same fix — ask "what am I missing?" rather than thrashing.
- **Before a risky or irreversible change** — migrations, public-API changes, deletes, wide blast radius.
- **At a genuine fork** — two defensible approaches, load-bearing choice.

Do **not** consult for things you can verify by reading the code, routine edits, or anything where you'd accept your own first answer. Over-consulting is the failure mode; it turns the cheap session expensive one call at a time. Aim for roughly once per task.

### Two invariants

1. **Cheap main loop** (above).
2. **Relevant packet, not raw.** A sub-agent starts fresh and cannot share the main loop's prompt cache — caches are model-scoped, so every byte you send is cold, expensive context. That argues against pasting the *whole transcript* or unrelated files, not against detail: a thin packet earns thin, generic advice. Send everything that bears on the question — the full situation, what you tried and how it failed, the relevant code with real context — and cut only what doesn't. You get back a plan, a decision, or what you're missing — never code; the executor writes the code. For a task with several checkpoints, keep one advisor alive and `SendMessage` it the delta rather than re-spawning cold each time.

### Usage

```
/advisor                          # picks a tier by difficulty, default Opus
/advisor --tier=fable <question>  # force the top tier for a hard call
```

Tier is chosen per invocation: Opus 4.8 by default (half of Fable's cost, still a large jump over a Sonnet executor); Fable 5 for the hardest, highest-stakes questions.

---

## `/orchestrate` — expensive plan, cheap execution

### The shape

A high-intelligence planner (Opus/Fable) decomposes the task into independent work items; a fleet of **Sonnet 5** workers does the token-heavy execution in parallel; a thin synthesis pass stitches the results. Most tokens are generated at the worker rate. The `Workflow` tool maps almost 1:1 — it takes a per-agent `model` override — so the command is mostly about setting tiers and keeping the expensive stages thin.

Unlike `/advisor`, this works from **any main-loop model**, because it keeps the expensive model scarce by construction (one planning pass, one synthesis pass) regardless of what the session runs on.

### The rule that matters

**Keep the expensive stages thin.** The planner should emit a compact list of independent work items, not prose. If the planner or synthesizer is doing token-heavy *reading*, you've mis-sliced it — that work belongs in a Sonnet worker. And **don't fan out what you haven't scoped**: discover the work-list inline first (list the files, find the call sites, scope the diff), then orchestrate over it, so the planner operates on a known shape instead of guessing.

### Usage

```
/orchestrate <task>
/orchestrate --planner=fable --worker=sonnet <task>
/orchestrate --worker=haiku <mechanical, well-specified task>
```

Defaults: Opus planner/synthesizer, Sonnet workers. Reach for Fable as planner only when plan quality is the bottleneck; drop workers to Haiku only for mechanical, fully-specified sub-tasks.

---

## Choosing between them

- **One thread of work, occasionally stuck** → `/advisor`. You're doing the work; you just want a course-correction at the hard moments.
- **The task decomposes into parallel, independent pieces** → `/orchestrate`. Research sweeps, multi-file migrations, audits — anything where N workers can run blind to each other.
- **A single hard task with no decomposition and no cheap main loop** → just do it directly, or drop the session to Sonnet and use `/advisor` for the one or two moments that need a bigger model.

They compose: an `/orchestrate` planner can itself lean on `/advisor` judgment, and a Sonnet session can alternate between doing work and orchestrating a fan-out when a sub-task is wide.

## Gotchas, collected

- **`/advisor` on an Opus/Fable session is a no-op economically.** Move the main loop to Sonnet first.
- **Model-scoped caches** mean each one-shot consult pays cold context. Keep packets relevant and consult rarely; for a task with several checkpoints, keep one advisor alive and `SendMessage` the delta instead of re-spawning.
- **Subscription vs API.** On a Max plan the "executor rate" isn't literal dollars, but the same logic governs your rate-limit and context budget. On API it's real money.
- **Workflows are opt-in scale.** `/orchestrate` can spawn many agents. Use it when the task genuinely decomposes; for a single-thread task the fan-out is pure overhead.
- **`second-opinion` / codex MCP** is a real external advisor, but a different provider — a fit for "sanity-check with an outside model," not for same-family cost arbitrage.
