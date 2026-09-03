// Serialises application state ({mode, effect, params, generator}) into the URL query string
// so a result — including noisefield seed, colors and spots — is reproducible from a shared link,
// with no server involved. Only the recipe travels in the URL, never image data.

import type { FilterParams } from "./wasm";

const EFFECT_KEY = "fx";
const PARAM_PREFIX = "p.";
const MODE_KEY = "mode";

export interface GeneratorURLState {
  tool: "noise" | "multistop" | "patterns";
  field?: string;
  style?: string;
  texture?: string;
  angle?: number;
  scale?: number;
  distortion?: number;
  seed?: number;
  ratio?: string;
  colors?: string[];
  spots?: Array<[number, number]>;
  space?: string;
  hue?: string;
  easing?: string;
  kind?: string;
  stops?: Array<{ color: string; pos: number }>;
  pattern?: string;
  patternParams?: Record<string, any>;
}

export interface URLState {
  mode: "filter" | "pattern" | "generator" | "palette";
  effect: string | null;
  params: FilterParams;
  generator?: GeneratorURLState;
}

export function readStateFromURL(): URLState {
  const q = new URLSearchParams(location.search);
  const rawMode = q.get(MODE_KEY);

  if (rawMode === "pattern" || rawMode === "generator" || q.has("gen") || q.has("field") || q.has("pattern")) {
    const rawTool = q.get("tool");
    const isPattern = rawMode === "pattern" || rawTool === "patterns" || q.has("pattern");
    const tool =
      rawTool === "multistop"
        ? "multistop"
        : isPattern
          ? "patterns"
          : "noise";
    const genState: GeneratorURLState = { tool };

    if (tool === "patterns") {
      genState.pattern = q.get("pattern") || q.get("name") || "contours";
      if (q.has("ar")) genState.ratio = q.get("ar")!;
      const pParams: Record<string, any> = {};
      for (const [key, raw] of q) {
        if (!key.startsWith(PARAM_PREFIX)) continue;
        pParams[key.slice(PARAM_PREFIX.length)] = decodeValue(raw);
      }
      genState.patternParams = pParams;
    } else if (tool === "noise") {
      if (q.has("field")) genState.field = q.get("field")!;
      if (q.has("style")) genState.style = q.get("style")!;
      if (q.has("tex")) genState.texture = q.get("tex")!;
      if (q.has("deg")) genState.angle = Number(q.get("deg")) || 0;
      if (q.has("scale")) genState.scale = Number(q.get("scale")) || 50;
      if (q.has("dist")) genState.distortion = Number(q.get("dist")) || 55;
      if (q.has("seed")) {
        const s = Number(q.get("seed"));
        genState.seed = Number.isFinite(s) ? Math.round(s) : 0;
      }
      if (q.has("ar")) genState.ratio = q.get("ar")!;
      if (q.has("cols")) {
        genState.colors = q
          .get("cols")!
          .split(",")
          .map((c) => (c.startsWith("#") ? c : `#${c}`));
      }
      if (q.has("spots")) {
        genState.spots = q
          .get("spots")!
          .split(",")
          .map((s) => {
            const [x, y] = s.split("@").map(Number);
            return [Number.isNaN(x) ? 0.5 : x, Number.isNaN(y) ? 0.5 : y];
          });
      }
    } else {
      if (q.has("space")) genState.space = q.get("space")!;
      if (q.has("hue")) genState.hue = q.get("hue")!;
      if (q.has("ease")) genState.easing = q.get("ease")!;
      if (q.has("kind")) genState.kind = q.get("kind")!;
      if (q.has("deg")) genState.angle = Number(q.get("deg")) || 90;
      if (q.has("stops")) {
        genState.stops = q
          .get("stops")!
          .split(",")
          .map((item) => {
            const [col, pos] = item.split("@");
            const hex = col.startsWith("#") ? col : `#${col}`;
            return { color: hex, pos: Number(pos) || 0 };
          });
      }
    }

    return {
      mode: genState.tool === "patterns" ? "pattern" : "generator",
      effect: null,
      params: {},
      generator: genState,
    };
  }

  if (rawMode === "palette") {
    return {
      mode: "palette",
      effect: null,
      params: {},
    };
  }

  const params: FilterParams = {};
  for (const [key, raw] of q) {
    if (!key.startsWith(PARAM_PREFIX)) continue;
    params[key.slice(PARAM_PREFIX.length)] = decodeValue(raw);
  }
  return {
    mode: "filter",
    effect: q.get(EFFECT_KEY),
    params,
  };
}

let replaceStateTimer: number | undefined;
let pendingURL: string | null = null;

export function flushURLState(): void {
  if (replaceStateTimer !== undefined) {
    clearTimeout(replaceStateTimer);
    replaceStateTimer = undefined;
  }
  if (pendingURL !== null && typeof history !== "undefined") {
    try {
      history.replaceState(null, "", pendingURL);
    } catch {}
    pendingURL = null;
  }
}

function debouncedReplaceState(url: string, immediate = false): void {
  pendingURL = url;
  if (immediate) {
    flushURLState();
    return;
  }
  if (replaceStateTimer !== undefined) {
    clearTimeout(replaceStateTimer);
  }
  replaceStateTimer = window.setTimeout(() => {
    flushURLState();
  }, 300);
}

// Writes active filter and params to URL query string
export function writeStateToURL(effect: string, params: FilterParams, immediate = false): void {
  const q = new URLSearchParams();
  q.set(EFFECT_KEY, effect);
  for (const [key, value] of Object.entries(params)) {
    q.set(PARAM_PREFIX + key, String(value));
  }
  const url = `${location.pathname}?${q.toString()}${location.hash}`;
  debouncedReplaceState(url, immediate);
}

// Writes active generator state to URL query string
export function writeGeneratorStateToURL(state: GeneratorURLState, immediate = false): void {
  const q = new URLSearchParams();

  if (state.tool === "patterns") {
    q.set(MODE_KEY, "pattern");
    if (state.pattern) q.set("pattern", state.pattern);
    if (state.ratio) q.set("ar", state.ratio);
    if (state.patternParams) {
      for (const [key, value] of Object.entries(state.patternParams)) {
        if (value !== undefined && value !== null && value !== "") {
          q.set(PARAM_PREFIX + key, String(value));
        }
      }
    }
  } else {
    q.set(MODE_KEY, "generator");
    q.set("tool", state.tool);
    if (state.tool === "noise") {
    if (state.field) q.set("field", state.field);
    if (state.style) q.set("style", state.style);
    if (state.texture) q.set("tex", state.texture);
    if (state.angle !== undefined) q.set("deg", String(Math.round(state.angle)));
    if (state.scale !== undefined) q.set("scale", String(Math.round(state.scale)));
    if (state.distortion !== undefined) q.set("dist", String(Math.round(state.distortion)));
    if (state.seed !== undefined) q.set("seed", String(state.seed));
    if (state.ratio) q.set("ar", state.ratio);
    if (state.colors && state.colors.length) {
      q.set("cols", state.colors.map((c) => c.replace("#", "")).join(","));
    }
    if (state.spots && state.spots.length) {
      q.set(
        "spots",
        state.spots
          .map(([x, y]) => `${Number(x.toFixed(3))}@${Number(y.toFixed(3))}`)
          .join(","),
      );
    }
  } else {
    if (state.space) q.set("space", state.space);
    if (state.hue) q.set("hue", state.hue);
    if (state.easing) q.set("ease", state.easing);
    if (state.kind) q.set("kind", state.kind);
    if (state.angle !== undefined) q.set("deg", String(Math.round(state.angle)));
    if (state.stops && state.stops.length) {
      q.set(
        "stops",
        state.stops
          .map((s) => `${s.color.replace("#", "")}@${s.pos}`)
          .join(","),
      );
    }
  }
}

  const url = `${location.pathname}?${q.toString()}${location.hash}`;
  debouncedReplaceState(url, immediate);
}

function decodeValue(raw: string): number | string | boolean {
  if (raw === "true") return true;
  if (raw === "false") return false;
  if (raw.trim() === "") return raw;
  const n = Number(raw);
  return Number.isNaN(n) ? raw : n;
}
