export const meta = {
  name: 'review-loop',
  description:
    'Adversarial review→triage→fix loop until a cold verifier signs off',
  whenToUse:
    'Invoked by the /review-loop skill after it resolves scope and writes a ' +
    'design brief. Fans out lens-specific adversarial reviewers, triages ' +
    'findings against the code, applies confirmed fixes as fixup commits, ' +
    'loops on the new surface, and ends only when a cold verifier approves.',
  phases: [
    { title: 'Find', detail: 'one adversarial reviewer per lens' },
    { title: 'Triage', detail: 'verify, dedup, kill false positives' },
    { title: 'Apply', detail: 'implement confirmed fixes as fixup commits' },
    { title: 'Verify', detail: 'cold sign-off on the full final diff' },
  ],
}

// args (from the skill's Phase 0):
//   diffCmd  - exact git diff command defining the review surface
//   base     - base branch (for fixup targeting and the final diff)
//   brief    - contents of .review-loop/brief.md (intent + constraints)
//   lenses   - [{ key, agentType, hunt }] selected from the changed files
//   cutoff   - 'high' | 'medium' | 'low' severity floor for in-loop fixes
//   maxIters - safety cap on rounds (not a target)
const {
  diffCmd,
  base = 'main',
  brief = '',
  lenses = [],
  cutoff = 'medium',
  maxIters = 5,
} = args || {}

// Severity ordering so we can compare against the cutoff floor.
const SEV = { critical: 4, high: 3, medium: 2, low: 1, info: 0 }
const floor = SEV[cutoff] ?? 2

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

// finderPrompt builds the adversarial prompt for one lens. Every finder
// reviews the same surface and gets the same brief, differing only in lens.
function finderPrompt(lens, round) {
  return `You are an ADVERSARIAL reviewer (round ${round}). BREAK this change,
do not grade it. Only report findings you can argue concretely from the code.

Scope — review exactly this surface:
  ${diffCmd}

Design brief (intent + hard constraints; respect accepted tradeoffs and the
pre-flight baseline noted here — do not report pre-existing breakage):
${brief}

Your lens: ${lens.key}
Hunt specifically for: ${lens.hunt}

For each finding give a stable id (prefix with "${lens.key}-"), file:line,
severity, a concrete trigger SCENARIO (the exact sequence that causes the
failure), and a minimal fix sketch. If a suspicion turns out safe after you
read the code, omit it (or note it as verified-safe). No false positives —
triage will verify and reject anything you cannot substantiate.`
}

// runFinders fans out one adversarial reviewer per lens for a round.
async function runFinders(round) {
  const results = await parallel(
    lenses.map((lens) => () =>
      agent(finderPrompt(lens, round), {
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

// triage verifies findings against the code, dedups, and classifies.
async function triage(findings, round) {
  const list = JSON.stringify(findings, null, 2)

  return agent(
    `You are the triage judge (round ${round}). You have read access to the
code. Verify every finding below against the actual cited lines.

Scope: ${diffCmd}
Design brief:
${brief}

Findings from the adversarial finders:
${list}

Do all of:
1. VERIFY each finding by re-reading the cited code. Reject anything you cannot
   reproduce from the source, or that the brief marks as accepted/out-of-scope/
   pre-existing.
2. DEDUP across lenses; merge complementary findings into one stronger item.
3. CLASSIFY survivors:
   - fixNow: severity at or above "${cutoff}". Give a concrete, repo-style fix
     sketch, the files touched, and the commit sha the fix should fix up
     (fixupTarget) if identifiable from git blame / the diff.
   - followUp: real but deferrable (below cutoff, or needs a design decision).
   - rejected: false positive or not worth it, with the reason.
Be decisive. A confirmed false positive is a success, not a gap.`,
    {
      label: `triage:r${round}`,
      phase: 'Triage',
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

Return one applied entry per fix with its commit sha and status.`,
    {
      label: `apply:r${round}`,
      phase: 'Apply',
      schema: APPLY_SCHEMA,
    }
  )
}

// coldVerify spawns a fresh reviewer that has seen none of the rounds and asks
// it to independently approve or re-open the full final diff.
async function coldVerify() {
  return agent(
    `Read this complete change cold — you have seen none of the prior review.
Decide whether it meets the bar to merge. Hunt for anything iterative reviewers
may have grown blind to.

Full diff: git diff ${base}..HEAD
Design brief:
${brief}

Return decision "approve", or "reopen" with concrete findings (id, file:line,
severity, scenario, fix). When in doubt, reopen.`,
    {
      label: 'verify:cold',
      phase: 'Verify',
      agentType: 'code-reviewer',
      schema: VERDICT_SCHEMA,
    }
  )
}

// atOrAboveCutoff keeps only fix-now items at or above the severity floor.
function atOrAboveCutoff(fixNow) {
  return (fixNow || []).filter((f) => (SEV[f.severity] ?? 0) >= floor)
}

// ---- main loop ------------------------------------------------------------

const rounds = []
const allApplied = []
const allFollowUp = []
const allRejected = []

let round = 0
let converged = false

while (round < maxIters && !converged) {
  round += 1
  phase('Find')
  log(`Round ${round}: dispatching ${lenses.length} adversarial finders`)

  const findings = await runFinders(round)
  if (findings.length === 0) {
    log(`Round ${round}: finders found nothing — clean round`)
    rounds.push({ round, found: 0, fixNow: 0 })
    converged = true
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
    break
  }

  phase('Apply')
  const applied = await applyFixes(fixNow, round)
  allApplied.push(...(applied?.applied || []))
  rounds.push({ round, found: findings.length, fixNow: fixNow.length })
  // Loop: the next round re-runs finders on the updated surface.
}

// Cold acceptance verifier. A clean find/triage round is necessary but not
// sufficient — a fresh reviewer must approve the full diff. A reopen feeds one
// more triage→apply round (bounded by maxIters via the loop above already
// having run; we do at most one verifier-driven repair pass here to stay
// bounded and deterministic).
phase('Verify')
let verifier = await coldVerify()

if (verifier.decision === 'reopen' && round < maxIters) {
  round += 1
  log(`Verifier re-opened with ${verifier.findings?.length || 0} findings`)

  phase('Triage')
  const verdict = await triage(verifier.findings || [], round)
  const fixNow = atOrAboveCutoff(verdict.fixNow)
  allFollowUp.push(...(verdict.followUp || []))
  allRejected.push(...(verdict.rejected || []))

  if (fixNow.length > 0) {
    phase('Apply')
    const applied = await applyFixes(fixNow, round)
    allApplied.push(...(applied?.applied || []))
  }
  rounds.push({ round, found: verifier.findings?.length || 0,
    fixNow: fixNow.length, source: 'verifier' })

  phase('Verify')
  verifier = await coldVerify()
}

return {
  converged,
  rounds,
  applied: allApplied,
  followUp: allFollowUp,
  rejected: allRejected,
  verdict: verifier.decision,
  verifierSummary: verifier.summary || '',
  hitMaxIters: round >= maxIters && verifier.decision !== 'approve',
}
