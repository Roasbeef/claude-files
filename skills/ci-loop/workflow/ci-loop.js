// NOTE: `meta` must be a PURE LITERAL — no string concatenation, no template
// interpolation, no variables. The Workflow tool rejects anything else with
// "meta must be a pure literal", which silently breaks every run. Keep every
// field below a single literal.
export const meta = {
  name: 'ci-loop',
  description:
    'Babysit CI after a push: watch a run, classify failures, fix or rerun, loop until green or a justified bail',
  whenToUse:
    'Invoked by the /ci-loop skill after it pushes a branch and resolves the PR/run to watch. Acts as a state machine: monitor the run to completion, classify each failure (lint/build vs test vs infra), mechanically fix trivial breakage, reproduce test failures locally to tell flake from real bug, rerun suspected flakes within a budget, hand reproduced failures to a fixer agent that commits a fixup and pushes, and end only when CI is green or it can justify a bail back to the user.',
  phases: [
    { title: 'Monitor', detail: 'watch the latest CI run to completion' },
    { title: 'Classify', detail: 'bucket failures: lint/build, test, infra' },
    { title: 'Fix', detail: 'mechanically repair trivial/build failures' },
    { title: 'Reproduce', detail: 'run failing tests locally — flake or real?' },
    { title: 'Repair', detail: 'fixer agent fixes the real failure, commits, pushes' },
  ],
}

// args (from the skill's Phase 0):
//   pr           - PR number to watch (optional; branch is enough)
//   branch       - feature branch the run belongs to (default: current)
//   base         - base branch for fixup targeting (default: main)
//   brief        - what the change is + any context the fixers need
//   briefPath    - path to the brief on disk; agents Read it fresh so a mid-run
//                  edit is honored on the next phase
//   maxIters     - safety cap on monitor→remediate cycles
//   flakeRetries - per-job rerun budget before a suspected flake is escalated
//   infraRetries - per-run rerun budget for non-actionable infra failures
//   allowPushToDefault - if true, may push to main/master (default false)
//   profile      - 'lite' | 'standard' | 'thorough' speed-vs-rigor dial
const {
  pr,
  branch = '',
  base = 'main',
  brief = '',
  briefPath = '.ci-loop/brief.md',
  maxIters,
  flakeRetries = 2,
  infraRetries = 2,
  allowPushToDefault = false,
  profile = 'standard',
} = args || {}

// Profiles set the speed-vs-rigor point. They differ in the monitor-cycle cap
// (how many CI runs we are willing to babysit) and how aggressively reproduce
// re-runs a test before trusting its verdict. The bounded/mechanical phases
// (monitor, classify, reproduce orchestration) run on Sonnet across all
// profiles — that is the floor. The two quality-critical phases — the trivial
// fixer and the test fixer — inherit the strong main-loop model, because a
// wrong code fix pushed to CI is the expensive mistake.
const PROFILES = {
  lite: { maxIters: 5, reproRuns: 3, model: 'sonnet' },
  standard: { maxIters: 10, reproRuns: 5, model: 'sonnet' },
  thorough: { maxIters: 20, reproRuns: 10, model: 'sonnet' },
}
const prof = PROFILES[profile] || PROFILES.standard
const iterCap = maxIters ?? prof.maxIters

// Budget awareness. `budget.total` is null when the user set no token target,
// in which case we never gate on it. When a target IS set we refuse to start a
// monitor cycle we cannot pay for, so the loop reaches a verdict rather than
// dying mid-fix and leaving a half-applied repair on the branch.
const BUDGETED = !!(budget && budget.total)
const PER_CYCLE_EST = 90_000

// k formats a token count for logs.
function k(n) {
  return Math.round(n / 1000) + 'k'
}

// pushTargetLine tells the fixers where they may push. The skill resolves the
// branch; we only ever push fixups to that feature branch, never to the default
// branch unless explicitly allowed.
const pushGuard = allowPushToDefault
  ? 'You MAY push to any branch.'
  : `You must NOT push to ${base} or any default branch (main/master). Only ` +
    `push to the feature branch${branch ? ` "${branch}"` : ''}. If HEAD is on ` +
    `a default branch, STOP and report status "cannot-fix" with that reason.`

// briefLine tells an agent to prefer the on-disk brief over the frozen copy.
const briefLine =
  `Context for this change lives at ${briefPath} — Read it fresh from disk ` +
  `first; a mid-run edit there overrides the inline copy below.`

// targetSelector is the gh selector every agent uses to find the run: a PR
// number when we have one, else the branch.
const targetSelector = pr
  ? `the PR #${pr}`
  : `the most recent run on branch "${branch}" (use the current branch if empty)`

// ---- schemas --------------------------------------------------------------

// What the monitor agent reports after watching one run to completion.
const MONITOR_SCHEMA = {
  type: 'object',
  required: ['green', 'noRun'],
  properties: {
    green: { type: 'boolean', description: 'true iff the run concluded success' },
    noRun: {
      type: 'boolean',
      description: 'true if no run could be found/started for the head commit',
    },
    runId: { type: 'string' },
    headSha: { type: 'string' },
    conclusion: {
      type: 'string',
      description: 'success | failure | cancelled | timed_out | startup_failure',
    },
    failedJobs: {
      type: 'array',
      items: {
        type: 'object',
        required: ['name'],
        properties: {
          name: { type: 'string' },
          conclusion: { type: 'string' },
        },
      },
    },
  },
}

// How classify buckets each failing job.
const CLASSIFY_SCHEMA = {
  type: 'object',
  required: ['failures'],
  properties: {
    failures: {
      type: 'array',
      items: {
        type: 'object',
        required: ['job', 'category'],
        properties: {
          job: { type: 'string' },
          category: {
            type: 'string',
            enum: ['lint', 'build', 'test', 'infra', 'other'],
            description:
              'lint=format/static/generated-out-of-date; build=compile; ' +
              'test=unit/integration assertion; infra=runner/network/dep ' +
              'outage unrelated to the code; other=unclear',
          },
          tests: {
            type: 'array',
            items: { type: 'string' },
            description: 'failing test identifiers if category is test',
          },
          excerpt: { type: 'string', description: 'short failing-log excerpt' },
          fixHint: { type: 'string' },
        },
      },
    },
  },
}

// What a fixer (trivial or test) reports.
const FIX_SCHEMA = {
  type: 'object',
  required: ['status', 'pushed'],
  properties: {
    status: {
      type: 'string',
      enum: ['fixed', 'cannot-fix'],
      description:
        'fixed = a fixup was committed and the local check passes; ' +
        'cannot-fix = could not produce a working fix (triggers a bail)',
    },
    pushed: { type: 'boolean' },
    commits: { type: 'array', items: { type: 'string' } },
    files: { type: 'array', items: { type: 'string' } },
    addressed: {
      type: 'array',
      items: { type: 'string' },
      description: 'job names / test ids this fix addressed',
    },
    reason: {
      type: 'string',
      description: 'if cannot-fix, the user-facing explanation of why',
    },
  },
}

// Per-test reproduce verdict. The agent is told to bias toward "reproduced"
// when unsure, so a real bug is never silently dismissed as a flake.
const REPRODUCE_SCHEMA = {
  type: 'object',
  required: ['verdicts'],
  properties: {
    verdicts: {
      type: 'array',
      items: {
        type: 'object',
        required: ['test', 'verdict'],
        properties: {
          test: { type: 'string' },
          job: { type: 'string' },
          verdict: {
            type: 'string',
            enum: ['flake', 'reproduced', 'cannot-run'],
            description:
              'flake=passed locally across repeated runs; ' +
              'reproduced=failed locally (a real bug); ' +
              'cannot-run=needs CI-only env, could not run locally',
          },
          runsLocal: { type: 'number' },
          failsLocal: { type: 'number' },
          detail: { type: 'string' },
        },
      },
    },
  },
}

// What a rerun action reports.
const RERUN_SCHEMA = {
  type: 'object',
  required: ['rerun'],
  properties: {
    rerun: { type: 'boolean' },
    note: { type: 'string' },
  },
}

// ---- agents ---------------------------------------------------------------

// monitorCI finds the latest run for the target and watches it to completion.
// Bounded/mechanical → Sonnet. It must keep emitting output so it never trips
// the no-progress watchdog during a long run.
async function monitorCI(iter) {
  return agent(
    `You are the CI monitor (cycle ${iter}). Find and watch ${targetSelector}
to completion, then report the outcome. Do NOT fix anything.

Steps, in bounded shell calls (never one long silent command):
1. Identify the head commit and its run:
     git rev-parse HEAD
     gh run list ${pr ? `` : `--branch "${branch || '$(git branch --show-current)'}"`} --limit 5 \\
       --json databaseId,headSha,status,conclusion,workflowName,event
   Pick the most recent run whose headSha matches HEAD. If none exists yet
   (you just pushed), wait and re-list a few times (e.g. sleep 15 between
   tries, up to ~2 min) until a run appears.
2. Watch it to completion. Prefer:
     gh run watch <id> --interval 30 --exit-status
   It streams progress, so it will not go silent. If your shell caps command
   time, fall back to short polling: loop \`sleep 20 && gh run view <id> --json status,conclusion\`
   until status is "completed".
3. When complete, collect the failing jobs:
     gh run view <id> --json conclusion,jobs

Report: green (true iff conclusion is success), the runId, headSha, the overall
conclusion, and the list of failed jobs (name + conclusion). If you genuinely
cannot find or start a run for the head commit after retrying, set noRun=true.`,
    {
      label: `monitor:c${iter}`,
      phase: 'Monitor',
      model: prof.model,
      schema: MONITOR_SCHEMA,
    }
  )
}

// classify pulls the failing logs and buckets each job. Bounded → Sonnet.
async function classify(runId, failedJobs, iter) {
  const jobs = JSON.stringify(failedJobs, null, 2)
  return agent(
    `You are the CI failure classifier (cycle ${iter}). Read the failing logs
for run ${runId} and bucket each failed job. Do NOT fix anything.

Failed jobs:
${jobs}

Fetch logs with:
  gh run view ${runId} --log-failed
  gh run view ${runId} --json jobs   (for job/step structure)

${briefLine}
Inline context:
${brief}

For each failed job, assign a category:
- lint  — gofmt/goimports, golangci-lint, ast-grep/sg scan, generated code out
          of date, commit-message lint, docs/spelling — mechanically fixable.
- build — compile / vet / type errors.
- test  — a unit or integration test assertion failed. List the failing test
          identifiers (package + test name) in "tests".
- infra — runner died, network/registry/dependency-download outage, OOM,
          unrelated timeout — NOT caused by this change's code.
- other — genuinely unclear from the log.

Give a short failing-log excerpt and a fix hint per job. Be precise about which
failures are test assertions vs environment problems — the loop treats them
very differently.`,
    {
      label: `classify:c${iter}`,
      phase: 'Classify',
      model: prof.model,
      schema: CLASSIFY_SCHEMA,
    }
  )
}

// fixTrivial repairs lint/build/other-mechanical failures, commits a fixup, and
// pushes. Quality-critical (it edits code) → strong main-loop model.
async function fixTrivial(items, iter) {
  const list = JSON.stringify(items, null, 2)
  return agent(
    `Fix these mechanically-fixable CI failures (cycle ${iter}), then push so CI
re-runs. These are lint/format/static/build failures — deterministic, no
investigation needed beyond the log.

Failures:
${list}

Base branch: ${base}
${briefLine}
Inline context:
${brief}

For each failure:
1. Reproduce the check locally where possible (e.g. make lint / make fmt /
   golangci-lint run / sg scan / go build ./... / make). Apply the minimal fix
   that matches surrounding conventions. For "generated code out of date",
   regenerate and commit the result.
2. Confirm the local check now passes (or, if it cannot run locally, that the
   fix is unambiguous from the log).
3. Stage exactly your changes and commit as a fixup against the commit that
   introduced the issue: \`git add <files> && git commit --fixup=<sha>\`. If no
   clear target, a normal \`ci: fix <thing>\` commit is fine.

Then push. ${pushGuard}

Return status "fixed" with the commit shas, changed files, addressed job names,
and pushed=true. If you cannot produce a working fix, return status
"cannot-fix" with a clear reason for the user.`,
    // Deliberately NO `model` key → inherits the strong main-loop model. This
    // is a quality-critical state (it edits and pushes code); do NOT add
    // `model: prof.model` here, which would silently downgrade it to Sonnet.
    { label: `fix:c${iter}`, phase: 'Fix', schema: FIX_SCHEMA }
  )
}

// reproduce runs each failing test locally several times to tell flake from
// real bug. Bounded orchestration → Sonnet, but it is instructed to bias toward
// "reproduced" when unsure so a real bug is never dismissed as a flake.
async function reproduce(testJobs, iter) {
  const list = JSON.stringify(testJobs, null, 2)
  return agent(
    `You are the flake-vs-real verifier (cycle ${iter}). For each failing test
below, decide whether it is a FLAKE (passes reliably when run locally) or a
REAL, reproduced failure. Do NOT fix anything — only classify.

Failing test jobs:
${list}

${briefLine}
Inline context:
${brief}

Method, per failing test:
1. Run the specific test locally, in isolation, ${prof.reproRuns} times
   (e.g. \`go test -run '^TestName$' -count=${prof.reproRuns} ./pkg/...\`, or the
   project's documented way to run that test). Use -race where the suite does.
2. Tally local runs vs local failures.

Verdict rules:
- "reproduced" — it failed at least once locally. It is a real bug.
- "flake" — it passed every local run AND nothing in the diff plausibly causes
  intermittency. Only call flake when you have positive local evidence of
  passing.
- "cannot-run" — the test needs CI-only infrastructure you cannot stand up
  locally. Do NOT guess flake in this case.

IMPORTANT: when you are unsure, return "reproduced", never "flake". A flake
verdict tells the loop to just rerun CI; a wrong flake verdict hides a real bug.
Report runsLocal / failsLocal and a short detail per test.`,
    {
      label: `reproduce:c${iter}`,
      phase: 'Reproduce',
      model: prof.model,
      schema: REPRODUCE_SCHEMA,
    }
  )
}

// fixTest is the real-bug fixer: investigate, fix, verify locally, commit a
// fixup, push. Quality-critical → strong main-loop model. If it cannot produce
// a working fix it returns "cannot-fix", which bails the loop back to the user.
async function fixTest(realTests, iter) {
  const list = JSON.stringify(realTests, null, 2)
  return agent(
    `You are the CI fixer (cycle ${iter}). These test failures reproduce locally
(or could not be dismissed as flakes). Find the root cause, fix it, prove the
fix locally, commit a fixup, and push so CI re-runs.

Reproduced failures:
${list}

Base branch: ${base}
${briefLine}
Inline context:
${brief}

Do all of:
1. Investigate the failing test and the code under test. Find the actual root
   cause — do NOT weaken the assertion or skip the test to make it pass. (If you
   believe the TEST itself is wrong, fix the test to assert the correct
   behavior, and say so in your reason.)
2. Implement the minimal fix consistent with the surrounding code.
3. Run the failing test(s) locally until they pass reliably (re-run a few times,
   with -race where the suite uses it). Run the affected package's tests so you
   do not regress neighbors.
4. Commit as a fixup against the right commit:
   \`git add <files> && git commit --fixup=<sha>\` (else a \`ci: fix <thing>\`
   commit). Use \`hunk stage\` if a file mixes your fix with unrelated changes.
5. Push. ${pushGuard}

Return status "fixed" with commit shas, changed files, the tests addressed, and
pushed=true. If after a genuine effort you cannot produce a working fix —
the failure is outside this change, needs a design decision, or resists every
fix you try — return status "cannot-fix" with a precise, user-facing reason and
what you tried. Do NOT commit a fix you have not verified locally.`,
    // Deliberately NO `model` key → inherits the strong main-loop model. This
    // is the most quality-critical state in the loop; do NOT add
    // `model: prof.model` here, which would silently downgrade it to Sonnet.
    { label: `repair:c${iter}`, phase: 'Repair', schema: FIX_SCHEMA }
  )
}

// rerunJobs reruns the failed jobs of a run (for suspected flakes / transient
// infra). Mechanical → Sonnet. A rerun starts a fresh run the monitor picks up.
async function rerunJobs(runId, why, iter) {
  return agent(
    `Rerun the failed jobs of CI run ${runId} (reason: ${why}). Run exactly:
  gh run rerun ${runId} --failed
Then confirm a new attempt started (gh run view ${runId} --json status). Do
nothing else — do not edit code. Report rerun=true if the rerun was triggered.`,
    {
      label: `rerun:c${iter}`,
      phase: 'Monitor',
      model: prof.model,
      schema: RERUN_SCHEMA,
    }
  )
}

// ---- state machine --------------------------------------------------------

const history = []
const fixesApplied = []
const flakesRerun = []

// Per-job flake counter: how many times we have rerun a job as a suspected
// flake. Once it exceeds flakeRetries we stop trusting "flake" and escalate the
// job to the fixer as a real failure.
const flakeCount = {}
let infraReruns = 0

let iter = 0
let finalState = 'unknown'
let bail = null

log(
  `Profile: ${profile} (cap ${iterCap} cycles, ${prof.reproRuns} local repro ` +
  `runs, flake budget ${flakeRetries}/job)`
)

while (iter < iterCap) {
  if (BUDGETED && budget.remaining() < PER_CYCLE_EST) {
    log(`Budget low (${k(budget.remaining())} left) — stopping`)
    finalState = 'budget'
    break
  }

  iter += 1
  phase('Monitor')
  const budgetNote = BUDGETED ? ` (${k(budget.remaining())} budget left)` : ''
  log(`Cycle ${iter}: watching CI${budgetNote}`)

  const mon = await monitorCI(iter)

  if (!mon) {
    finalState = 'monitor-error'
    break
  }
  if (mon.noRun) {
    log(`Cycle ${iter}: no CI run found for HEAD`)
    finalState = 'no-ci'
    history.push({ iter, action: 'monitor', result: 'no-run' })
    break
  }
  if (mon.green) {
    log(`Cycle ${iter}: CI is GREEN (run ${mon.runId})`)
    finalState = 'green'
    history.push({ iter, runId: mon.runId, action: 'monitor', result: 'green' })
    break
  }

  const failedJobs = mon.failedJobs || []
  log(`Cycle ${iter}: run ${mon.runId} failed — ${failedJobs.length} job(s)`)
  history.push({
    iter,
    runId: mon.runId,
    action: 'monitor',
    result: 'failed',
    failedJobs: failedJobs.map((j) => j.name),
  })

  phase('Classify')
  const cls = await classify(mon.runId, failedJobs, iter)
  const failures = (cls && cls.failures) || []

  const trivial = failures.filter(
    (f) => f.category === 'lint' || f.category === 'build' || f.category === 'other'
  )
  const testFails = failures.filter((f) => f.category === 'test')
  const infra = failures.filter((f) => f.category === 'infra')

  // State 1: mechanically-fixable breakage. Fix it, push, re-monitor the fresh
  // run (a push re-evaluates everything, so we handle one class per cycle).
  if (trivial.length > 0) {
    phase('Fix')
    log(`Cycle ${iter}: ${trivial.length} trivial/build failure(s) → fixing`)
    const fx = await fixTrivial(trivial, iter)
    history.push({ iter, action: 'fix-trivial', result: fx?.status })
    if (fx && fx.status === 'fixed' && fx.pushed) {
      fixesApplied.push({ iter, kind: 'trivial', ...fx })
      continue
    }
    // Could not fix the mechanical failure — that is a real bail.
    bail = {
      reason: (fx && fx.reason) || 'trivial fixer could not push a fix',
      kind: 'trivial',
    }
    finalState = 'bailed-cannot-fix'
    break
  }

  // State 2: test failures. Reproduce locally to separate flake from real bug.
  if (testFails.length > 0) {
    phase('Reproduce')
    const rep = await reproduce(testFails, iter)
    const verdicts = (rep && rep.verdicts) || []

    // Escalate suspected flakes whose rerun budget is exhausted: a "flake" that
    // keeps coming back is treated as a real bug for the fixer.
    const real = []
    const flakeJobs = new Set()
    for (const v of verdicts) {
      // Key the flake budget on the job, falling back to the test id so a
      // verdict with no job name gets its own budget rather than sharing one
      // collapsed '' bucket with every other job-less flake.
      const jobName = v.job || v.test || ''
      if (v.verdict === 'reproduced' || v.verdict === 'cannot-run') {
        real.push(v)
      } else if (v.verdict === 'flake') {
        const seen = flakeCount[jobName] || 0
        if (seen >= flakeRetries) {
          log(`Cycle ${iter}: "${v.test}" flaked past budget → escalating as real`)
          real.push({ ...v, verdict: 'reproduced', escalatedFlake: true })
        } else {
          flakeJobs.add(jobName)
        }
      }
    }

    // If anything is real, fix it (a push re-runs the flaky jobs too).
    if (real.length > 0) {
      phase('Repair')
      log(`Cycle ${iter}: ${real.length} real test failure(s) → fixer`)
      const fix = await fixTest(real, iter)
      history.push({ iter, action: 'fix-test', result: fix?.status })
      if (fix && fix.status === 'fixed' && fix.pushed) {
        fixesApplied.push({ iter, kind: 'test', ...fix })
        continue
      }
      bail = {
        reason: (fix && fix.reason) || 'fixer could not resolve the failure',
        kind: 'test',
        tests: real.map((r) => r.test),
      }
      finalState = 'bailed-cannot-fix'
      break
    }

    // Only flakes remained — rerun the failed jobs within budget.
    if (flakeJobs.size > 0) {
      for (const j of flakeJobs) flakeCount[j] = (flakeCount[j] || 0) + 1
      log(`Cycle ${iter}: only flakes — rerunning ${mon.runId}`)
      const rr = await rerunJobs(mon.runId, 'suspected flake', iter)
      flakesRerun.push({ iter, runId: mon.runId, jobs: [...flakeJobs] })
      history.push({ iter, action: 'rerun-flake', result: rr?.rerun })
      if (rr && rr.rerun) continue
      // Could not even rerun — bail rather than spin.
      bail = { reason: 'suspected flake but rerun could not be triggered' }
      finalState = 'stuck'
      break
    }

    // Test jobs failed but reproduce produced no actionable verdict — re-running
    // the same red run would only spin to the cap, so stop and report.
    log(`Cycle ${iter}: test failures yielded no actionable verdict — stopping`)
    bail = {
      reason: 'CI reported test failures but reproduce returned no verdict',
      kind: 'test',
      jobs: testFails.map((f) => f.job),
    }
    finalState = 'stuck'
    break
  }

  // State 3: only infra failures. Rerun within budget, else bail — we cannot fix
  // a broken runner or a registry outage from here.
  if (infra.length > 0) {
    if (infraReruns < infraRetries) {
      infraReruns += 1
      log(`Cycle ${iter}: infra failure — rerun ${infraReruns}/${infraRetries}`)
      const rr = await rerunJobs(mon.runId, 'infra/transient', iter)
      history.push({ iter, action: 'rerun-infra', result: rr?.rerun })
      if (rr && rr.rerun) continue
    }
    bail = {
      reason: 'non-actionable infra failure persisted past the rerun budget',
      kind: 'infra',
      jobs: infra.map((f) => f.job),
    }
    finalState = 'bailed-infra'
    break
  }

  // Nothing actionable was found this cycle (e.g. classify returned empty while
  // the run was red). Avoid an infinite spin.
  if (trivial.length === 0 && testFails.length === 0 && infra.length === 0) {
    log(`Cycle ${iter}: run is red but no failure was classified — stopping`)
    bail = { reason: 'CI is red but no actionable failure could be classified' }
    finalState = 'stuck'
    break
  }
}

if (finalState === 'unknown') finalState = 'maxIters'

return {
  profile,
  finalState,
  iterations: iter,
  green: finalState === 'green',
  fixesApplied,
  flakesRerun,
  bail,
  history,
  tokensSpent: BUDGETED ? budget.spent() : null,
}
