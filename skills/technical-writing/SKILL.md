---
name: technical-writing
description: Clear-writing guide distilled from Steven Pinker's "The Sense of Style." Use when writing or revising prose that must be clear to a reader — documentation, design docs, specs, explanations, essays, emails, reports, RFCs, release notes — or when asked to make writing clearer, tighter, less academic, or less jargon-laden. Activate for "make this clearer", "tighten this", "why is this hard to read", "edit this for clarity", or any prose-quality pass.
---

# Technical Writing (The Sense of Style)

A practical guide to clear prose, distilled from Steven Pinker's *The Sense of
Style: The Thinking Person's Guide to Writing in the 21st Century* (2014). The
whole book reduces to one image and one cause, with four sets of consequences.

Use this when **drafting** prose (write it right the first time) or **revising**
prose (run the revision pass at the bottom). The reference files under
`references/` hold the dense tables and examples; load the one you need when a
section says so.

## The one image: prose is a window onto the world

Classic style treats writing as a **clean window**: the writer has seen
something real and aims the reader's gaze at it. The reader is a **competent
equal** — smart, not a student to be lectured, not a judge to be appeased. The
writer has nothing to prove.

Everything that makes prose bad is a **smudge on the glass** — a place where the
words point at *themselves*, at the *writer's anxiety*, or at an *abstraction*,
instead of at the thing in the world.

> **Master heuristic.** For any sentence ask: *Is this a clean window onto a
> thing in the world, with a writer showing a reader something — or has it
> fogged into talk about the text, the writer's caution, or a concept about a
> concept?* Fix toward the window.

Write in classic style by default. Break it **locally and deliberately** — for a
real legal caveat, a genuine scientific limitation, or honest uncertainty — never
out of habit or insecurity. Details and the style taxonomy:
`references/classic-style.md`.

## The one cause: the curse of knowledge

The single best explanation for why good people write bad prose: **you can't
imagine what it's like not to know what you already know.** It is a cognitive
blind spot, not malice or laziness (Hanlon's razor applies). Because it operates
*beneath your awareness*, the feeling that something is "obvious" or "clear" is
the **symptom, not a defense**.

It surfaces as:
- **Unexplained jargon and acronyms** — spell out every acronym on first use.
- **Functional fixity** — naming the technical role instead of showing the thing
  ("an assessment word" vs. "the word TRUE or FALSE"). Show the thing.
- **Chunking** — stacking expert abstractions the reader can't unpack. Unpack one
  level.
- **Skipped steps** — the "obvious" intermediate the reader can't divine. Include it.

The reliable fixes are **external**, because you can't self-debug a blind spot.
"Just imagine your reader" *fails*. In order of reliability: show a draft to a
real reader like your audience → put it in a drawer and reread later as a stranger
→ read it aloud. Be clear without being condescending. Details:
`references/curse-of-knowledge.md`.

## Consequence 1 — coherence: stitch sentences into a train of thought

The reader receives only the linear **string** of words and rebuilds your **tree**
of ideas. Individually perfect sentences can still be an incoherent mess. Manage
the reader's mental model:

- **Given before new. Topic before comment. Light before heavy.** Anchor each new
  fact to something already in the reader's mind, placed at the *front*; land the
  new payload at the *end*.
- **Hold one consistent topic** in subject position across a passage. Use pronouns
  or repeat the noun — do **not** swap in synonyms ("the organelle", "the
  powerhouse") to seem varied.
- **Make the relation between adjacent sentences unambiguous.** *When in doubt,
  connect* — under-connecting (stripping *but / because / so / however* for false
  "brevity") is the systematic error, because the relation is obvious to you and
  not to the reader.
- **Prefer affirmatives.** Readers remember the proposition and drop the "not"
  tag; "X is not dead" can leave them believing "X is dead." Negate only what the
  reader was already inclined to believe.

Coherence-relation table, connectives, and rewrites: `references/coherence.md`.

## Consequence 2 — syntax: build sentences the reader's memory can parse

A sentence is a tree; the reader receives a string and reconstructs the tree one
word at a time through a narrow memory bottleneck. Lighten the load:

- **Right-branch by default** — hang the heaviest, most complex phrase at the
  *end* (end-weight). Front-loaded subordinate clauses hold the main clause hostage.
- **Never center-embed** — don't jam a clause into the middle of another. Fasten
  it to an edge or split the sentence.
- **Keep subject and verb close** — don't wedge long material between tightly
  bound words.
- **Kill garden paths** — restore the *that / which / who* you deleted for
  "brevity" when its absence invites a wrong first reading.
- **Parallel meaning → parallel syntax.** Don't vary structure capriciously across
  a list or coordination.
- **Explode noun-piles** (≤ 2–3 modifiers before a head noun) with prepositions
  and relative clauses.

Universal escape hatch: **when a sentence goes unruly, split it in two.** Garden
paths, end-weight, and examples: `references/syntax.md`.

## Consequence 3 — words and mechanics: the surface craft

- **Reverbify zombie nouns.** Nominalizations embalm verbs into lifeless nouns:
  `the cancellation of X` → `cancel X`; `make an appearance` → `appear`. Keep one
  only when it names an already-introduced topic so the next sentence can comment
  on it.
- **The passive is a legitimate tool — never auto-convert it.** Use it when the
  affected entity is the running topic, or when the agent is unknown, irrelevant,
  or heavy. It's wrong only when it hides a responsible agent ("mistakes were
  made") or breaks topic flow.
- **Cut compulsive hedges** (*somewhat, fairly, apparently, in part,
  presumably*) — but keep a qualifier that carries real scope. **Qualify, don't
  hedge:** spell out the actual condition under which the claim fails instead of
  sprinkling vague escape-words.
- **Strip metaconcepts** — concepts about concepts (*level, framework, process,
  perspective, approach, model, issue*). Name the act or object they wrap.
- **Trim officialese** — `for the purpose of` → `to`, `at this point in time` →
  `now`. Substitution table in the reference.
- **Intensifiers are disguised hedges** — prefer one vivid word: `very big` →
  `huge`. Cut tics (*actually, basically, really*).

Full tables, the passive decision rule, punctuation-as-parsing-aid:
`references/mechanics.md`.

## Consequence 4 — usage: tell real rules from superstitions

Not "anything goes," but reasoned judgment. A rule earns its keep only if it
serves the reader. **Don't enforce the superstitions:** split infinitives,
terminal prepositions, sentence-initial *and/but/because*, singular *they*,
restrictive *which*, *who* for *whom*, sentence-adverb *hopefully*, *less* with
measurements, the blanket passive-voice ban. **Do enforce the real ones:**
*its/it's*, parallel structure, subject-verb agreement, ambiguous pronoun
reference, dangling modifiers *that cause real ambiguity*, *literally*, true
malaprops (*lie/lay*, *flout/flaunt*).

Decision order when a dispute arises: **clarity first → consult data, not dogma →
diagnose the rule's pedigree (Latin-aping or schoolroom myth ⇒ ignore) → know
your audience** (in a sticklerish context, observe even a superstition to avoid
distracting the reader). Full myth-bust and real-rules tables:
`references/usage.md`.

## The revision pass

When revising existing prose, run these in order. Each maps to a reference file.

1. **Window check (whole draft).** Where does the prose point at itself, the
   writer's caution, or an abstraction instead of the world? Cut metadiscourse
   ("In this section I will…"), apologies, professional narcissism. → classic-style
2. **Curse-of-knowledge sweep.** Spell out every acronym on first use. Replace
   functional labels with the concrete thing. Unpack stacked abstractions. Add the
   skipped "obvious" step. Where possible, get a real reader — or set it aside and
   reread as a stranger. → curse-of-knowledge
3. **Coherence sweep.** Check given-before-new ordering; one consistent topic per
   passage; an unambiguous connective between adjacent sentences; affirmatives over
   negations. → coherence
4. **Syntax sweep.** Find garden paths, center-embedding, subject–verb gaps,
   front-loaded heavy clauses, non-parallel lists, noun-piles. Split unruly
   sentences. → syntax
5. **Word/mechanics sweep.** Reverbify zombie nouns; justify or rewrite each
   passive; cut hedges and intensifiers; strip metaconcepts; apply the
   officialese→plain table. → mechanics
6. **Usage sweep.** Fix the real errors; stop "correcting" the superstitions;
   resolve disputes by clarity + audience. → usage

Don't apply the rules as a mechanical checklist over the master heuristic — the
window comes first. A rule yields whenever following it would fog the glass.
