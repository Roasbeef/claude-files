# Arcs of Coherence — Stitching Sentences into a Train of Thought

From Ch. 5 "Arcs of Coherence" (with the structural frame from Ch. 4). Core
thesis: **a series of individually well-formed sentences can still be an
incoherent mess.** Coherence is not a property of the prose on the page — it's a
property of the *mental model* the prose builds in the reader's head. The writer's
job is to help the reader grasp the topic, get the point, keep track of the
players, and see how one idea follows from another.

## The web, the tree, and the string

Three representations are in play; confusing them is the root of incoherence.

- **The WEB** — the tangled network of associated ideas in the writer's mind.
  Non-linear, everything connected to everything.
- **The TREE** — the hierarchical structure of the content: topics nested in
  subtopics, the logical skeleton.
- **The STRING** — the actual linear sequence of words the reader experiences, one
  at a time, through a narrow working-memory bottleneck.

The writer holds the whole web and tree at once (the curse of knowledge). The
reader receives only the **string** and rebuilds the tree as they go. Coherence =
giving the reader enough cues at each step to reconstruct your tree from your
string, without ever losing their place.

Consequences:
- **Order is not neutral.** The reader can't see ahead; every sentence must make
  sense given only what came before it.
- **There is no algorithm.** Coherence requires judgment about the reader's state
  of knowledge.

## Given-before-new (the master ordering principle)

> **Topic, then comment. Given, then new. Light before heavy.**

Put information the reader *already has* (given/old) at the **front**, where it
links back to prior context, and put the **new** information at the **end**, where
it lands as the payload and becomes the "given" for the next sentence. This
minimizes memory load: the reader anchors a new fact onto an already-active mental
peg instead of holding an unanchored fact in suspense.

This drives three devices:

### Clause / event order
"She showered before she ate" is easier than "She ate after she showered" — the
first orders events the way the reader builds them.

### The passive voice — a legitimate tool
Contra the "avoid passive" dogma, the passive exists precisely to serve
given-before-new: it lets a writer postpone a doer that is heavy, old news, or
both. Use it when the affected entity is the running topic, or when the agent is
unknown/irrelevant/heavy.

**Before (active, topic-discontinuous):** *A spike in cortisol triggers the
amygdala.* — but the amygdala was the topic of the whole paragraph, buried here as
the object.
**After (passive, given-first):** *The amygdala is triggered by a spike in
cortisol.* — the known player is the subject; the new cause lands last.

(Full passive decision rule in `mechanics.md`.)

### Topicalization / preposing
Move a phrase to the front to mark it as the topic, even when it's grammatically
an object:
Before: *I will never forget the day I first saw the Pacific.*
After: *That day — the day I first saw the Pacific — I will never forget.*

## Topic strings — keep one consistent subject

Let the reader in on the topic early, then keep a **consistent subject in subject
position** so the reader tracks one protagonist instead of re-orienting every
sentence. Shifting grammatical subjects forces the reader to re-establish "who are
we talking about now?" each time — a hidden tax.

**Before (subject lurches around — incoherent despite fine sentences):**
> The mitochondrion produces ATP. Cellular respiration is the process involved.
> Oxygen is consumed by this reaction. A proton gradient drives the synthase.

Four subjects in four sentences; the reader has no thread.

**After (consistent topic string):**
> The mitochondrion produces ATP through cellular respiration. It consumes oxygen
> in the process, and it uses the resulting proton gradient to drive ATP synthase.

The mitochondrion (then *it*) holds subject position throughout; new info is
appended at each predicate.

> Use pronouns or **repeat the noun** — do NOT swap in synonyms ("the organelle,"
> "the powerhouse") to seem varied. Consistent wording aids coherence; elegant
> variation harms it.

## Coherence relations — make the link explicit

A coherent text is one in which **the reader always knows which relation holds
between one sentence and the next.** These relations are components of *reason*,
not just language. Pinker groups them under Hume's three heads: resemblance,
contiguity, and cause-and-effect.

| Relation | What it does | Signal connectives |
|---|---|---|
| **Similarity** | Second claim parallels the first | *and, similarly, likewise, too, also* |
| **Contrast** | Alike in most ways, differ in one | *but, in contrast, on the other hand, whereas* |
| **Elaboration** | Restate the same content in more detail | *that is, in other words, namely* (often a **colon**) |
| **Exemplification** | Give an instance of a generalization | *for example, for instance, such as* (often a **colon**) |
| **Generalization** | Abstract a rule from instances | *in general, more generally, broadly* |
| **Exception** | Instance that violates the generalization | *however, except, then again* |
| **Sequence / Enablement** | One event sets up or precedes the next | *then, next, before, after, and then* |
| **Result** (cause→effect order) | First clause causes the second | *so, as a result, therefore, thus, consequently, which is why* |
| **Cause / Explanation** (effect→cause order) | Second clause causes the first | *because, since, owing to, as, for* |
| **Violated expectation** | A cause that fails to produce its effect | *but, nevertheless, nonetheless, yet, despite, although* |
| **Attribution** | Marks a claim as someone's belief, not the writer's | *according to, X claims that, supposedly, reportedly* |

Two craft notes:
- **Similarity/contrast can be signaled without a connective** — use parallel
  syntax and vary only the contrasting word ("Some came to bury Caesar; others
  came to praise him").
- **Elaboration and exemplification are what the colon is for** (when tempted to
  say *that is, in other words, for example, voilà*).

## Connectives matter — too few is the real danger

Misguided "brevity" edits strip out *however / therefore / but / because* to cut
word count. **This destroys coherence**, because those words *are* the coherence
relations made visible. Remove them and the reader must guess how each sentence
relates to the last — and often guesses wrong.

> Tied to the curse of knowledge: the relation is obvious *to the writer* (who
> holds the whole web), not to the reader (who has only the string). The
> systematic error runs toward **under**-connecting. **When in doubt, connect.**

**Before (connectives stripped — relations ambiguous):**
> The cache was cold. Latency tripled. We added a warm-up phase. p99 stayed flat.

Is sentence 2 a result or a coincidence? Is sentence 4 a result of 3, or a
*violated* expectation?

**After (connectives restored):**
> The cache was cold, **so** latency tripled. We **therefore** added a warm-up
> phase — **but** p99 stayed flat.

The *but* signals violated expectation: the fix didn't work, which is the whole
point.

Caveat: too many connectives belabor the obvious and patronize the reader.
Calibrate to the audience — and, since you can't calibrate from inside your own
head, show a draft to representative readers.

## A consistent point, and the negation problem

### Consistent point
Coherence builds over many paragraphs from the author's grasp of the text as a
whole. The reader continuously asks *what is the point of all this?* — are you
explaining a topic, conveying new facts, advancing an argument, or illustrating a
generalization? **Tell them early which one it is.** Locally perfect transitions
with no governing point still read as incoherent.

### The negation problem
Believing a proposition true costs nothing beyond understanding it; believing it
false requires adding and holding a mental "not" tag. So "The king is not dead" is
harder than "The king is alive." Worse, readers tend to **remember the asserted
proposition and forget the negation tag** — so "X is *not* Y" can leave them
believing "X is Y." Rules:
- **Prefer affirmation.** State what *is*, not what *isn't*. (Kennedy: "We choose
  to go to the moon **not because it is easy but because it is hard**" beats an
  all-negative version.)
- **Count hidden negations.** Many negators don't start with *n*: *few, little,
  seldom, rarely, instead, doubt, deny, avoid, ignore, fail to, absent, lack.*
  Stacking them overwhelms.
- **Negate only plausible propositions.** Negation is easy when you deny something
  the reader already believed ("Contrary to the common belief that X, in fact Y").
  Negating an idea the reader never entertained just plants it (the "white bear"
  effect).

## Paragraphing and signposting (reconciling anti-signposting with pro-coherence)

This looks contradictory — Pinker mocks heavy signposting yet demands coherence —
but resolves cleanly:

- **"There is no such thing as a paragraph."** No outline node reliably
  corresponds to a block of text. What exists is the **paragraph break** — a visual
  bookmark that lets the reader pause, breathe, assimilate, and find their place
  again. Break paragraphs to rest the reader and mark a shift in the discourse tree,
  not by a rigid "one idea per paragraph" rule.
- **Coherence ≠ signposting.** The good way to show how parts relate is
  *intrinsic*: consistent topic strings, given-before-new ordering, the right
  connective, parallel syntax.
- **The bad, heavy signposting** is metadiscourse / GPS-narration: *"In this
  section I will discuss three points. The first point is… Having discussed the
  first point, I now turn to the second…"* That narrates the table of contents
  instead of conveying content; it's a *substitute* for coherence, not coherence.

> **Reconciliation.** Make the *connections between ideas* explicit (connectives,
> topic continuity, clear relations) — that's real coherence. Don't make the
> *structure of the document* explicit through clunky meta-commentary. Show the
> arcs of coherence; don't announce the outline. Exception: genuinely tree-like,
> navigational documents (reference manuals, specs) legitimately use headings and
> explicit structure, because there the reader *is* navigating a tree non-linearly
> rather than reading a string.

## Quick-reference

1. Reader sees only the linear string; you hold the whole web — compensate.
2. Given before new; light before heavy; topic then comment.
3. Use the passive (or topicalization) when it puts the given element first.
4. Hold one consistent subject across a passage; pronouns/repetition, not synonyms.
5. Make the relation between every adjacent sentence unambiguous.
6. When in doubt, add the connective — under-connecting is the systematic error.
7. Strip metadiscourse ("In this section…"), but keep logical connectives.
8. Prefer affirmatives; hunt hidden negatives; negate only what the reader believes.
9. Keep one consistent point per arc; tell the reader early what kind of text it is.
10. Break paragraphs to rest the reader and mark tree shifts — not by formula.
