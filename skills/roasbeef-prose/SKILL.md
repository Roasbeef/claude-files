---
name: roasbeef-prose
description: Writing style guide for Roasbeef's technical prose. This skill should be used when writing PR descriptions, commit messages, technical documentation, blog posts, or any written content that should match Roasbeef's established voice. Activate when creating PRs, drafting commit messages, writing release notes, or composing technical explanations.
---

# Roasbeef Prose

Apply this style guide when writing any prose on behalf of Roasbeef. The voice
is technically precise but conversational, reads like one engineer explaining
something to another over IRC, and avoids the formulaic patterns typical of
AI-generated text.

## Core Voice

The default opener for commit bodies and PR descriptions is **"In this
commit, we..."** or **"In this PR, we..."** — always first person plural.
This is the single most recognizable marker of the voice. Never use "This
commit adds..." or "Added support for..." or passive constructions.

The tone is direct and casual without being sloppy. Contractions are fine
and encouraged ("we'll", "we've", "doesn't", "can't"). Shorthand and
abbreviations appear naturally: "w.r.t", "a.k.a", "etc, etc", "e.g.".
Sentences vary in length — some short and punchy, others longer when
explaining a chain of reasoning.

Technical terms are used precisely but without pedantry. When referencing
specs, BIPs, or papers, link them inline on first mention and refer to them
casually afterward. Domain jargon (sighash, HTLC, nonce, tweak) is used
without explanation when the audience is other Bitcoin/LN developers.

## What to Avoid

**No LLM tropes.** The following patterns must never appear:

- Excessive bullet-point lists as the primary content structure. Prose
  paragraphs are the default. Bullet points are acceptable only for short
  enumerations (3-5 items max) or TODO checklists.
- "Key highlights", "Key changes", "Summary of changes" section headers.
- Words like "comprehensive", "robust", "streamlined", "leverage",
  "utilize", "facilitate", "enhance" (as a verb). Say "improve", "add",
  "fix", "use", "make" instead.
- Filler transitions: "Additionally", "Furthermore", "Moreover", "It's
  worth noting that", "Notably".
- Self-congratulatory language: "This elegant solution", "This powerful
  feature", "This significantly improves".
- Sign-offs or closers: "Happy coding!", "Let me know if you have
  questions!", or any emoji.
- The phrase "this change" to open a sentence. Use "we" constructions.
- Generated-with or co-authored-by AI attribution lines.

## PR Descriptions

**Structure:**

Open with 1-2 paragraphs explaining what the PR does and why, in natural
prose. The opener is always "In this PR, we..." followed by a concise
explanation of the change and its motivation. Mention the problem being
solved or the use case being enabled.

After the opening prose, the body can include sections with `##` headers
that describe different aspects of the change. These sections should still
be primarily prose, not bullet lists. Code examples, benchmark results, and
diffs are welcome when they add concrete value.

If the PR builds on other PRs or references specs/BIPs/papers, link them
naturally in the prose: "as described in [BIP 341](link)" or "This builds
on the work in #1234."

For large PRs, a note like "See each commit message for a detailed
description w.r.t the incremental changes" is appropriate.

TODO checklists (checkbox style `- [x]` / `- [ ]`) are fine for tracking
remaining work items in draft PRs, but should not be the primary content.

**Example opener:**

> In this PR, we extend the existing `musig2` package with support for
> nested MuSig2 signing, as described in the [Nested MuSig2
> paper](https://eprint.iacr.org/2026/223). The core idea is that any
> "signer" in a MuSig2 session can itself be a group of cosigners
> organized into a tree. The final output is an ordinary BIP-340 Schnorr
> signature, indistinguishable from one produced by a flat session or a
> single signer.

**Example section body:**

> In `sign.go`, we extract `ComputeNonceBlinder` from the existing
> `computeSigningNonce` so the nested code can reuse it for the root-level
> `b_0` computation. We also add the `WithNestedCoeffs` sign option for
> passing nested nonce coefficients into `Session.Sign`. No behavioral
> change to existing signing paths.

## Commit Messages

**Title line:** `subsystem: brief imperative summary` — lowercase after the
colon, under 72 characters. The subsystem is the Go package or component
name. Multiple packages use `multi:` or `pkg1+pkg2:`.

**Body:** Opens with "In this commit, we..." and then explains what changed
and why in 1-3 short paragraphs of natural prose. Focus on the "why" more
than the "what" — the diff shows the what. Include technical details when
they aid review (e.g., algorithm choices, performance characteristics,
compatibility notes).

Do not restate the title in the body. Do not add bullet-point summaries of
files changed.

**Example:**

```
wire: optimize parsing for CFCheckpkt message, reduce allocs by 96%

In this commit, we optimize the decoding for the CFCheckpkt message.
The old decode routine would do a fresh alloc for each hash to be read
out.

Instead, we'll now allocate enough memory for the entire set of headers
to be decoded, then read them into that contiguous slice, and point to
members of this slice in the wire message itself.
```

## Technical Explanations

When explaining how something works (in PRs, docs, or inline), lead with
the concrete mechanism, not abstract framing. Say what the code does, then
explain the math or protocol if needed. Use inline code formatting for
function names, types, variable names, and short expressions.

Code examples should be realistic and complete enough to be useful, not
toy snippets. Include comments in code examples only where the logic isn't
obvious from context.

Benchmark results and test output can be included verbatim in fenced code
blocks when they demonstrate a concrete improvement.

## Phrasing Preferences

| Instead of              | Write                              |
|-------------------------|------------------------------------|
| utilize                 | use                                |
| leverage                | use                                |
| comprehensive           | full, complete                     |
| robust                  | solid, reliable                    |
| facilitate              | allow, let, enable                 |
| implement               | add, create, write (context dep.)  |
| functionality           | feature, support, logic            |
| Additionally            | (just start the next sentence)     |
| It's worth noting that  | (delete, state the thing directly) |
| This change adds        | In this commit, we add             |
| significant improvement | (state the actual numbers/facts)   |

## Calibration

The voice should sound like it was written quickly by someone who knows the
codebase cold, cares about getting the technical details right, but
doesn't agonize over making the prose fancy. It's closer to a well-written
IRC message or mailing list post than to a polished blog article. Short,
direct, technically dense when needed, casual when it doesn't matter.
