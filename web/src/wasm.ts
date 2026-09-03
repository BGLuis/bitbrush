// Loads the Go WASM runtime and wraps the globals it registers in typed
// functions. All pixel work crosses the JS <-> WASM boundary here, once
// per call, whole buffers only.

declare global {
  // Defined by /wasm_exec.js (shipped with the Go toolchain).
  var Go: new () => {
    run(instance: WebAssembly.Instance): Promise<void>;
    importObject: WebAssembly.Imports;
  };
  function bitbrushApplyFilter(
    name: string,
    rgba: Uint8Array,
    w: number,
    h: number,
    paramsJSON: string,
  ): { ok: boolean; data: Uint8Array | null; error: string };
  function bitbrushAsciiText(
    rgba: Uint8Array,
    w: number,
    h: number,
    paramsJSON: string,
  ): { ok: boolean; text: string; error: string };
  function bitbrushRenderGIF(
    name: string,
    rgba: Uint8Array,
    w: number,
    h: number,
    keyframesJSON: string,
    optionsJSON: string,
  ): { ok: boolean; data: Uint8Array | null; error: string };
  function bitbrushRenderGradient(
    paramsJSON: string,
  ): { ok: boolean; data: Uint8Array | null; error: string };
  function bitbrushGradientCSS(
    paramsJSON: string,
    cssOptionsJSON: string,
  ): { ok: boolean; css: string; error: string };
  function bitbrushRenderNoiseField(
    paramsJSON: string,
    w: number,
    h: number,
  ): { ok: boolean; data: Uint8Array | null; error: string };
  function bitbrushRenderNoiseFieldGIF(
    startParamsJSON: string,
    endParamsJSON: string,
    w: number,
    h: number,
    optionsJSON: string,
  ): { ok: boolean; data: Uint8Array | null; error: string };
  function bitbrushRenderGenerator(
    name: string,
    paramsJSON: string,
    w: number,
    h: number,
  ): { ok: boolean; data: Uint8Array | null; error: string };
  function bitbrushExtractPalette(
    rgba: Uint8Array,
    w: number,
    h: number,
    optionsJSON: string,
  ): { ok: boolean; colors: string[]; error: string };
  function bitbrushGenPalette(
    optionsJSON: string,
  ): { ok: boolean; colors: string[]; error: string };
}

export interface GradientStop {
  color: string; // "#rrggbb"
  pos: number; // [0,1]
}

export interface GradientParams {
  stops: GradientStop[];
  space: string; // srgb | linear | hsl | lab | lch | oklab | oklch
  hue: string; // shorter | longer | increasing | decreasing
  easing: string; // linear | ease | ease-in-out | cubic-bezier(...) | steps(...) | linear(...)
  width: number;
  height: number;
  angle: number; // CSS degrees
}

export interface GradientCSSOptions {
  kind: string; // linear | radial | conic
  angle: number;
  native: boolean; // emit "in oklch" vs baked stops
  samples: number; // baked stop count when !native
  selector: string; // "" => bare value
  property: string; // default "background"
}

export interface NoiseFieldSpot {
  color: string;
  x: number; // [0,1]
  y: number; // [0,1]
}

export interface NoiseFieldParams {
  field: string;
  style: string;
  texture: string;
  stops: string[]; // hex, for geometric fields
  spots: NoiseFieldSpot[]; // for organic fields
  angle: number;
  scale: number; // 0..100
  distortion: number; // 0..100
  seed: number;
  time: number; // animation clock, seconds
}

export interface PaletteExtractOptions {
  count: number;
  method: string; // "median-cut" | "kmeans"
  space: string; // "srgb" | "oklab"
  sort: string; // "none" | "luma" | "hue" | "population"
  alphaThreshold: number;
  seed: number;
}

export interface PaletteHarmonyOptions {
  base: string; // "#rrggbb"
  rule: string; // complementary | analogous | triadic | tetradic | split | monochromatic
  n: number;
  wheel: string; // "hsl" | "oklch"
  spread: number; // degrees
}

/** One params snapshot pinned at normalized time `t` in [0,1]. */
export interface GifKeyframe {
  t: number;
  params: FilterParams;
}

export interface GifOptions {
  frames: number;
  fps: number;
  loop: boolean;
  pingPong: boolean;
  /** Longest side of the GIF in px before rendering; 0 = keep source size. */
  maxDimension: number;
}

let ready: Promise<void> | null = null;

export function initWasm(url = "/main.wasm"): Promise<void> {
  if (!ready) {
    ready = (async () => {
      if (typeof Go !== "function") {
        throw new Error("wasm_exec.js not loaded — run `make wasm` first");
      }
      const go = new Go();
      const { instance } = await WebAssembly.instantiateStreaming(
        fetch(url),
        go.importObject,
      );
      void go.run(instance); // resolves only when the module exits
    })();
  }
  return ready;
}

export type FilterParams = Record<string, number | string | boolean>;

export function applyFilter(
  name: string,
  img: ImageData,
  params: FilterParams = {},
): ImageData {
  const res = bitbrushApplyFilter(
    name,
    new Uint8Array(img.data.buffer.slice(0)),
    img.width,
    img.height,
    JSON.stringify(params),
  );
  if (!res.ok || !res.data) throw new Error(res.error || "filter failed");
  // Copy into a fresh Uint8ClampedArray (plain ArrayBuffer) — ImageData
  // rejects the ArrayBufferLike that Go's Uint8Array is typed as.
  return new ImageData(new Uint8ClampedArray(res.data), img.width, img.height);
}

// asciiText mirrors applyFilter's shape but returns plain text (the ASCII
// filter's "copy as text" export) instead of pixels.
export function asciiText(img: ImageData, params: FilterParams = {}): string {
  const res = bitbrushAsciiText(
    new Uint8Array(img.data.buffer.slice(0)),
    img.width,
    img.height,
    JSON.stringify(params),
  );
  if (!res.ok) throw new Error(res.error || "ascii text failed");
  return res.text;
}

// renderGIF drives the internal/anim pipeline: it interpolates `params`
// between keyframes over `options.frames` frames, applies `name` to each,
// and returns encoded GIF bytes. Heavier than a single filter call — the
// Worker backend keeps it off the UI thread.
export function renderGIF(
  name: string,
  img: ImageData,
  keyframes: GifKeyframe[],
  options: GifOptions,
): Uint8Array {
  const res = bitbrushRenderGIF(
    name,
    new Uint8Array(img.data.buffer.slice(0)),
    img.width,
    img.height,
    JSON.stringify(keyframes),
    JSON.stringify(options),
  );
  if (!res.ok || !res.data) throw new Error(res.error || "gif render failed");
  return new Uint8Array(res.data);
}

// --- generators (internal/gradient, internal/noisefield) ---

export function renderGradient(params: GradientParams): ImageData {
  const res = bitbrushRenderGradient(JSON.stringify(params));
  if (!res.ok || !res.data) throw new Error(res.error || "gradient render failed");
  return new ImageData(new Uint8ClampedArray(res.data), params.width, params.height);
}

export function gradientCSS(params: GradientParams, options: GradientCSSOptions): string {
  const res = bitbrushGradientCSS(JSON.stringify(params), JSON.stringify(options));
  if (!res.ok) throw new Error(res.error || "gradient css failed");
  return res.css;
}

export function renderNoiseField(params: NoiseFieldParams, w: number, h: number): ImageData {
  const safe = { ...params, seed: Math.round(params.seed || 0) };
  const res = bitbrushRenderNoiseField(JSON.stringify(safe), w, h);
  if (!res.ok || !res.data) throw new Error(res.error || "noise field render failed");
  return new ImageData(new Uint8ClampedArray(res.data), w, h);
}

export function renderNoiseFieldGIF(
  start: NoiseFieldParams,
  end: NoiseFieldParams,
  w: number,
  h: number,
  options: GifOptions,
): Uint8Array {
  const safeStart = { ...start, seed: Math.round(start.seed || 0) };
  const safeEnd = { ...end, seed: Math.round(end.seed || 0) };
  const res = bitbrushRenderNoiseFieldGIF(
    JSON.stringify(safeStart),
    JSON.stringify(safeEnd),
    w,
    h,
    JSON.stringify(options),
  );
  if (!res.ok || !res.data) throw new Error(res.error || "noise field gif failed");
  return res.data;
}

// A flat param bag for the algorithmic generators (internal/generators
// registry) — attractors, harmonographs, Truchet tilings, and so on. Each
// generator names its own keys; see web/src/ui/generators.ts.
export type GeneratorParams = Record<string, number | string | boolean>;

export function renderGenerator(
  name: string,
  params: GeneratorParams,
  w: number,
  h: number,
): ImageData {
  const res = bitbrushRenderGenerator(name, JSON.stringify(params), w, h);
  if (!res.ok || !res.data) throw new Error(res.error || "generator render failed");
  return new ImageData(new Uint8ClampedArray(res.data), w, h);
}

// --- palette (internal/palette) ---

export function extractPalette(img: ImageData, options: PaletteExtractOptions): string[] {
  const res = bitbrushExtractPalette(
    new Uint8Array(img.data.buffer.slice(0)),
    img.width,
    img.height,
    JSON.stringify(options),
  );
  if (!res.ok) throw new Error(res.error || "palette extract failed");
  return res.colors;
}

export function genPalette(options: PaletteHarmonyOptions): string[] {
  const res = bitbrushGenPalette(JSON.stringify(options));
  if (!res.ok) throw new Error(res.error || "palette generate failed");
  return res.colors;
}
