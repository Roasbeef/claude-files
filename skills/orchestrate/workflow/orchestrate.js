// NOTE: `meta` must be a PURE LITERAL — no string concatenation, no template
// interpolation, no variables. The Workflow tool rejects anything else with
// "meta must be a pure literal", which silently breaks every run. Keep every
// field below a single literal.
export const meta = {
  name: 'orchestrate',
  description:
    'Expensive planner decomposes a task, cheap workers execute in parallel, expensive synthesizer stitches the results',
  whenToUse:
    'Invoked by the /orchestrate skill once the task is scoped and tiers are chosen. Runs Plan (expensive tier, skipped if the caller already supplies items) -> Execute (cheap tier, one worker per item, in parallel) -> Synthesize (expensive tier, one coherent answer).',
  phases: [
    { title: 'Plan', detail: 'expensive tier decomposes the task into independent work items' },
    { title: 'Execute', detail: 'cheap tier, one worker per item, in parallel' },
    { title: 'Synthesize', detail: 'expensive tier stitches worker results into one answer' },
  ],
}

// args (from the skill's Phase 0):
//   task      - the task to decompose and run, verbatim
//   planner   - 'opus' | 'fable' (expensive tier for Plan/Synthesize; default 'opus')
//   worker    - 'sonnet' | 'haiku' (cheap tier for Execute; default 'sonnet')
//   isolation - true if workers mutate files in parallel and would collide (default false)
//   items     - optional pre-scoped [{ id, spec }, ...]; when supplied, Plan is skipped
//               entirely (the caller already knows the work-list, so paying for a
//               decomposition pass would be wasted expensive-tier tokens)
const {
  task,
  planner = 'opus',
  worker = 'sonnet',
  isolation = false,
  items: suppliedItems = null,
} = args || {}

const ISOLATION = isolation ? 'worktree' : undefined

const PLAN_SCHEMA = {
  type: 'object',
  properties: {
    items: {
      type: 'array',
      items: {
        type: 'object',
        properties: {
          id: { type: 'string' },
          spec: { type: 'string' },
        },
        required: ['id', 'spec'],
      },
    },
  },
  required: ['items'],
}

const RESULT_SCHEMA = {
  type: 'object',
  properties: {
    id: { type: 'string' },
    summary: { type: 'string' },
    changes: { type: 'array', items: { type: 'string' } },
  },
  required: ['id', 'summary'],
}

const FINAL_SCHEMA = {
  type: 'object',
  properties: {
    summary: { type: 'string' },
    conflicts: { type: 'array', items: { type: 'string' } },
    gaps: { type: 'array', items: { type: 'string' } },
  },
  required: ['summary'],
}

phase('Plan')
let items = suppliedItems
if (items && items.length) {
  log(`Using ${items.length} caller-supplied work items; skipping the planner`)
} else {
  const plan = await agent(
    `Decompose this task into independent work items. Return a compact list; ` +
      `each item is self-contained and needs no cross-item context beyond what ` +
      `you give it. Task: ${task}`,
    { model: planner, effort: 'high', phase: 'Plan', schema: PLAN_SCHEMA }
  )
  items = plan.items
  log(`Planner produced ${items.length} work items`)
}

phase('Execute')
const results = await parallel(
  items.map(item => () =>
    agent(
      `Do this work item and report what changed. Item id: ${item.id}. Spec: ${item.spec}`,
      {
        label: `execute:${item.id}`,
        phase: 'Execute',
        model: worker,
        schema: RESULT_SCHEMA,
        isolation: ISOLATION,
      }
    )
  )
)

phase('Synthesize')
const answer = await agent(
  `Synthesize these worker results into a single coherent result for the task: ${task}. ` +
    `Flag conflicts and gaps. Results: ${JSON.stringify(results.filter(Boolean))}`,
  { model: planner, effort: 'high', phase: 'Synthesize', schema: FINAL_SCHEMA }
)

return { items, results: results.filter(Boolean), answer }
