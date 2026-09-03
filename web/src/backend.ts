// The seam between the browser shell and whatever actually runs the pixel
// work. Every effect call in the app goes through a FilterBackend, so the
// engine underneath can change (main-thread WASM, a Web Worker, a WebGL2
// path) without the UI knowing.
//
// The interface is async on purpose: the Worker backend (the default once
// available) can only answer asynchronously, and a synchronous seam would
// have to be torn out to add it. GPU support is expressed as a per-effect
// capability check — a GPU backend delegates anything it can't accelerate
// to the CPU one rather than pretending to cover every effect.

import type {
  FilterParams,
  GifKeyframe,
  GifOptions,
  GradientParams,
  GradientCSSOptions,
  NoiseFieldParams,
  GeneratorParams,
  PaletteExtractOptions,
  PaletteHarmonyOptions,
} from "./wasm";

export type BackendKind = "cpu" | "cpu-worker" | "gpu";
export type BackendPref = "auto" | "cpu" | "gpu";

export interface FilterBackend {
  readonly kind: BackendKind;
  /** Resolve once the engine is loaded and ready to take calls. */
  init(): Promise<void>;
  /** Whole-image filter. Returns fresh ImageData; never mutates the input. */
  applyFilter(name: string, img: ImageData, params: FilterParams): Promise<ImageData>;
  /** ASCII filter's text export. */
  asciiText(img: ImageData, params: FilterParams): Promise<string>;
  /**
   * Render an animated GIF by interpolating params between keyframes. Runs
   * on the CPU core regardless of backend (multi-frame, GPU-unfriendly), so
   * every backend implements it the same way.
   */
  renderGIF(
    name: string,
    img: ImageData,
    keyframes: GifKeyframe[],
    options: GifOptions,
  ): Promise<Uint8Array>;
  /** Render a multi-stop gradient (internal/gradient). */
  renderGradient(params: GradientParams): Promise<ImageData>;
  /** Emit the CSS string for a gradient. */
  gradientCSS(params: GradientParams, options: GradientCSSOptions): Promise<string>;
  /** Render a generative noise-field gradient (internal/noisefield). */
  renderNoiseField(params: NoiseFieldParams, w: number, h: number): Promise<ImageData>;
  /** Render an animated GIF from noise-field gradient params (internal/anim). */
  renderNoiseFieldGIF(
    start: NoiseFieldParams,
    end: NoiseFieldParams,
    w: number,
    h: number,
    options: GifOptions,
  ): Promise<Uint8Array>;
  /** Render a named algorithmic generator (internal/generators registry). */
  renderGenerator(name: string, params: GeneratorParams, w: number, h: number): Promise<ImageData>;
  /** Extract a palette from an image (internal/palette). */
  extractPalette(img: ImageData, options: PaletteExtractOptions): Promise<string[]>;
  /** Generate a harmony palette from a base colour. */
  genPalette(options: PaletteHarmonyOptions): Promise<string[]>;
  /**
   * True when this backend runs `name` on its own fast path. A GPU backend
   * returns false for effects it hasn't got a shader for (the caller then
   * uses the CPU backend for those); CPU backends accelerate everything, so
   * they always return true.
   */
  accelerates(name: string): boolean;
}

const instances = new Map<BackendPref, Promise<FilterBackend>>();

// Registered by whichever module owns each implementation, so backend.ts
// doesn't import the Worker or WebGL code (and their bundles) unless asked.
type Factory = () => FilterBackend;
const factories: Partial<Record<"cpu" | "cpu-worker" | "gpu", Factory>> = {};

export function registerBackend(kind: keyof typeof factories, make: Factory): void {
  factories[kind] = make;
}

/**
 * Resolve the backend for a preference, reusing the cached backend per preference.
 * "auto" prefers the Worker (no main-thread jank), then GPU, then main-thread CPU;
 * "cpu" forces a CPU path (Worker if present); "gpu" forces GPU and throws if it isn't available.
 * Candidates are tried in order and the first whose init() succeeds wins.
 */
export function getBackend(pref: BackendPref = "auto"): Promise<FilterBackend> {
  const cached = instances.get(pref);
  if (cached) return cached;

  const p = (async () => {
    const errors: unknown[] = [];
    for (const make of candidates(pref)) {
      try {
        const backend = make();
        await backend.init();
        return backend;
      } catch (err) {
        errors.push(err);
      }
    }
    throw new AggregateError(errors, `no usable backend for preference "${pref}"`);
  })();

  instances.set(pref, p);
  return p;
}

function candidates(pref: BackendPref): Factory[] {
  const cpuWorker = factories["cpu-worker"];
  const cpu = factories["cpu"];
  const gpu = factories["gpu"];
  if (!cpu) throw new Error("no CPU backend registered");

  if (pref === "gpu") {
    if (!gpu) throw new Error("GPU backend unavailable");
    return [gpu, cpuWorker ?? cpu];
  }
  if (pref === "cpu") return [cpuWorker ?? cpu, cpu];
  // auto: Worker first (no jank, covers everything), then GPU, then main thread
  return [cpuWorker, gpu, cpu].filter((f): f is Factory => Boolean(f));
}

/** Drop the cached backends so the next getBackend() rebuilds them. */
export function resetBackend(): void {
  instances.clear();
}
