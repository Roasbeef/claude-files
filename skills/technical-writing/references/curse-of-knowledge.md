# The Curse of Knowledge — The Chief Cause of Bad Prose

From Ch. 3 of *The Sense of Style*. Pinker calls this "the single best explanation
I know of why good people write bad prose."

## The definition

**The curse of knowledge: the difficulty of imagining what it's like for someone
else not to know something you already know.**

It is *insidious* because it hides not only the *contents* of your thoughts but
their very *form* — when you know something well, you don't realize how abstractly
you've come to think about it. You are the last person to notice your own jargon,
missing steps, and assumed context.

> **Rule.** Treat the writer's sense of "this is obvious / clear / standard" as an
> *unreliable signal*. Obviousness-to-the-author is the symptom, not a defense.

## Why it beats every other explanation

Pinker rejects the usual villains:

| Common theory | Why it's wrong |
|---|---|
| **Malice / deliberate obfuscation** | "They are not trying to bamboozle us — that's just the way they think." |
| **Status / showing off** | The same blindness appears when writing for peers, where no status game helps. |
| **Laziness / stupidity** | It afflicts *good, smart, careful* people most, because expertise is the cause. |

This is **Hanlon's razor** applied to prose: never attribute to malice what a
cognitive blind spot explains. The fix is therefore always "surface the hidden
thing," never "try harder to care."

## The symptoms (your detectors)

Two cognitive mechanisms — **chunking** and **functional fixity** — plus their
surface effects.

### Unexplained jargon, acronyms, abbreviations
The writer has mastered the "argot of her guild" and forgets the reader hasn't.
Produces alphabet soup.

**Before:** "We ran the TX through the LN node's HTLC switch; the CLTV delta
triggered an on-chain sweep via the UTXO set."
**After:** "We sent the payment through the routing node. Because the payment's
time-lock window was about to expire, the node had to reclaim the funds on the
blockchain itself rather than off-chain."
*(or define each term on first use: "…through the HTLC switch (the component that
forwards in-flight payments)…")*

### Functional fixity — naming the concept instead of showing the thing
Experts understand objects through *how they habitually use them*, so they write
the functional label instead of the concrete thing. Pinker's signature example:

**Before:** "Participants read assertions whose veracity was either affirmed or
denied by the subsequent presentation of an *assessment word*."
**After:** "Participants read sentences, each followed by the word *TRUE* or
*FALSE*."

Note the plain version is **shorter *and* clearer**. "Assessment word" named the
role the word played, not the word itself.

**Technical analog —**
Before: "The system emits a *liveness indicator* at a configurable cadence."
After: "Every 30 seconds, the server sends a heartbeat message."

### Chunking — compressing ideas into abstract chunks the reader can't unpack
Experts stack chunks into ever-higher abstractions to save mental space. **My
chunks are not your chunks.** How much abstraction you can use depends entirely on
the reader's expertise.

**Before:** "We applied standard reconciliation to converge divergent replica
state under partition."
**After:** "When two servers fall out of sync (for example, because the network
split them apart), we compare their records field by field and keep the newer
value for each — so both end up with the same data."

The before-sentence stacks three expert chunks; the after-sentence unpacks one
level.

### Assuming shared context the reader lacks
Skipping "intermediate steps that seem too obvious to mention" and scenes the
writer can visualize but the reader can't.

**Before:** "Initialize the daemon with the appropriate flags before invoking the
RPC."
**After:** "Start the daemon with `--rpclisten=0.0.0.0:10009` (this opens the port
the next command connects to). Then run the RPC call below."

### Nominalization / zombie nouns (abstraction made grammatical)
Chunking shows up as abstract nouns that smother actors and actions.
Before: "Cancellation of the subscription results in the revocation of access
privileges."
After: "When you cancel your subscription, you lose access."
(See `mechanics.md`.)

## The remedies, ranked by reliability

### ❌ "Just imagine your reader" — mostly FAILS
The most tempting remedy is the weakest. We're poor at figuring out what others are
thinking even when we try hard, because the curse operates *beneath awareness* —
you can't imagine the gap you can't see. Never rely on "picture your audience."
Use external, mechanical checks instead.

### ✅ Show a draft to a real, representative reader — MOST RELIABLE
The primary remedy. Show it to someone like your intended audience and find out
whether they can follow it; if not, revise. "You'd often be surprised to find that
what's obvious to you is not obvious to anyone else." This replaces unreliable
imagination with real signal.

### ✅ Put the draft away, then reread later as a stranger
Set the work aside **until it feels unfamiliar**, then revise with fresh eyes.
Time partially restores your prior ignorance. (Use when no real reader is
available.)

### ✅ Read it aloud
Reading slowly and literally helps you **see through the words to what they
represent**. A complementary exercise: take writing you admire and reverse-engineer
how it was built, to internalize clarity.

### ✅ Concrete language, examples, define terms on first use
Adopt classic style: point to things in the world the reader can see. Add "for
example" and show the thing. Define every term on first use. Spell out every
acronym the first time unless certain *every* reader knows it.

### ✅ Minimize abstraction / nominalization
Convert zombie nouns back into verbs and name the actors. Replace functional
labels with the concrete thing they label.

**Two caveats:**
- **Don't confuse clarity with condescension.** Explaining something doesn't
  insult the reader.
- **…but don't overcorrect into "motherese"** — the grating, I-know-best tone of
  explaining to a six-year-old.

## Why it bites technical writing hardest

Technical expertise *is* the cause. The better the engineer, the worse the blind
spot:

- **Expertise guarantees the curse.** If you know enough about a topic to have
  something to say, you have come to think about it in abstract chunks and
  functional labels that are now second nature to you but unfamiliar to your
  readers.
- **The amount of jargon you can use is a function of the audience**, never of your
  own comfort. Calibrate to the reader's chunks, not yours.
- **Jargon, abstractions, metaconcepts, and zombie nouns are the surface residue**
  of functional fixity + chunking — signals to be unpacked, not vices to be
  scolded.
- **The default action for unexplained jargon is mechanical:** spell it out on
  first use, give a concrete example, or replace it with the plain thing —
  regardless of how obvious it feels.

## Checklist

1. Distrust "this is obvious/standard" — it's the symptom, not a defense.
2. Spell out every acronym on first use.
3. Replace functional labels with the concrete thing ("assessment word" → "the
   word TRUE or FALSE").
4. Unpack one level of chunking when a noun phrase stacks 2+ expert abstractions.
5. Convert zombie nouns into verbs + named actors.
6. Show, don't name: add a concrete example or scenario for any abstract claim.
7. Don't skip "obvious" intermediate steps — the reader can't divine them.
8. Do NOT rely on "imagine your reader" — it fails.
9. Prefer external checks: real reader > drawer-then-reread > read aloud.
10. Calibrate jargon to the audience's expertise, never your own.
11. Be clear without being condescending; avoid "motherese."
