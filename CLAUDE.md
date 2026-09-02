# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Status

Go core + WASM adapter + Svelte 5 browser shell are all in place. The shell was ported from
hand-built imperative DOM to Svelte 5 for the "Estúdio" redesign (dark 3-zone workspace); the
three bespoke panels (GIF / generators / palette) are still imperative, hosted inside Svelte via
an action, and migrate one at a time. Keep this file in sync as reality diverges.

## What this is

BitBrush: a fully static web app for generating **algorithmic** image filters, gradients, and
colour palettes. Every effect is deterministic code — no ML models, no pretrained data, no
external image services. That constraint is a product principle: it caps dependencies and means
any output can be reproduced from its parameters alone.

v1 scope:

- Image filters: coloured ASCII art, pixelate (8-bit), dithering (error-diffusion —
  Floyd–Steinberg / Atkinson / Stucki / Jarvis / Sierra / Burkes — plus an ordered Bayer mode),
  Sobel edge detection, glitch / RGB shift, colour quantization (poster), halftone (AM screen,
  mono / CMYK / RGB), Voronoi stippling (`stipple`, uses `internal/voronoi`).
- Algorithmic generators — `internal/generators` registry + `internal/genall` link aggregator,
  each generator a self-registering package: `truchet` (multi-scale Truchet tiles),
  `harmonograph` (damped-sinusoid figure), `attractor` (De Jong / Clifford / Svensson density
  plots), `contours` (fBm heightfield + marching-squares topo map), `flowfield` (Sumi-ink
  flow-field strokes), `lsystem` (L-system turtle graphics), `flame` (fractal flames / IFS).
  All are pure `func(json.RawMessage, w, h) (*image.RGBA, error)`, deterministic on seed, and
  cross the boundary through `bitbrushRenderGenerator(name, paramsJSON, w, h)`. UI descriptors
  live in `web/src/ui/generators.ts` (mirrors `ui/controls.ts`); no imperative panel surfaces
  them yet — see BACKLOG.md.
- Gradient generator + CSS tool — `internal/gradient`: multi-stop, angle, interpolation across
  sRGB / linear / HSL / Lab / LCh / OKLab / OKLCh with CSS Color 4 hue arcs, easing between
  stops (`linear` / named / `cubic-bezier()` / `steps()` / `linear()`), emits either baked hard
  stops or native `linear-gradient(… in oklch shorter hue …)`.
  NOTE on the "gurade" reference: `https://gurade.netlify.app` no longer hosts the perceptual/
  easing tool that inspired this — it's now an unrelated single-file WebGL gradient-art toy
  ("Gradient Studio"). The easing/CSS reference is larsenwork's `postcss-easing-gradients` plus
  CSS Color 4. The *generative* half of that toy was worth porting and became `internal/noisefield`.
- Generative gradient — `internal/noisefield`: a deterministic Go port of Gradient Studio's
  fragment shader. 8 fields (Linear…Flow, the last three noise-warped/organic) × 8 styles
  (Metallic…Rainbow colour grades) × 6 textures (Smooth…Paper). Explicit `Seed` + `Time` so it
  animates and is reproducible.
- Palette — `internal/palette`: extraction from an image (median-cut or k-means, `Extract`
  dispatch + `SortPalette`) and generation from a base colour by harmony rule (`Harmony`:
  complementary / analogous / triadic / tetradic / split-complementary / monochromatic, in an
  HSL or OKLCh hue wheel).
- Animated GIF export: keyframe a filter's params (start/end) and render N interpolated frames
  to a GIF — `internal/anim`, all CPU, its own resolution cap. Small clips are cheap on CPU;
  the bottleneck is the encoder, not the per-frame filter.

## Architecture

Three layers, strictly separated:

1. **Algorithm core — `internal/`** (pure Go). All filter / gradient / palette math. No
   `syscall/js`, no `js`-tagged build constraints — it must run and be tested with plain
   `go test` on any OS. Effectively all real logic lives here.
2. **WASM adapter — `cmd/wasm/main.go`** — the *only* file that imports `syscall/js`. Registers
   named functions on `globalThis`, marshals data across the boundary, recovers panics, blocks
   forever. No algorithm logic.
3. **Browser shell — `web/src/`** (Svelte 5 + TypeScript + Vite). Controls, canvas I/O, URL
   state. No image math — it calls a **backend** for all pixel work. The "Estúdio" shell lives
   in `web/src/app/` (component tree + `store.svelte.ts` runes state); `main.ts` just registers
   the backends and `mount()`s `App.svelte`. Framework-agnostic modules stay outside `app/`:
   `backend.ts`, `state.ts` (URL codec), `settings.ts`, `wasm.ts`, `canvas.ts`, `preview.ts`,
   `ui/controls.ts` (effect descriptors), `worker/filter-worker.ts`.

### Backend seam — `web/src/backend.ts`

The shell never calls `wasm.ts` directly; it goes through a `FilterBackend`
(`applyFilter` / `asciiText` / `renderGIF`, all async). Implementations register themselves by
kind and `getBackend(pref)` resolves one, trying candidates in order so a broken one falls back:

- `web/src/backends/cpu.ts` — Go/WASM on the main thread. Always works; blocks the UI. The
  reference path and last-resort fallback.
- `web/src/backends/worker.ts` + `web/src/worker/filter-worker.ts` — the **preferred** backend:
  a second WASM instance in a classic Web Worker (loads `/wasm_exec.js` via `importScripts`), so
  a slow filter never janks the page. `filter-worker.ts` is `@ts-nocheck` on purpose — a thin
  message router, same rationale as `cmd/wasm/main.go`; its contract lives in the typed
  `WorkerBackend`.
- `web/src/backends/gpu.ts` — WebGL2 **accelerator**, opt-in via settings. Has GLSL shaders for
  `pixelate` (`accelerates()` gates the filter path) and for the whole `renderNoiseField` (a
  near-line-for-line port of `internal/noisefield`, for a smooth animated preview). Matches the
  Go core exactly on the five geometric fields; the noise-warped fields (Mesh/Freeform/Flow)
  drift a little — highp `float` on the GPU vs `float64` in Go — so the CPU port stays
  authoritative for PNG/GIF export. Everything else (gradient sampling, palette, text/GIF
  exports) delegates to the Go core. Floyd–Steinberg dither and ASCII are CPU-only by design
  (serial / glyph rasterisation) and will never get a shader here.

`web/src/preview.ts` caps the live-preview image to `settings.previewMaxDim` (a per-viewer
localStorage pref, *not* URL state); exports always run on the full-resolution original.

### Svelte shell — `web/src/app/`

`App.svelte` is the three-zone Estúdio layout (`Topbar` / `ToolRail` · `Stage` · `Inspector` /
`Statusbar`). Shared state is `store.svelte.ts` (`ui` runes object + `refs.canvas`); a single
`$effect` in `App.svelte` is the live filter loop — it writes the URL recipe and hands a
coalesced job to `app/lib/render.ts`. Filter params render from the `ui/controls.ts` descriptors
via `app/ParamControls.svelte` (the Svelte port of the old `ui/panel.ts`).

The three hand-built imperative panels — `gif.ts`, `generators-panel.ts`, `palette-panel.ts`
(with `ui/widgets.ts`) — are **unchanged**: they still return `{ element }` and are hosted by
the `app/lib/panel.ts` Svelte action (factories in `app/lib/wrapped.ts`). Migrating one to a
native component = delete its factory + source, add the component. Type: system stack for now;
`style.css` carries the design tokens and restyles the legacy panel classes — drop self-hosted
woff2 + `@font-face` there for the Space Grotesk / Hanken Grotesk / IBM Plex Mono identity (no
runtime font CDN — deps stay capped).

### JS ↔ WASM boundary contract

Cross the boundary **once per operation, whole buffers only**. Never call into WASM per pixel or
per row — the call overhead dominates.

- TS reads `ImageData` from a `<canvas>`, calls a registered global (e.g. `bitbrushApplyFilter`)
  with: RGBA bytes as a `Uint8Array`, `width`, `height`, and a params object (plain JS object or
  JSON string).
- Go: `js.CopyBytesToGo` → wrap as `*image.RGBA` → run the core function → `js.CopyBytesToJS`
  the result back.
- TS writes the returned bytes with `putImageData`.
- Registered globals return an `{ok, data, error}`-shaped result; a panic in core code is
  recovered in the adapter and returned as `error`, never left to abort the instance.
- Globals: `bitbrushApplyFilter`, `bitbrushAsciiText`, `bitbrushRenderGIF`, and the generator /
  palette set in `cmd/wasm/generators.go`: `bitbrushRenderGradient(paramsJSON)`,
  `bitbrushGradientCSS(paramsJSON, cssOptionsJSON)`, `bitbrushRenderNoiseField(paramsJSON, w, h)`,
  `bitbrushExtractPalette(rgba, w, h, optionsJSON)` → `{ok, colors: string[], error}`,
  `bitbrushGenPalette(optionsJSON)` → `{ok, colors, error}`. All go through the FilterBackend
  seam (worker by default); enums cross as lower-case name strings.

### Registry pattern

Filters and generators are registered by string name with a uniform signature.

- **Filters** — `internal/filters/registry.go`. Adding one is a three-touch-point change: one new
  file in `internal/filters/` (calls `Register` from its `init()`) + one control descriptor in
  `web/src/ui/controls.ts` (rendered by `app/ParamControls.svelte`). No adapter change — the
  `bitbrushApplyFilter` global dispatches by name.
- **Generators** (image from parameters alone, no input image) — `internal/generators/registry.go`.
  Adding one: a new self-registering package under `internal/` + one blank-import line in
  `internal/genall/genall.go` + one descriptor in `web/src/ui/generators.ts`. Signature
  `func(json.RawMessage, w, h int) (*image.RGBA, error)`; dispatched by `bitbrushRenderGenerator`.
- `internal/gradient` and `internal/noisefield` predate the generator registry and keep their own
  dedicated globals.

Keep that shape — don't special-case individual effects in the adapter or the shell.

### Shared pieces (do not duplicate)

- `internal/colorspace` — sRGB↔linear, HSL, LAB, LCH, OKLab. Used by dithering, quantization,
  gradient interpolation, and palette extraction alike.
- Median-cut colour quantization is shared between the **Colour Quantization filter** and
  **palette extraction** — one implementation, called by both.

### Determinism

Anything stochastic (glitch / RGB shift) takes an explicit integer `seed` param — no
`time.Now()`, no unseeded `rand`. All tool state (filter params, gradient stops, seed) must be
URL-encodable so a result is shareable and reproducible.

## Layout

```
cmd/wasm/main.go        syscall/js adapter — filters, ascii, gif
cmd/wasm/generators.go  syscall/js adapter — gradient / noisefield / palette / generator globals
internal/filters/       the image filters + registry (incl. halftone, stipple, dither kernels)
internal/anim/          keyframe param interpolation + multi-frame render + animated GIF encode
internal/gradient/      multi-stop gradient sampling, easing curves, CSS emission
internal/noisefield/    generative noise-field gradient (Gradient Studio shader port)
internal/palette/       median-cut / k-means extraction, sorting, harmony generation
internal/colorspace/    sRGB/linear, OKLab(+inverse)/OKLCh, CIE Lab/LCh, HSL, gamut clamp
internal/voronoi/       weighted Lloyd relaxation (jump-flooding) — backs the stipple filter
internal/generators/    generator registry + shared paint helpers (fill / AA line / density acc)
internal/genall/        blank-imports every generator package so cmd/wasm links them
internal/{truchet,harmonograph,attractor,contours,flowfield,lsystem,flame}/  the generators
web/                    Vite + Svelte 5 project
  index.html            <div id="app">
  svelte.config.js
  src/
    main.ts             registers backends, warms one, mount(App)
    wasm.ts             WASM loader + typed boundary wrappers
    backend.ts          the FilterBackend seam + getBackend()
    backends/           cpu.ts, worker.ts, gpu.ts
    worker/             filter-worker.ts (classic worker, @ts-nocheck)
    state.ts            {effect, params} <-> URL query codec
    settings.ts         per-viewer engine + preview-res prefs (localStorage)
    preview.ts          live-preview downscaling
    canvas.ts           canvas <-> ImageData helpers
    style.css           design tokens + restyle of the legacy panel classes
    ui/controls.ts      per-effect control descriptors (data only)
    ui/widgets.ts       imperative form widgets for the wrapped panels
    gif.ts              animated-GIF export panel      (imperative, wrapped)
    generators-panel.ts gradient / noise-field panel   (imperative, wrapped)
    palette-panel.ts    palette extraction + harmony   (imperative, wrapped)
    app/                the Svelte "Estúdio" shell
      App.svelte        3-zone layout + the live filter $effect
      store.svelte.ts   `ui` runes state + `refs.canvas`
      Topbar/ToolRail/Stage/Inspector/Statusbar.svelte
      ParamControls.svelte   descriptor -> live form (was ui/panel.ts)
      lib/render.ts     rAF-coalesced, seq-guarded filter render
      lib/panel.ts      action hosting an imperative `{ element }` panel
      lib/wrapped.ts    factories for the three wrapped panels
      lib/image.ts      file load + preview downscale, wired to `ui`
  public/               main.wasm, wasm_exec.js  (build outputs — gitignored)
  dist/                 static deploy output
Makefile                canonical entrypoints
```

## Commands

Go through the Makefile for anything that touches both halves.

```
make wasm       # GOOS=js GOARCH=wasm go build -o web/public/main.wasm ./cmd/wasm
                # + copy $(go env GOROOT)/lib/wasm/wasm_exec.js -> web/public/
                #   (Go <= 1.23 keeps it at misc/wasm/wasm_exec.js)
make dev        # make wasm, then (cd web && npm run dev)
make build      # make wasm, then (cd web && npm run build)  -> web/dist/
make test       # go test ./...
make check      # go vet ./... && gofmt -l . && (cd web && npm run check)  # svelte-check
```

The core is plain Go — run its tests directly:

```
go test ./internal/...
go test ./internal/filters -run TestFloydSteinberg      # single test
go test -bench=. ./internal/filters                     # benchmarks — watch per-effect cost
go test -race ./internal/...
```

`wasm_exec.js` must match the Go toolchain that built `main.wasm`; re-copy it after any Go
upgrade. `main.wasm` must be served as `application/wasm` (Vite does this; a custom host might
not).

## Toolchain decisions

- Build with **standard Go**. TinyGo (much smaller binary) is the planned size optimisation but
  is *not* in use — don't code the core around TinyGo's stdlib limits. Revisit when binary size
  is a real problem.
- 100% static hosting (Netlify / GitHub Pages / Cloudflare Pages). No backend, no database.
  Don't add server-side code without revisiting this decision.
- **Svelte 5** for the shell (runes; no SvelteKit — plain Vite SPA that `mount()`s one tree).
  `@sveltejs/vite-plugin-svelte` is pinned to the v5 line to stay on Vite 6. Type-check with
  `svelte-check`, not bare `tsc`. Keep pixel/gradient/palette logic out of components — it
  belongs in `internal/` behind the backend seam.
- **CPU-first.** The Go/WASM core is the canonical engine: static images must run well on
  devices with no usable GPU. The Worker backend keeps it off the UI thread; the WebGL2 backend
  is an *optional accelerator* for the few embarrassingly-parallel effects, never a requirement.
  Every effect must always have a working CPU path. GIF/animation stays CPU too.
