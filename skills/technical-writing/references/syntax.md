# Syntax — The Web, the Tree, and the String

From Ch. 4. Pinker's core linguistic argument: clear syntax is not etiquette — it
is engineering for the reader's brain. Every rule derives from one goal:
**structure sentences so they cooperate with the reader's limited working memory
and incremental, left-to-right parsing.**

## The model: web → tree → string

> "Syntax is an app that uses a tree of phrases to translate a web of thoughts into
> a string of words."

- **The web** — the non-linear network of ideas in the writer's mind.
- **The tree** — the hierarchical syntactic structure that groups words into
  phrases, embeds phrases in larger phrases, and encodes who-did-what-to-whom. The
  bridge.
- **The string** — the linear, one-word-at-a-time sequence the reader receives.

The central asymmetry: the writer *has* the tree; the reader receives **only the
string** and must reconstruct the tree from it, incrementally, before grasping the
meaning.

> **Principle.** Good syntax helps the reader recover the intended tree from the
> string on the first pass, without backtracking or guessing. Bad syntax forces
> the reader to build the wrong tree, hold too many unfinished branches, or rebuild
> from scratch.

## The reader's limited working memory

The reader parses incrementally — each word slots into a growing tree as it
arrives. Whenever a branch is opened but not yet completed, the reader must hold
the incomplete branch in memory until it resolves. Working memory is small; the
cost of a sentence is roughly **how many unresolved branches the reader juggles at
once** and **how long** each is held.

Sources of load: unresolved dependencies held open (a subject waiting for its
verb), stacked modifiers before a head noun, and center-embedding (the worst case).

> **Principle.** Minimize both the *number* of branches held open at once and the
> *distance* over which each is held. A sentence is hard not because it's long, but
> because it forces the reader to keep too much suspended before resolution.

## Garden-path sentences

A garden path has a **temporary ambiguity**: early words can be parsed more than
one way, the reader commits to the wrong tree, then must backtrack when later words
rule it out.

Two parsing defaults cause them:
1. **Frequency bias** — for an ambiguous word, readers grab the most frequent
   meaning first ("hunts ducks" binds as verb + object).
2. **Low-attachment bias** — for a phrase that could attach in two places, readers
   attach it low (to the nearest word).

Why writers produce them and speakers don't: speech carries **prosody** (pauses,
stress) that disambiguates in real time. Writing strips prosody away.

| Before (garden path) | Why it trips | After (fixed) |
|---|---|---|
| The man who hunts ducks out on weekends. | "hunts ducks" reads as verb+object | The man who hunts **slips** out on weekends. |
| The horse raced past the barn fell. | "raced" read as main verb, not reduced passive | The horse **that was** raced past the barn fell. |
| Fat people eat accumulates in their cells. | "Fat people" reads as a noun phrase | **The** fat **that** people eat accumulates in their cells. |
| The cotton clothing is usually made of grows in Mississippi. | "cotton clothing" binds as a compound | The cotton **that** clothing is usually made of grows in Mississippi. |

> **Scan** for any early sequence that invites a frequent-but-wrong reading. Break
> it by inserting a function word, repunctuating, or reordering so the misleading
> sequence never forms.

## Keep function words; use punctuation

Writers delete optional function words (*that, which, who, of*) for brevity. Fine
*only* when it doesn't open a garden path. When omission lets two content words fuse
into a wrong reading, **restore the function word** — it signposts where the branch
begins.

- **"that"** marks a clause boundary: "The cotton **that** clothing is made of…"
  prevents "cotton clothing" fusing.
- **"who/which"** marks a relative clause and pins the antecedent.

**Punctuation substitutes for prosody.** Commas, dashes, colons, and parentheses
re-insert the pauses speech would provide, fencing off phrases so the reader
doesn't merge them.

Before: *The data scientists collected proved unreliable.*
After: *The data **that** scientists collected proved unreliable.*

> Brevity is not free. Drop a function word only after confirming the remaining
> string has exactly one early parse. When in doubt, keep *that/which/who*.

## Right-branching, left-branching, center-embedding

Where you hang the complex material determines the memory cost.

### Right-branching (English's preference — DO THIS)
The complicated phrase hangs off the rightmost branch, at the **end**. By the time
the reader reaches it, earlier phrases are already resolved and discharged from
memory, so full attention is free for the hard part.

> A right-branching sentence works like a train: a powerful locomotive (the main
> clause) pulling any number of cars (modifiers) behind it.

### Left-branching (use sparingly)
Modifiers pile up *before* the head, so the reader stores them all without yet
knowing what they modify. Fine in short doses ("those 15 giggling girls") and for
jokes (setup before punchline), but dangerous when stacked. Chief offender: the
**noun pile**.

### Center-embedding (AVOID — worst case)
A phrase jammed into the *middle* of a larger phrase. Multiple branches stay
suspended at once, overflowing memory.
- *The rat the cat the dog chased killed ate the malt.* — grammatical, cognitively
  impossible.

**Fix:** disinter each embedded clause and place it side by side with the clause
that contained it. Often: split the sentence in two.

### End-weight ("save the heaviest for last")
Heavy/long/complex constituents go at the **end**; light ones go first. Three
forces converge: English syntax wants subject before object; memory wants light
before heavy; cognition wants topic before comment and given before new.

| Before (heavy-first, taxing) | After (heavy-last) |
|---|---|
| That the proposal we submitted last quarter after months of negotiation was rejected surprised no one. | **No one was surprised that** the proposal we submitted last quarter after months of negotiation was rejected. |
| A report describing every defect found during the audit and proposing remediation steps for each is attached. | **Attached is** a report describing every defect found during the audit and proposing remediation steps for each. |

> Reorder so the one long constituent comes last. Front-loaded subordinate clauses
> ("That…", "Because…", "Although the committee, which had…") tax memory by holding
> the main clause hostage; trail them, or split.

## Keep subject and verb close

The subject opens a dependency that stays open until the verb arrives. Long
material wedged between keeps that branch suspended; the reader may lose the
subject entirely.

| Before (subject and verb separated) | After (close together) |
|---|---|
| The **proposal**, which the committee had debated for three contentious meetings spanning most of a fiscal quarter, **was rejected.** | The committee debated the proposal for three contentious meetings over most of the quarter. In the end, the **proposal was rejected.** |
| Our new **caching layer**, designed to reduce database load during spikes and built on a distributed in-memory store replicated across three regions, **improved latency by 40%.** | Our new **caching layer improved latency by 40%.** It reduces database load during spikes using a distributed in-memory store replicated across three regions. |

> Don't separate a subject from its verb (or any two tightly-bound words) with long
> intervening material. Move it before the subject or after the verb, or split.

## Structural parallelism

When ideas are coordinate or parallel in meaning, give them **parallel syntax**.
The reader parses the shape once, then reuses the template for each later item —
slashing memory load. Breaking parallelism forces a fresh re-parse per item and can
trigger garden paths.

> Pinker's rule: do not vary sentence structure capriciously. Constant structure
> for parallel content; vary it only to signal a genuine change in meaning.

| Before (non-parallel) | After (parallel) |
|---|---|
| The service validates the input, transformation of the payload, and then it will write to the queue. | The service **validates** the input, **transforms** the payload, and **writes** to the queue. |
| She liked swimming, to hike, and biked on weekends. | She liked **swimming, hiking, and biking** on weekends. |
| The function checks for nulls, whether the index is in range, and validating the checksum. | The function checks **for nulls, for an in-range index, and for a valid checksum.** |

## Noun-piles and ambiguous attachment

**Noun piles** are long left-branching stacks of nouns-modifying-nouns. The reader
must hold every modifier in memory before reaching the head noun, *and* the
attachment relationships are ambiguous (which noun modifies which?). Endemic to
headlines, product names, and jargon.

Two failure modes: memory (modifiers buffered before the head appears) and
ambiguous attachment (low-attachment bias makes the reader guess, often wrong).

| Before (noun pile) | After (unpacked with function words) |
|---|---|
| the police-involved shootings | the shootings **involving** police |
| a server-side request forgery vulnerability scan report | a report **on scanning for** server-side request-forgery vulnerabilities |
| customer account fraud detection rule engine | the rule engine **that detects fraud in** customer accounts |

> Cap modifier stacks at ~2–3 before a head noun. Break longer piles apart with
> prepositions and relative clauses (*of, for, that, involving*) that make
> attachment explicit and let the head noun arrive sooner.

## The encodable rule set

1. **Tree-recoverability** — can the reader build the right tree in one left-to-right
   pass? If not, fix.
2. **Garden-path scan** — does any early sequence invite a frequent-but-wrong
   reading? Insert *that/which/who*, repunctuate, or reorder.
3. **Don't over-prune function words** — drop *that/which/who* only if exactly one
   early parse remains.
4. **Right-branch by default** — heaviest/most complex constituent at the end.
5. **No center-embedding** — fasten clauses to an edge, or split.
6. **Subject ↔ verb proximity** — don't wedge long material between bound words.
7. **Order conventions** — topic→comment, given→new, light→heavy, short→long.
8. **Parallel meaning → parallel syntax** — don't vary structure capriciously.
9. **Cap modifier stacks (~2–3)** — explode noun piles with prepositions/relative
   clauses.
10. **When in doubt, split the sentence in two** — the universal escape hatch.
