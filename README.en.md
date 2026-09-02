<div align="center">

<!-- GitHub status badges -->
![GitHub Stars](https://www.shieldcn.dev/github/stars/bgluis/bitbrush.svg?variant=secondary&size=sm)
![GitHub Forks](https://www.shieldcn.dev/github/forks/bgluis/bitbrush.svg?variant=secondary&size=sm)
![Watchers](https://www.shieldcn.dev/github/watchers/bgluis/bitbrush.svg?variant=secondary&size=sm)
![Contributors](https://www.shieldcn.dev/github/contributors/bgluis/bitbrush.svg?theme=emerald&size=sm)
![License](https://www.shieldcn.dev/github/license/bgluis/bitbrush.svg?variant=ghost&size=sm)

<br/>

<!-- Technology badges -->
![Go](https://www.shieldcn.dev/badge/Go-1.27-00ADD8.svg?logo=go&variant=branded&size=sm)
![TypeScript](https://www.shieldcn.dev/badge/TypeScript-5.7-3178C6.svg?logo=typescript&variant=branded&size=sm)
![Svelte](https://www.shieldcn.dev/badge/Svelte-5-FF3E00.svg?logo=svelte&variant=branded&size=sm)
![WebAssembly](https://www.shieldcn.dev/badge/WebAssembly-wasm-654FF0.svg?logo=webassembly&variant=branded&size=sm)
![Vite](https://www.shieldcn.dev/badge/Vite-6-646CFF.svg?logo=vite&variant=branded&size=sm)

  <h3>BitBrush</h3>
  Fully static web studio for algorithmic image filters, gradients and colour palettes — Go core compiled to WebAssembly, Svelte 5 shell.

  <br/>

  [🇧🇷 Português](./README.md) · 🇺🇸 **English**

</div>

# 📖 About

**BitBrush** is a fully static web app for generating **algorithmic image filters, gradients and
colour palettes**. Every effect is deterministic code — no ML models, no pretrained data, no
external image services. That constraint is a product principle: it caps dependencies and means
any output can be reproduced from its parameters alone.

The project is split into three strictly separated layers:

1. **Algorithm core — `internal/`** (pure Go): all filter, gradient and palette math. Runs and
   is tested with `go test` on any OS.
2. **WASM adapter — `cmd/wasm/`**: the only place that imports `syscall/js`. Registers named
   functions on `globalThis` and marshals data across the JS ↔ WASM boundary.
3. **Browser shell — `web/src/`** (Svelte 5 + TypeScript + Vite): controls, canvas I/O and URL
   state. No image math — everything is delegated to the Go core.

**What you can do:**

- **Image filters** — coloured ASCII art, pixelate (8-bit), error-diffusion dithering
  (Floyd–Steinberg / Atkinson / Stucki / Jarvis / Sierra / Burkes) plus ordered Bayer, Sobel
  edge detection, glitch / RGB shift, colour quantization (poster), halftone (mono / CMYK /
  RGB) and Voronoi stippling.
- **Algorithmic generators** — `truchet` (multi-scale Truchet tiles), `harmonograph`
  (damped-sinusoid figure), `attractor` (De Jong / Clifford / Svensson), `contours` (animated
  topographic map), `flowfield` (Sumi-ink flow-field strokes), `lsystem` (L-system turtle
  graphics) and `flame` (fractal flames / IFS).
- **Gradient generator + CSS tool** — multi-stop, angle, interpolation across
  sRGB / linear / HSL / Lab / LCh / OKLab / OKLCh with CSS Color 4 hue arcs, easing between
  stops, emitting baked hard stops or native `linear-gradient(… in oklch …)`.
- **Generative gradient** — a deterministic Go port of a fragment shader: 8 fields × 8 styles ×
  6 textures, with explicit `Seed` and `Time` (it animates and is reproducible).
- **Palettes** — extraction from an image (median-cut or k-means) and generation from a base
  colour by harmony rule (complementary / analogous / triadic / tetradic / split-complementary
  / monochromatic).
- **Animated GIF export** — keyframe a filter's params (start/end) and render N interpolated
  frames to a GIF, all on CPU.

**Determinism:** anything stochastic takes an explicit integer `seed` param (no `time.Now()`, no
unseeded `rand`) and all tool state is URL-encodable — a result is always shareable and
reproducible.

# 📋 Motivation

The project started from an idea I had while browsing X (Twitter), seeing a bunch of projects
that generated geometric and fractal patterns as well as an animated gradient generator. It was
also built as an experiment with **WebAssembly**, since I had never touched that technology
before.

# 💻 Getting started

### Requirements

- [Go](https://go.dev/dl/) **1.27+** (required by `go.mod`; used for the `GOOS=js GOARCH=wasm` target)
- [Node.js](https://nodejs.org/) **20 LTS or newer** (Vite 6)
- [Make](https://www.gnu.org/software/make/) — optional but recommended: the canonical entry
  points live in the `Makefile`
- A modern browser with **WebAssembly** and **WebGL2** support

### Installation

1. Clone the repository:
  ```sh
  git clone https://github.com/BGLuis/bitbrush.git
  ```

2. Enter the project directory:
  ```sh
  cd bitbrush
  ```

3. Install the front-end dependencies:
  ```sh
  cd web && npm install && cd ..
  ```

4. Start the development environment (builds the WASM and starts Vite):
  ```sh
  make dev
  ```
  Without `make`, the equivalent steps are:
  ```sh
  # 1. build the Go core to WebAssembly and copy wasm_exec.js
  GOOS=js GOARCH=wasm go build -o web/public/main.wasm ./cmd/wasm
  cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/public/   # Go <= 1.23: misc/wasm/wasm_exec.js

  # 2. start the dev server
  cd web && npm run dev
  ```

5. To produce the static production build in `web/dist/`:
  ```sh
  make build
  ```

### Handy commands

| Command       | What it does                                                          |
| ------------- | ------------------------------------------------------------------- |
| `make wasm`   | Builds `cmd/wasm` to `web/public/main.wasm` and copies `wasm_exec.js` |
| `make dev`    | `make wasm` + `npm run dev` (dev server)                            |
| `make build`  | `make wasm` + `npm run build` → static output in `web/dist/`        |
| `make test`   | `go test ./internal/...`                                            |
| `make check`  | `go vet` + `gofmt -l .` + `svelte-check`                            |
| `make clean`  | Removes build artefacts and `node_modules`                          |

# 🤝 Contributors

 <a href="https://github.com/BGLuis/bitbrush/graphs/contributors">
   <img src="https://contrib.rocks/image?repo=BGLuis/bitbrush"/>
 </a>

# 📄 License

Released under the **MIT** License. See [`LICENSE`](./LICENSE) for details.
