# Classic Style — Prose as a Window onto the World

The central thesis of *The Sense of Style*, borrowed from Francis-Noël Thomas &
Mark Turner's *Clear and Simple as the Truth*. This is the frame every other rule
serves.

## The metaphor and its stance

Prose is a **clean window** onto the world. The writer has seen something real
that the reader hasn't yet noticed, and directs the reader's gaze so the reader
sees it too. Writing is **presentation**, and the motive is **disinterested
truth**.

Four presuppositions:

- **The truth can be known** and exists independently of the prose describing it.
  The prose is a transparent medium, not the thing itself.
- **The window must be clean.** Any feature of the prose that calls attention to
  itself — jargon, hedging, signposting, throat-clearing — is a smudge that blocks
  the view.
- **Clarity and simplicity are the proof of success**, not decoration. They are
  evidence the language has aligned with the truth (hence the source title,
  *Clear and Simple as the Truth*).
- **The writer presents rather than argues.** You show the reader the thing; you
  don't lecture them into believing it exists.

> **Encodable rule.** Treat every sentence as directing the reader's eye to a
> thing in the world. If a phrase points at your own writing process instead of at
> that thing, cut it.

## The writer–reader relationship

The model scene is **two competent equals in conversation** — not
lecturer/student, not expert/supplicant.

- **The reader is competent.** They can recognize the truth once given an
  unobstructed view, so you needn't over-explain, over-qualify, or argue
  defensively.
- **The writer has nothing to prove.** No need to display erudition, signal guild
  membership, or defend against accusations of naïveté. The writer is at ease.
- **Symmetry of intelligence.** Write as if to a smart peer who simply hasn't
  happened to notice this particular thing. This single assumption kills most
  academese: you stop hedging, name-dropping, and signposting, because an equal
  needs none of it.

> Imagine pointing something out to an intelligent friend who trusts you. Would
> you say "It should be noted that, to some extent…"? No. So don't write it.

**Before:** It is widely acknowledged in the literature that one might plausibly
argue that caching can, under certain conditions, improve performance.
**After:** Caching makes this faster.

## Classic style vs. the other styles

Every style is a different imagined scene — who the writer and reader are, and
what the writer is doing.

| Style | Imagined scene | Goal | Distinguishing feature |
|---|---|---|---|
| **Classic** | Two equals; writer shows reader a truth in the world | Present truth, disinterestedly | Transparent window; writer confident and unhurried; takes whatever length the truth needs |
| **Practical** | Defined roles (boss/employee, teacher/student) | Satisfy the reader's *need* for specific info | Fixed templates (memo, lab report, runbook); brief because the reader is in a hurry. The Strunk & White style. |
| **Plain** | Equals, everything stated outright | Inform with maximal simplicity | Closest to classic, but lays all in view; classic lets the reader do a little work to arrive at the truth |
| **Contemplative / Romantic** | Writer sharing inner experience | Express idiosyncratic, emotional reactions | Subjective; truth-of-feeling. Fits diaries, poetry, fiction, personal essays |
| **Prophetic / Oracular** | Writer as seer | Reveal what no one else *can* see | Asymmetric: special visionary access. Classic writer sees only what the reader could see too |
| **Self-conscious / Postmodern** | Writer anxious about being caught | Avoid conviction, dodge "naïveté about one's enterprise" | Constant meta-commentary, scare quotes, hedging about the impossibility of truth. **The chief antithesis of classic style.** |
| **Official / Bureaucratic** | Faceless institution to subject | Avoid liability, diffuse agency | Passives, zombie nouns, agentless constructions ("Mistakes were made") |
| **Academic** | Practical template + postmodern self-consciousness | Satisfy a journal format while signaling sophistication | Pinker's diagnosis of why academics stink at writing: template rigidity + defensive hedging + curse of knowledge |

**Key nuance — classic style is itself a pretense.** Even scientists know truth is
hard to reach and theory-laden. The difference is that a good writer *artfully
brackets* that anxiety for clarity's sake rather than flaunting it in every
sentence. When you open a cookbook you set aside — and expect the author to set
aside — deep philosophical questions about the nature of cooking. Classic style
brackets such questions as inappropriate to the task.

> **Default to classic** for expository/technical prose. Drop to practical style
> only when the reader needs a fixed template fast (release notes, runbooks, API
> refs). Never drift into postmodern self-consciousness or bureaucratic
> agentlessness.

## Prose habits classic style encourages

**a) Concrete nouns + active verbs; name the agent.**
Before: *An assessment of the impact of the configuration change on system
stability was performed.*
After: *We tested how the config change affected stability.*

**b) Verbs for actions, not nominalized nouns.**
Before: *The cancellation of the request results in the deallocation of the
buffer.*
After: *Cancelling the request frees the buffer.*

**c) Show, don't hedge.**
Before: *The data seem to suggest that there may possibly be a relationship.*
After: *The data show X causes Y.* (Then state the one real caveat, if any.)

**d) Present decisions as if obvious to a peer.**
Before: *One could perhaps make the argument that, in some sense, using a mutex
here might be preferable.*
After: *A mutex is simpler here: there are no concurrent readers.*

**e) Minimize metadiscourse and signposting; use conversational asides.**
Before: *In this section, we will first define the problem, then present our
approach, and finally discuss the results.*
After: *(Just write it.)* If a transition is genuinely needed, make it
conversational: *"As we just saw…" / "Now for the hard part…"*

**f) Let the prose take the length the truth needs** — but no padding.

## What classic style avoids

Pinker's roster of "soggy prose" habits — keep the window metaphor in mind rather
than memorizing this as a checklist:

- **Metadiscourse** — verbiage about verbiage (*this essay, the present review,
  the following section discusses…*). The reader spends more effort parsing the
  signpost than it saves.
- **Signposting** — "say what you'll say, say it, say what you said." Cut advance
  roadmaps; just make the argument.
- **Hedging / fluff** — *almost, apparently, fairly, in part, somewhat, to a
  certain degree, I would argue*. Implies you won't stand behind your claim.
- **Apologizing** — pre-emptive throat-clearing about how hard or contested the
  topic is.
- **Professional narcissism** — writing about your field's internal debates
  instead of the subject.
- **Clichés & mixed metaphors** — *"the company's bread and butter had risen…
  coasting on its laurels… a low-octane swan song."*
- **Metaconcepts** — *approach, assumption, concept, context, framework, issue,
  level, model, process, role, strategy, variable.* Often deletable.
  Before: *We adopted a caching strategy in order to address performance issues.*
  After: *We cached the results to make it faster.*
- **Zombie nouns** — spry verbs embalmed into lifeless nouns (*–ance, –ment,
  –ation*). See `mechanics.md`.
- **Unnecessary passives** — only the *gratuitous* agent-hiding ones; the passive
  itself is a legitimate tool.

Root cause of all of these: the **curse of knowledge** — experts default to
abstractions and zombie nouns because that's how they now think. See
`curse-of-knowledge.md`.

## The legitimate limits of classic style

Classic style is an **ideal to strive toward, not a universal law.** It explicitly
does *not* fit every situation:

- **When you genuinely must hedge.** The rule is selectivity, not abstinence:
  *save the qualifications for the claims that really need them.* And **qualify
  (spell out the actual failure condition) rather than hedge (sprinkle vague
  escape-words).**
  - Hedge (bad): *This is somewhat safe under certain conditions.*
  - Qualify (good): *This is safe unless two goroutines hold the lock
    simultaneously, which the scheduler prevents.*
- **Legal / scientific / safety-critical caveats.** Contracts, specs, limitations
  sections, and liability-sensitive text legitimately need exhaustive, defensive
  precision. Completeness beats elegance.
- **Practical-style genres** — templates, forms, reports — where brevity-to-format
  is correct.
- **Contemplative / romantic genres** — diaries, poetry, fiction, personal essays.
- **When the subject genuinely is your own uncertainty.** If the truth is "we
  don't yet know," saying so plainly *is* classic style. The failure is
  performative anxiety, not honest doubt.

> **Meta-rule.** Write in classic style by default. Break it deliberately and
> locally — for a real legal caveat, a genuine scientific limitation, or an honest
> expression of uncertainty — not habitually out of insecurity.

## One-line encoding

> Treat prose as a clean window onto a truth in the world; address a competent
> equal; use concrete nouns and active verbs with named agents; present
> confidently; delete metadiscourse, signposting, compulsive hedges, zombie nouns,
> and professional narcissism — and break these rules only when a genuine legal,
> scientific, or uncertainty caveat demands it.
