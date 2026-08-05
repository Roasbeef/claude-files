---
name: explainer-video
description: >
  Produce a narrated explainer video end to end — interview-driven, like
  /ideate but the output is a rendered mp4. Pipeline: script → TTS or recorded
  VO → Whisper word timestamps → Remotion scenes cued to the spoken word →
  headless 4K render. Use when asked to "make a video", "produce an explainer",
  or "turn this script/post into a video".
---

# Explainer Video — interview-driven e2e production

Turn a rough idea, script, or blog post into a finished narrated motion-graphics
video. The edit is *text*: because every overlay is cued to a transcript word,
the VO is a swappable input and the whole video is re-renderable from source.

## Pipeline shape

```
script.md ──→ VO audio (TTS or recorded) ──→ Whisper word timestamps
                                                    ↓
        shared components + anim.tsx  ──→  Remotion scenes (React)
                                                    ↓
                              cue sheet times every overlay to the spoken word
                                                    ↓
                              headless render ──→ out/final.mp4
```

The expensive half is **graphics**, not editing. Budget accordingly.

## Phase 0 — Interview (mandatory, /ideate style)

Batch 3–4 questions per `AskUserQuestion` call. Cover these areas before
touching anything; target ~80% confidence:

1. **Audience & framing** — who is this for, and what one scene matters most
   to them? Every video has one scene the audience actually came for — find
   it, name it in the plan, and give it the most visual care and room to
   breathe.
2. **Arc & scenes** — how many scenes, what's the one-line idea of each?
   Target length? (~140 wpm spoken; TTS reads ~10% faster than estimate.)
3. **VO route** — **A:** user records (~20 min, authentic) or **B:** ElevenLabs
   TTS (instant, re-generatable while the script is still moving).
   Recommend B for the first render, swap A in later — the graphics don't care.
   If B: confirm `ELEVENLABS_API_KEY` and optional voice preference.
4. **Design system source** — a product screenshot to derive from beats an
   invented look every time. Ask for one. Extract: palette, type rules (mono
   for numerics is a strong default), signature components worth rebuilding
   as animatable React.
5. **Logistics** — repo location, resolution (default 3840×2160 @ 30fps),
   deadline pressure. If the day is short, agree on a **fast path** now: which
   scene subset is a coherent standalone cut?

Write the answers into `production.md` in the repo (scene briefs + design
system + fast path), and the narration into `script.md`. These two files are
the contract every downstream agent reads.

## Phase 1 — Setup (hard-won gotchas, do not rediscover)

- Scaffold: `npm init -y`, then `npm i remotion @remotion/cli react react-dom`
  and `npm i -D typescript@5 @types/react @types/react-dom`.
  - **Pin `typescript@5`.** TypeScript 7 (the Go compiler) drops `ts.sys`,
    which `@remotion/bundler` needs; the failure is an opaque
    `Cannot read properties of undefined (reading 'readFile')` in esbuild-loader.
  - Do **not** set `"type": "module"` in package.json.
- Whisper: `brew install whisper-cpp`, model via
  `curl -L -o work/models/ggml-base.en.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin`
  (~142MB; base.en is plenty for clean VO). The old pip `whisper` binary on
  this machine is dead (stale python3.9 shebang) — don't use it.
- Secrets: user exports live in `~/.zshrc`, which non-interactive shells skip.
  Snapshot with `zsh -lic 'printf "ELEVENLABS_API_KEY=%s\n" "$ELEVENLABS_API_KEY" > .env'`
  (note `-i`), `chmod 600`, gitignore it.
- Renders and installs may need the sandbox disabled (TLS/cert + Chrome).
  Remotion downloads its own headless Chrome on first render.
- Layout: `vo/` (audio + per-scene .txt), `work/transcripts/`, `work/models/`,
  `public/vo/` (copies for `staticFile`), `src/scenes/`, `src/components/`,
  `out/` (gitignored), `refs/` (design reference screenshots).

## Phase 2 — VO + word timestamps

- One `.txt` per scene from the script's narration (blockquotes only, not
  stage directions). **Spell out tickers/initialisms phonetically** for TTS
  ("P-S-B-T", "L-N-D") — the graphics show the real spelling.
- Generate via the ElevenLabs API in a re-runnable `vo/generate.sh`: for each
  scene, `jq -Rs '{text: ., model_id: "eleven_multilingual_v2", voice_settings:
  {stability: 0.5, similarity_boost: 0.75, style: 0.25}}' < vo/sceneN.txt`
  POSTed to `/v1/text-to-speech/<VOICE_ID>?output_format=mp3_44100_128` with
  the `xi-api-key` header. Voice "Brian" (`nPczCjzI2devNBz1zQrb`) is a good
  narration default.
- Report total runtime vs target immediately — this is the moment to trim.
- Transcribe:
  `ffmpeg -i vo/sceneN.mp3 -ar 16000 -ac 1 work/sceneN.wav` then
  `whisper-cli -m work/models/ggml-base.en.bin -f work/sceneN.wav -ml 1 -sow -oj -of work/transcripts/sceneN`.
  `-ml 1 -sow` gives one word per segment: `.transcription[].offsets.from/to`
  in **milliseconds**. Frame = `ms / 1000 * fps`.

## Phase 3 — Design system + shared components (single writer)

Build these yourself *before* fanning out scenes, so parallel agents never
fight over shared files:

- `src/anim.tsx` — all global timing knobs (reveal/stagger/overlayIn/
  overlayOut/easing), palette, fonts. "Make it snappier" must be a one-line
  change. Eased motion only, never bouncy.
- `src/components/<system>.tsx` — the signature components rebuilt from the
  reference screenshot as animatable React (props expose the animatable bits,
  e.g. a bar's `fill: 0..1`). Reuse components across scenes — one asset-bar
  component can serve both a wallet view and a set of supply counters — so
  the video reads as one product.
- Stubs for every scene + `Root.tsx` + `FinalEdit.tsx`, typecheck, and render
  one smoke still — prove the whole render path before the expensive phase.

## Phase 4 — Scenes in parallel (Workflow fan-out)

One agent per scene (`parallel`, one phase). Each agent:

- owns exactly one file, `src/scenes/SceneN.tsx`, keeps the propless
  `React.FC` export;
- reads `production.md`, `script.md`, its transcript JSON, `anim.tsx`,
  the shared components;
- gets its frame budget (VO duration + ~0.9s tail) and a concrete visual
  brief *in the prompt* — beats named against narration phrases;
- runs a mandatory verify loop, ≥2 iterations: `tsc --noEmit`, then
  `npx remotion still SceneN out/review/sceneN-f<F>.png --frame <F>` at 4–6
  beat frames, Read the PNGs, critique, fix;
- returns structured output: beats + frame numbers, decisions, still paths.

After the fan-out: full-project `tsc`, personally Read a few stills from each
scene (cross-scene consistency is the orchestrator's job, not the agents'),
then commit.

## Phase 5 — Cue sheet + render + QA

- `FinalEdit.tsx`: `<Series>` of scenes, each wrapped with
  `<Audio src={staticFile('vo/sceneN.mp3')} />`; boundaries from real
  (ffprobe) durations + the tail pad, mirrored in `Root.tsx`.
- Render in the background: `npx remotion render FinalEdit out/final.mp4
  --codec h264`. ~4.7k 4K frames ≈ 10–25 min on an M-series; progress doesn't
  stream through pipes, check the process not the log.
- QA **from the mp4 itself**, not the review stills:
  `ffprobe` (duration, both streams present);
  `ffmpeg -ss <t> -i out/final.mp4 -frames:v 1` at ~10 timestamps including
  scene boundaries, Read them; `-af volumedetect` on a slice (expect max
  around −6dB, no clipping).
- Deliver the path + a hero frame (Substrate attachments are image-only —
  don't try to attach the mp4).

## Iterating after v1

- New VO (e.g. the user records route A): drop files in `vo/`, re-run
  Phase 2, update durations in `FinalEdit.tsx`/`Root.tsx`, re-render. Scene
  internals only shift if wording changed enough to move the cue words.
- "Make it snappier / calmer": edit `anim.tsx` only.
- Script wording changes: regenerate that scene's VO + transcript, then ask
  that scene's agent (or edit directly) to re-cue the affected beats.
