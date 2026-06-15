// NOTE: `meta` must be a PURE LITERAL — no string concatenation, no template
// interpolation, no variables. The Workflow tool rejects anything else with
// "meta must be a pure literal", which silently breaks every run. Keep every
// field below a single literal.
export const meta = {
  name: 'review-loop',
  description:
    'Adversarial review→triage→fix loop until a cold verifier signs off',
  whenToUse:
    'Invoked by the /review-loop skill after it resolves scope and writes a design brief. Fans out lens-specific adversarial reviewers, triages findings against the code, applies confirmed fixes as fixup commits, loops on the new surface, and ends only when a chunked cold verifier approves.',
  phases: [
    { title: 'Find', detail: 'one adversarial reviewer per lens' },
    { title: 'Triage', detail: 'verify, dedup, kill false positives' },
    { title: 'Apply', detail: 'implement confirmed fixes as fixup commits' },
    { title: 'Verify', detail: 'chunked cold sign-off, one slice per area' },
  ],
}

// args (from the skill's Phase 0):
//   diffCmd   - exact git diff command defining the review surface
//   base      - base branch (for fixup targeting and the final diff)
//   brief     - contents of .review-loop/brief.md (intent + constraints)
//   briefPath - path to the brief on disk; agents Read it fresh so a mid-run
//               edit is honored on the next phase (the frozen `brief` arg is a
//               fallback only)
//   lenses    - [{ key, agentType, hunt }] selected from the changed files
//   cutoff    - 'high' | 'medium' | 'low' severity floor for in-loop fixes
//   maxIters  - safety cap on rounds (overrides the profile default)
//   profile   - 'lite' | 'standard' | 'thorough' speed-vs-rigor dial
const {
  diffCmd,
  base = 'main',
  brief = '',
  briefPath = '.review-loop/brief.md',
  lenses = [],
  cutoff = 'medium',
  maxIters,
  profile = 'standard',
} = args || {}

// Profiles set the speed-vs-rigor point. lite collapses the lens panel into one
// combined finder and runs a single round; thorough keeps the full panel and
// adds a completeness-critic pass. The mechanical/bounded phases (slice
// planning, triage) run on Sonnet across all profiles — that is the floor, we
// do not drop below it; finders, apply, and the cold verifiers inherit the
// strong main-loop model because those are the steps where quality is
// load-bearing.
const PROFILES = {
  lite: {
    maxIters: 1,
    combineFinders: true,
    critic: false,
    planModel: 'sonnet',
    triageModel: 'sonnet',
  },
  standard: {
    maxIters: 5,
    combineFinders: false,
    critic: false,
    planModel: 'sonnet',
    triageModel: 'sonnet',
  },
  thorough: {
    maxIters: 8,
    combineFinders: false,
    critic: true,
    planModel: 'sonnet',
    triageModel: 'sonnet',
  },
}
const prof = PROFILES[profile] || PROFILES.standard
const iterCap = maxIters ?? prof.maxIters

// Severity ordering so we can compare against the cutoff floor.
const SEV = { critical: 4, high: 3, medium: 2, low: 1, info: 0 }
const floor = SEV[cutoff] ?? 2

// Budget awareness. `budget.total` is null when the user set no token target,
// in which case we never gate on it. When a target IS set we reserve a slice
// for the (heaviest) verify phase and refuse to start a round we can't pay for,
// so the loop always reaches a verdict instead of dying mid-round.
const BUDGETED = !!(budget && budget.total)
const VERIFY_RESERVE = 180_000
const PER_ROUND_EST = 120_000

// k formats a token count for logs.
function k(n) {
  return Math.round(n / 1000) + 'k'
}

// briefLine tells an agent to prefer the on-disk brief over the frozen copy.
const briefLine =
  `The authoritative design brief lives at ${briefPath} — Read it fresh from ` +
  `disk first; a mid-run edit there overrides the inline copy below.`

// A finding as every finder must return it.
const FINDING = {
  type: 'object',
  required: ['id', 'severity', 'file', 'scenario', 'fix'],
  properties: {
    id: { type: 'string', description: 'stable short id, lens-prefixed' },
    severity: {
      type: 'string',
      enum: ['critical', 'high', 'medium', 'low', 'info'],
    },
    file: { type: 'string', description: 'file:line of the issue' },
    title: { type: 'string' },
    scenario: {
      type: 'string',
      description: 'concrete sequence that triggers the failure',
    },
    fix: { type: 'string', description: 'minimal fix sketch' },
  },
}

const FINDINGS_SCHEMA = {
  type: 'object',
  required: ['findings'],
  properties: { findings: { type: 'array', items: FINDING } },
}

// Triage verdict: survivors split into fix-now / follow-up / rejected.
const TRIAGE_SCHEMA = {
  type: 'object',
  required: ['fixNow', 'followUp', 'rejected'],
  properties: {
    fixNow: {
      type: 'array',
      items: {
        type: 'object',
        required: ['ids', 'severity', 'rationale', 'fixSketch', 'files'],
        properties: {
          ids: { type: 'array', items: { type: 'string' } },
          severity: {
            type: 'string',
            enum: ['critical', 'high', 'medium', 'low', 'info'],
          },
          rationale: { type: 'string' },
          fixSketch: { type: 'string' },
          files: { type: 'array', items: { type: 'string' } },
          fixupTarget: {
            type: 'string',
            description: 'commit sha the fix belongs to, or empty',
          },
        },
      },
    },
    followUp: {
      type: 'array',
      items: {
        type: 'object',
        required: ['ids', 'why', 'issueTitle'],
        properties: {
          ids: { type: 'array', items: { type: 'string' } },
          why: { type: 'string' },
          issueTitle: { type: 'string' },
        },
      },
    },
    rejected: {
      type: 'array',
      items: {
        type: 'object',
        required: ['ids', 'why'],
        properties: {
          ids: { type: 'array', items: { type: 'string' } },
          why: { type: 'string' },
        },
      },
    },
  },
}

const APPLY_SCHEMA = {
  type: 'object',
  required: ['applied'],
  properties: {
    applied: {
      type: 'array',
      items: {
        type: 'object',
        required: ['ids', 'commit', 'status'],
        properties: {
          ids: { type: 'array', items: { type: 'string' } },
          commit: { type: 'string' },
          status: { type: 'string', enum: ['applied', 'failed', 'skipped'] },
          files: {
            type: 'array',
            items: { type: 'string' },
            description: 'files actually changed by this fix',
          },
          note: { type: 'string' },
        },
      },
    },
  },
}

const VERDICT_SCHEMA = {
  type: 'object',
  required: ['decision'],
  properties: {
    decision: { type: 'string', enum: ['approve', 'reopen'] },
    findings: { type: 'array', items: FINDING },
    summary: { type: 'string' },
  },
}

// The verify planner returns bounded review slices so no single cold verifier
// has to read the whole diff (the failure mode that stalled the watchdog).
const VERIFY_PLAN_SCHEMA = {
  type: 'object',
  required: ['slices'],
  properties: {
    slices: {
      type: 'array',
      items: {
        type: 'object',
        required: ['area', 'paths'],
        properties: {
          area: { type: 'string', description: 'short slice name' },
          paths: {
            type: 'array',
            items: { type: 'string' },
            description:
              'plain file or directory path prefixes owned by this slice ' +
              '(NO globs/wildcards — e.g. "db/" or "oor/registry.go")',
          },
          approxLines: { type: 'number' },
        },
      },
    },
  },
}

// rejectedDigest carries earlier rounds' rejected findings forward so finders
// stop re-discovering the same triaged-out non-bugs (the main driver of rounds
// that never converge). A materially new angle on the same code is still fine.
function rejectedDigest(rejected) {
  if (!rejected || rejected.length === 0) return ''
  const lines = rejected
    .slice(-40)
    .map((r) => `- [${(r.ids || []).join(',')}] ${r.why}`)
    .join('\n')
  return `
Already triaged OUT in earlier rounds — do NOT re-raise these (a genuinely new,
materially different angle on the same code is allowed, a restatement is not):
${lines}
`
}

// lensBlock renders one lens's hunt list for a prompt.
function lensBlock(lens) {
  return `- ${lens.key}: ${lens.hunt}`
}

// finderPrompt builds the adversarial prompt for one lens. Every finder
// reviews the same surface and gets the same brief, differing only in lens.
function finderPrompt(lens, round, rejected) {
  return `You are an ADVERSARIAL reviewer (round ${round}). BREAK this change,
do not grade it. Only report findings you can argue concretely from the code.

Scope — review exactly this surface:
  ${diffCmd}

${briefLine}
Inline brief (intent + hard constraints; respect accepted tradeoffs and the
pre-flight baseline noted here — do not report pre-existing breakage):
${brief}
${rejectedDigest(rejected)}
Your lens: ${lens.key}
Hunt specifically for: ${lens.hunt}

For each finding give a stable id (prefix with "${lens.key}-"), file:line,
severity, a concrete trigger SCENARIO (the exact sequence that causes the
failure), and a minimal fix sketch. If a suspicion turns out safe after you
read the code, omit it (or note it as verified-safe). No false positives —
triage will verify and reject anything you cannot substantiate.`
}

// combinedFinderPrompt collapses every lens into one reviewer for the lite
// profile: a single strong pass that hunts across all lenses at once.
function combinedFinderPrompt(round, rejected) {
  return `You are an ADVERSARIAL reviewer (round ${round}). BREAK this change,
do not grade it. Only report findings you can argue concretely from the code.
Review across ALL of the lenses below in one pass.

Scope — review exactly this surface:
  ${diffCmd}

${briefLine}
Inline brief (intent + hard constraints; respect accepted tradeoffs and the
pre-flight baseline noted here — do not report pre-existing breakage):
${brief}
${rejectedDigest(rejected)}
Lenses to hunt across:
${lenses.map(lensBlock).join('\n')}

For each finding give a stable lens-prefixed id, file:line, severity, a concrete
trigger SCENARIO, and a minimal fix sketch. No false positives — triage will
verify and reject anything you cannot substantiate.`
}

// runFinders fans out one adversarial reviewer per lens for a round (or a
// single combined reviewer under the lite profile).
async function runFinders(round, rejected) {
  if (prof.combineFinders) {
    const r = await agent(combinedFinderPrompt(round, rejected), {
      label: 'find:combined',
      phase: 'Find',
      schema: FINDINGS_SCHEMA,
    })
    return (r && r.findings) || []
  }

  const results = await parallel(
    lenses.map((lens) => () =>
      agent(finderPrompt(lens, round, rejected), {
        label: `find:${lens.key}`,
        phase: 'Find',
        agentType: lens.agentType,
        schema: FINDINGS_SCHEMA,
      })
    )
  )

  return results
    .filter(Boolean)
    .flatMap((r) => r.findings || [])
}

// criticPass (thorough profile) asks what the lens-scoped finders may have
// missed entirely: an area not reviewed, an interaction across files, a claim
// nobody verified. Strong model — this is a quality step, not a bounded one.
async function criticPass(round) {
  const r = await agent(
    `You are a completeness critic (round ${round}). The lens-scoped finders
have already run. Your job is to name what they STRUCTURALLY could not see:
an area of the diff no lens covered, an invariant that spans two files, a claim
in the brief nobody verified against the code, a failure mode that only appears
across a sequence of operations.

Scope: ${diffCmd}
${briefLine}
Inline brief:
${brief}

Return concrete findings only (id prefixed "critic-", file:line, severity,
scenario, fix). If coverage is genuinely complete, return an empty list.`,
    { label: `find:critic`, phase: 'Find', schema: FINDINGS_SCHEMA }
  )
  return (r && r.findings) || []
}

// triage verifies findings against the code, dedups, and classifies. Runs on
// Sonnet (the floor): the work is bounded because each finding says where to
// look. It is instructed to keep-when-uncertain so a cheaper judge never
// silently drops a real bug.
async function triage(findings, round) {
  const list = JSON.stringify(findings, null, 2)

  return agent(
    `You are the triage judge (round ${round}). You have read access to the
code. Verify every finding below against the actual cited lines.

Scope: ${diffCmd}
${briefLine}
Inline brief:
${brief}

Findings from the adversarial finders:
${list}

Do all of:
1. VERIFY each finding by re-reading the cited code. Reject anything you cannot
   reproduce from the source, or that the brief marks as accepted/out-of-scope/
   pre-existing. If you are UNSURE whether a finding is real, do NOT reject it —
   keep it (as fixNow if at/above cutoff, else followUp). Only reject what you
   can positively show is a non-issue.
2. DEDUP across lenses; merge complementary findings into one stronger item.
3. CLASSIFY survivors:
   - fixNow: severity at or above "${cutoff}". Give a concrete, repo-style fix
     sketch, the files touched, and the commit sha the fix should fix up
     (fixupTarget) if identifiable from git blame / the diff.
   - followUp: real but deferrable (below cutoff, or needs a design decision).
   - rejected: positively shown to be a false positive, with the reason.
Be decisive on what you can prove; conservative on what you cannot.`,
    {
      label: `triage:r${round}`,
      phase: 'Triage',
      model: prof.triageModel,
      schema: TRIAGE_SCHEMA,
    }
  )
}

// applyFixes implements the confirmed fix-now findings and commits each as a
// fixup. Runs as a single sequential agent because the fixes share one working
// tree and must not race; worktree isolation would defeat the fixup targeting.
async function applyFixes(fixNow, round) {
  const plan = JSON.stringify(fixNow, null, 2)

  return agent(
    `Apply these triage-confirmed fixes to the working tree (round ${round}),
in severity order. They share one tree — do them sequentially, do not spawn
parallel editors.

Base branch: ${base}
Fix plan:
${plan}

For each fix:
1. Implement the minimal change consistent with the surrounding code (match
   comment style, naming, line width, and project conventions).
2. Add or update tests that pin the fixed behavior when it is testable.
3. Build and run the relevant package tests; they must pass. Do not chase
   failures that predate this change set (see the brief's baseline).
4. Commit as a fixup: \`git add <files> && git commit --fixup=<fixupTarget>\`.
   If fixupTarget is empty, make a normal \`review:\` commit. Use \`hunk stage\`
   when a file mixes this fix with unrelated changes.

Return one applied entry per fix with its commit sha, status, and the list of
files you actually changed (used to decide which verify slices to re-check).`,
    {
      label: `apply:r${round}`,
      phase: 'Apply',
      schema: APPLY_SCHEMA,
    }
  )
}

// planVerify pre-materializes the full diff to a file and partitions the
// changed files into bounded slices. Cheap and bounded → runs on Sonnet.
async function planVerify() {
  return agent(
    `Prepare the cold verification surface. Be quick and do NOT review anything
yet. In bounded steps:
1. Run: git diff ${base}..HEAD --stat
2. Materialize the full diff: git diff ${base}..HEAD > .review-loop/final.diff
3. Group the changed files into 3-6 review slices by package/area, each bounded
   to roughly <=2000 changed lines so no single reviewer faces the whole diff.
   Keep related files together (a package and its tests in one slice).
For each slice, the "paths" it owns MUST be plain file or directory path
prefixes exactly as they appear in the diff (e.g. "db/" or "oor/registry.go").
Do NOT use globs or wildcards — downstream code matches these by literal prefix,
and every changed file must fall under exactly one slice's paths.
Return the slices (area name, the plain path prefixes it owns, approx lines).`,
    {
      label: 'verify:plan',
      phase: 'Verify',
      model: prof.planModel,
      schema: VERIFY_PLAN_SCHEMA,
    }
  )
}

// rematerialize rewrites .review-loop/final.diff after a repair pass changed
// the tree, so an incremental re-verify reads the current diff. Sonnet.
async function rematerialize() {
  return agent(
    `Run exactly: git diff ${base}..HEAD > .review-loop/final.diff
Then confirm the byte count with: wc -l .review-loop/final.diff
Do nothing else.`,
    { label: 'verify:rematerialize', phase: 'Verify', model: prof.planModel }
  )
}

// verifySlice runs one cold reviewer over a bounded slice. It reads the
// pre-materialized diff for its paths only and is told to work in bounded
// steps so it never goes silent long enough to trip the no-progress watchdog.
// Strong model (inherits): the cold cross-area verifier is the step that
// catches what the lens-scoped finders miss.
async function verifySlice(slice) {
  const paths = (slice.paths || []).join(' ')

  return agent(
    `Cold-verify ONE slice of a change you have not seen before. Approve it or
reopen it. Hunt for cross-cutting bugs the lens-scoped iterative finders may
have grown blind to (e.g. an effect applied in memory but not rolled back on a
later error, an invariant broken only across two files).

Your slice: ${slice.area}
Paths: ${paths}

The full diff is pre-materialized at .review-loop/final.diff. Review only your
slice: read it with \`git diff ${base}..HEAD -- ${paths}\` (or grep your paths
out of .review-loop/final.diff), and Read source files as needed.

${briefLine}

METHOD: work in bounded steps and keep making tool calls so you never go
silent for long. Do NOT run long builds or test suites — the tree is already
green from the apply phase; reason from the code. Return "approve", or
"reopen" with concrete findings (id, file:line, severity, scenario, fix). When
in doubt about something in YOUR slice, reopen.`,
    {
      label: `verify:${slice.area}`,
      phase: 'Verify',
      agentType: 'code-reviewer',
      schema: VERDICT_SCHEMA,
    }
  )
}

// verifySlices fans out one cold verifier per slice and aggregates: reopen if
// any slice reopens, else approve.
async function verifySlices(slices) {
  // Fallback: if the planner produced nothing, verify the whole diff as one
  // bounded slice rather than skipping verification.
  const effective =
    slices.length > 0 ? slices : [{ area: 'all', paths: [], approxLines: 0 }]

  log(`Cold verify: fanning out ${effective.length} slice verifiers`)

  const verdicts = await parallel(
    effective.map((s) => () =>
      verifySlice(s).then((v) => (v ? { slice: s.area, ...v } : null))
    )
  )

  const ok = verdicts.filter(Boolean)
  const reopened = ok.filter((v) => v.decision === 'reopen')

  return {
    decision: reopened.length > 0 ? 'reopen' : 'approve',
    findings: reopened.flatMap((v) => v.findings || []),
    summary: ok.map((v) => `${v.slice}:${v.decision}`).join(' '),
    sliceVerdicts: ok,
  }
}

// safeVerify wraps a verify fan-out so a stall/throw never discards the
// accumulated find/triage/apply work (already durable in git).
async function safeVerify(slices) {
  try {
    return await verifySlices(slices)
  } catch (err) {
    log(`Cold verify failed (${err}); returning accumulated work unverified`)
    return {
      decision: 'unverified',
      findings: [],
      summary: String(err),
      sliceVerdicts: [],
    }
  }
}

// atOrAboveCutoff keeps only fix-now items at or above the severity floor.
function atOrAboveCutoff(fixNow) {
  return (fixNow || []).filter((f) => (SEV[f.severity] ?? 0) >= floor)
}

// filesOf flattens the file lists of a set of fix-now items.
function filesOf(fixNow) {
  return (fixNow || []).flatMap((f) => f.files || [])
}

// normPath strips glob/dir suffixes so slice paths and file paths compare as
// plain prefixes.
function normPath(p) {
  return (p || '')
    .replace(/\/?\*+$/, '')
    .replace(/\/\.\.\.$/, '')
    .replace(/\/+$/, '')
}

// sliceTouchedBy reports whether any of `files` falls inside this slice, by
// prefix match in either direction (a slice path can be a dir prefix of a file,
// or a file path the slice owns directly).
function sliceTouchedBy(slice, files) {
  const ps = (slice.paths || []).map(normPath).filter(Boolean)
  return (files || []).some((f) =>
    ps.some(
      (p) => f === p || f.startsWith(p + '/') || p.startsWith(f + '/')
    )
  )
}

// ---- main loop ------------------------------------------------------------

const rounds = []
const allApplied = []
const allFollowUp = []
const allRejected = []

// seenFiles tracks which files prior rounds already fixed. When a round adds no
// NEW file to that set we count it as "stale": the loop is churning the same
// surface. Two stale rounds in a row trips the diminishing-returns stop.
const seenFiles = new Set()
let stale = 0
const STALE_LIMIT = 2

let round = 0
let converged = false
let stopReason = 'cap'

log(`Profile: ${profile} (cap ${iterCap} rounds, triage/plan on Sonnet)`)

while (round < iterCap && !converged) {
  // Budget gate: never start a round we can't pay for AND still afford verify.
  if (BUDGETED && budget.remaining() < VERIFY_RESERVE + PER_ROUND_EST) {
    log(
      `Budget low (${k(budget.remaining())} left, reserving ` +
      `${k(VERIFY_RESERVE)} for verify) — stopping rounds`
    )
    stopReason = 'budget'
    break
  }

  round += 1
  phase('Find')
  const budgetNote = BUDGETED ? ` (${k(budget.remaining())} budget left)` : ''
  const finderCount = prof.combineFinders ? 1 : lenses.length
  log(`Round ${round}: dispatching ${finderCount} finder(s)${budgetNote}`)

  // Carry earlier rejections forward so finders stop re-raising them.
  const findings = await runFinders(round, allRejected)
  if (findings.length === 0) {
    log(`Round ${round}: finders found nothing — clean round`)
    rounds.push({ round, found: 0, fixNow: 0 })
    converged = true
    stopReason = 'clean'
    break
  }

  phase('Triage')
  const verdict = await triage(findings, round)
  const fixNow = atOrAboveCutoff(verdict.fixNow)
  allFollowUp.push(...(verdict.followUp || []))
  allRejected.push(...(verdict.rejected || []))

  log(
    `Round ${round}: ${findings.length} raw → ${fixNow.length} fix-now, ` +
    `${(verdict.followUp || []).length} follow-up, ` +
    `${(verdict.rejected || []).length} rejected`
  )

  if (fixNow.length === 0) {
    rounds.push({ round, found: findings.length, fixNow: 0 })
    converged = true
    stopReason = 'clean'
    break
  }

  // Diminishing-returns tracking: did this round reach any NEW file?
  const newFiles = filesOf(fixNow).filter((f) => !seenFiles.has(f))
  if (newFiles.length === 0) {
    stale += 1
  } else {
    stale = 0
    newFiles.forEach((f) => seenFiles.add(f))
  }

  phase('Apply')
  const applied = await applyFixes(fixNow, round)
  allApplied.push(...(applied?.applied || []))
  rounds.push({
    round,
    found: findings.length,
    fixNow: fixNow.length,
    newFiles: newFiles.length,
  })

  if (stale >= STALE_LIMIT) {
    log(
      `Round ${round}: ${stale} rounds with no new files touched — ` +
      `diminishing returns, stopping loop for the cold verifier`
    )
    stopReason = 'diminishing-returns'
    break
  }
  // Otherwise loop: the next round re-runs finders on the updated surface.
}

// Completeness critic (thorough profile only): one strong pass for what the
// lens finders structurally could not see, triaged + applied like any round.
if (prof.critic) {
  round += 1
  phase('Find')
  log(`Round ${round}: completeness critic pass`)
  const critFindings = await criticPass(round)
  if (critFindings.length > 0) {
    phase('Triage')
    const verdict = await triage(critFindings, round)
    const fixNow = atOrAboveCutoff(verdict.fixNow)
    allFollowUp.push(...(verdict.followUp || []))
    allRejected.push(...(verdict.rejected || []))
    if (fixNow.length > 0) {
      phase('Apply')
      const applied = await applyFixes(fixNow, round)
      allApplied.push(...(applied?.applied || []))
    }
    rounds.push({
      round,
      found: critFindings.length,
      fixNow: fixNow.length,
      source: 'critic',
    })
  }
}

// Cold acceptance verifier. A clean find/triage round is necessary but not
// sufficient — fresh reviewers must approve the diff. The verify is chunked
// (one bounded slice per area) and non-fatal. A reopen feeds at most one
// verifier-driven repair pass, after which we re-verify ONLY the slices that
// reopened (incremental) rather than the whole diff again.
phase('Verify')
const plan = await planVerify()
const allSlices = (plan && plan.slices) || []
let verifier = await safeVerify(allSlices)

const canRepair = () =>
  verifier.decision === 'reopen' &&
  round < iterCap &&
  (!BUDGETED || budget.remaining() > VERIFY_RESERVE)

if (canRepair()) {
  round += 1
  log(`Verifier re-opened with ${verifier.findings?.length || 0} findings`)

  // Remember which slices reopened so we re-verify only those.
  const reopenedAreas = new Set(
    (verifier.sliceVerdicts || [])
      .filter((v) => v.decision === 'reopen')
      .map((v) => v.slice)
  )

  phase('Triage')
  const verdict = await triage(verifier.findings || [], round)
  const fixNow = atOrAboveCutoff(verdict.fixNow)
  allFollowUp.push(...(verdict.followUp || []))
  allRejected.push(...(verdict.rejected || []))

  let repairedFiles = []
  if (fixNow.length > 0) {
    phase('Apply')
    const applied = await applyFixes(fixNow, round)
    allApplied.push(...(applied?.applied || []))
    // Prefer the files the apply agent reports it actually changed; fall back
    // to triage's predicted files if it reported none.
    repairedFiles = (applied?.applied || []).flatMap((a) => a.files || [])
    if (repairedFiles.length === 0) repairedFiles = filesOf(fixNow)
  }
  rounds.push({
    round,
    found: verifier.findings?.length || 0,
    fixNow: fixNow.length,
    source: 'verifier',
  })

  // Incremental re-verify with a spillover guard: re-verify the slices that
  // reopened PLUS any previously-approved slice the repair actually edited into
  // (a fix in a reopened area can bleed into an approved one). When the repair
  // stays within the reopened areas — the common case — this is exactly the
  // reopened set, so the optimization still holds; the guard only widens on
  // genuine spillover.
  phase('Verify')
  await rematerialize()
  const target = allSlices.filter(
    (s) => reopenedAreas.has(s.area) || sliceTouchedBy(s, repairedFiles)
  )
  const effectiveTarget = target.length > 0 ? target : allSlices
  const spill = effectiveTarget.length - reopenedAreas.size
  log(
    `Incremental re-verify: ${effectiveTarget.length} of ${allSlices.length} ` +
    `slices (${reopenedAreas.size} reopened` +
    `${spill > 0 ? `, +${spill} touched by repair` : ''})`
  )
  verifier = await safeVerify(effectiveTarget)
}

return {
  profile,
  converged,
  stopReason,
  rounds,
  applied: allApplied,
  followUp: allFollowUp,
  rejected: allRejected,
  verdict: verifier.decision,
  verifierSummary: verifier.summary || '',
  hitMaxIters: round >= iterCap && verifier.decision !== 'approve',
  tokensSpent: BUDGETED ? budget.spent() : null,
}
