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
  function bitbrushRenderComposite(
    baseRGBA: Uint8Array | null,
    baseW: number,
    baseH: number,
    specJSON: string,
    extrasRGBA: Uint8Array,
    extrasDimsJSON: string,
  ): { ok: boolean; data: Uint8Array | null; width: number; height: number; error: string };
  function bitbrushApplyPipeline(
    rgba: Uint8Array,
    w: number,
    h: number,
    chainJSON: string,
  ): { ok: boolean; data: Uint8Array | null; error: string };
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
    new Uint8Array(img.data.buffer, img.data.byteOffset, img.data.byteLength),
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
    new Uint8Array(img.data.buffer, img.data.byteOffset, img.data.byteLength),
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
    new Uint8Array(img.data.buffer, img.data.byteOffset, img.data.byteLength),
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
    new Uint8Array(img.data.buffer, img.data.byteOffset, img.data.byteLength),
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

// --- compositor (internal/compositor) — the "Compor" layer stack ---

export type BlendMode =
  | "normal"
  | "multiply"
  | "screen"
  | "overlay"
  | "darken"
  | "lighten"
  | "color-dodge"
  | "color-burn"
  | "hard-light"
  | "soft-light"
  | "difference"
  | "exclusion"
  | "linear-dodge"
  | "subtract";

export type LayerSource = "base" | "image" | "generator" | "gradient" | "noisefield" | "text";

export type FitMode = "cover" | "contain" | "stretch" | "center" | "tile";

export interface MaskParams {
  kind: "rect" | "ellipse" | "luma";
  x?: number; // [0, 1]
  y?: number; // [0, 1]
  w?: number; // [0, 1]
  h?: number; // [0, 1]
  feather?: number; // [0, 1]
  threshold?: number; // [0, 1] (luma)
  invert?: boolean;
}

export interface TextParams {
  content: string;
  fontFamily?: "go" | "gomono" | string;
  size?: number;
  color?: string;
  x?: number;
  y?: number;
  align?: "left" | "center" | "right" | string;
  opacity?: number;
}

/** One entry in a layer's own ordered filter chain. */
export interface ComposeStage {
  filter: string;
  params: FilterParams;
  mask?: MaskParams;
}

export interface ComposeLayer {
  enabled: boolean;
  source: LayerSource;
  imageIndex?: number; // source === "image"
  generator?: string; // source === "generator"
  genParams?: Record<string, unknown>; // generator / gradient / noisefield / text params
  fit?: FitMode; // default "cover"
  chain: ComposeStage[];
  blend: BlendMode;
  opacity: number; // 0..1
}

export interface ComposeSpec {
  width: number;
  height: number;
  layers: ComposeLayer[];
}

/** Concatenate image-layer buffers into one Uint8Array plus a [w,h] list,
 *  the shape bitbrushRenderComposite expects for its extras argument. */
export function packImages(imgs: ImageData[]): { buf: Uint8Array; dims: [number, number][] } {
  const dims: [number, number][] = imgs.map((im) => [im.width, im.height]);
  const total = imgs.reduce((n, im) => n + im.data.byteLength, 0);
  const buf = new Uint8Array(total);
  let off = 0;
  for (const im of imgs) {
    buf.set(new Uint8Array(im.data.buffer, im.data.byteOffset, im.data.byteLength), off);
    off += im.data.byteLength;
  }
  return { buf, dims };
}

/** Composite a layer stack. base is the primary image (or null); extras[i]
 *  backs a layer whose source is "image" and imageIndex is i. One boundary
 *  crossing: Go renders every generator/gradient/noise layer itself. */
export function renderComposite(
  base: ImageData | null,
  spec: ComposeSpec,
  extras: ImageData[] = [],
): ImageData {
  const { buf, dims } = packImages(extras);
  const baseBytes = base
    ? new Uint8Array(base.data.buffer, base.data.byteOffset, base.data.byteLength)
    : null;
  const res = bitbrushRenderComposite(
    baseBytes,
    base?.width ?? 0,
    base?.height ?? 0,
    JSON.stringify(spec),
    buf,
    JSON.stringify(dims),
  );
  if (!res.ok || !res.data) throw new Error(res.error || "composite failed");
  const w = res.width || spec.width || base?.width || 0;
  const h = res.height || spec.height || base?.height || 0;
  return new ImageData(new Uint8ClampedArray(res.data), w, h);
}

// --- pipeline (internal/filters.Pipeline) ---

/** One stage in a filter chain passed to applyPipeline. */
export interface PipelineStage {
  filter: string;
  params: FilterParams;
  mask?: MaskParams;
}

/** Apply an ordered chain of filter stages to an image in a single WASM call.
 *  Each stage's output is the next stage's input. An empty chain clones src. */
export function applyPipeline(img: ImageData, chain: PipelineStage[]): ImageData {
  const res = bitbrushApplyPipeline(
    new Uint8Array(img.data.buffer, img.data.byteOffset, img.data.byteLength),
    img.width,
    img.height,
    JSON.stringify(chain),
  );
  if (!res.ok || !res.data) throw new Error(res.error || "pipeline failed");
  return new ImageData(new Uint8ClampedArray(res.data), img.width, img.height);
}
