// Runes store for the "Compor" mode: an ordered, reorderable stack of
// layers composited bottom -> top by internal/compositor. Each layer has a
// source (base image / a 2nd uploaded image / a generator / gradient /
// noise field), its own ordered filter chain, a blend mode and an opacity.
//
// Modelled on generator-store.svelte.ts: local state + an explicit render
// loop + URL sync, so App.svelte only has to say "compose changed".

import { ui, refs, defaultParams } from "../store.svelte";
import { getBackend, type BackendPref } from "../../backend";
import { loadImageFile, imageElementToImageData } from "../../canvas";
import { downscaleImageData } from "../../preview";
import { calculateOutputDimensions } from "../generator/noisefield-data";
import { writeComposeStateToURL, type ComposeURLState } from "../../state";
import { scheduleComposeRender } from "./lib/render";
import { downloadCanvas } from "../lib/download";
import { showToast } from "../lib/toast.svelte";
import { generators } from "../../ui/generators";
import type { Control } from "../../ui/controls";
import type {
  BlendMode,
  ComposeLayer,
  ComposeSpec,
  ComposeTransform,
  FitMode,
  LayerSource,
  MaskParams,
} from "../../wasm";

export const BLEND_MODES: BlendMode[] = [
  "normal",
  "multiply",
  "screen",
  "overlay",
  "darken",
  "lighten",
  "color-dodge",
  "color-burn",
  "hard-light",
  "soft-light",
  "difference",
  "exclusion",
  "linear-dodge",
  "subtract",
];

export const FIT_MODES: FitMode[] = ["cover", "contain", "stretch", "center", "tile"];

export const SOURCE_LABELS: Record<LayerSource, string> = {
  base: "Imagem base",
  image: "Imagem enviada",
  generator: "Gerador",
  gradient: "Gradiente",
  noisefield: "Campo de ruído",
  text: "Texto",
};

export interface ChainStage {
  id: string;
  filter: string;
  params: Record<string, number | string | boolean>;
  mask?: MaskParams;
}

export interface ImageSlot {
  id: string;
  name: string;
  original: ImageData;
  preview: ImageData;
}

export interface LayerUI {
  id: string;
  enabled: boolean;
  source: LayerSource;
  slotId?: string; // source === "image"; undefined => unresolved (shared link)
  generator: string; // source === "generator"
  genParams: Record<string, any>; // generator / gradient / noisefield params
  fit: FitMode;
  transform: ComposeTransform;
  /** Masks this whole layer against what's below it (distinct from a chain
   *  stage's own `mask`, which only masks that filter's before/after). */
  mask?: MaskParams;
  chain: ChainStage[];
  blend: BlendMode;
  opacity: number; // 0..1
}

let idc = 0;
const uid = (p: string) => `${p}${(idc++).toString(36)}${Math.random().toString(36).slice(2, 6)}`;
const clamp01 = (v: number) => (v < 0 ? 0 : v > 1 ? 1 : v);

function identityTransform(): ComposeTransform {
  return { offsetX: 0, offsetY: 0, scale: 1, rotation: 0 };
}

export function generatorControls(name: string): Control[] {
  return generators.find((g) => g.name === name)?.controls ?? [];
}

function defaultGeneratorParams(name: string): Record<string, any> {
  const out: Record<string, any> = {};
  for (const c of generatorControls(name)) out[c.key] = c.default;
  return out;
}

function defaultGradientParams(): Record<string, any> {
  return {
    stops: [
      { color: "#0ea5e9", pos: 0 },
      { color: "#8b5cf6", pos: 1 },
    ],
    space: "oklab",
    hue: "shorter",
    easing: "linear",
    angle: 90,
  };
}

function defaultNoiseParams(): Record<string, any> {
  return {
    field: "flow",
    style: "duotone",
    texture: "smooth",
    stops: ["#101828", "#dc78a0"],
    spots: [],
    angle: 0,
    scale: 50,
    distortion: 55,
    seed: 0,
    time: 0,
  };
}

function defaultTextParams(): Record<string, any> {
  return {
    content: "BitBrush",
    fontFamily: "go",
    size: 72,
    color: "#ffffff",
    x: 0.5,
    y: 0.5,
    align: "center",
    opacity: 1.0,
  };
}

function newLayer(source: LayerSource): LayerUI {
  const L: LayerUI = {
    id: uid("L"),
    enabled: true,
    source,
    generator: "truchet",
    genParams: {},
    fit: "cover",
    transform: identityTransform(),
    chain: [],
    blend: "normal",
    opacity: 1,
  };
  switch (source) {
    case "generator":
      L.genParams = defaultGeneratorParams("truchet");
      L.blend = "screen";
      break;
    case "gradient":
      L.genParams = defaultGradientParams();
      L.blend = "overlay";
      L.opacity = 0.65;
      break;
    case "noisefield":
      L.genParams = defaultNoiseParams();
      L.blend = "soft-light";
      L.opacity = 0.5;
      break;
    case "text":
      L.genParams = defaultTextParams();
      L.blend = "normal";
      break;
    case "image":
    case "base":
      break;
  }
  return L;
}

interface ComposeSnapshot {
  layers: LayerUI[];
  selected: number;
  ratio: string;
}

// How long a burst of continuous edits (dragging a slider, holding an
// arrow key) is allowed to run before it settles into its own undo step.
// Longer than render.ts's 50ms render debounce on purpose.
const HISTORY_BURST_MS = 800;
const HISTORY_LIMIT = 50;

class ComposeStore {
  layers = $state<LayerUI[]>([newLayer("base")]);
  selected = $state(0);
  slots = $state<ImageSlot[]>([]);
  ratio = $state("16/9");
  status = $state("");

  // --- undo / redo ---
  // Plain arrays, not $state: nothing renders their contents, only whether
  // they're empty (exposed via the tick counters below).
  #history: ComposeSnapshot[] = [];
  #future: ComposeSnapshot[] = [];
  #baseline: ComposeSnapshot | undefined;
  #baselineKey: string | undefined;
  #burstTimer: ReturnType<typeof setTimeout> | undefined;
  #restoring = false;
  #historyTick = $state(0);
  #futureTick = $state(0);

  get canUndo(): boolean {
    void this.#historyTick;
    return this.#history.length > 0;
  }
  get canRedo(): boolean {
    void this.#futureTick;
    return this.#future.length > 0;
  }

  #snapshot(): ComposeSnapshot {
    return {
      layers: $state.snapshot(this.layers) as LayerUI[],
      selected: this.selected,
      ratio: this.ratio,
    };
  }

  // Called as the first thing scheduleRender() does, so it sees every edit
  // path — named mutators AND the ad-hoc in-place mutations ParamForm /
  // ChainEditor / LayerEditor's gradient+noise fields make before calling
  // scheduleRender() themselves.
  #noteEdit(): void {
    if (this.#restoring) return;
    const snap = this.#snapshot();
    const key = JSON.stringify(snap);
    if (this.#baseline === undefined) {
      // First call ever (post-hydration): seed the baseline, no history yet.
      this.#baseline = snap;
      this.#baselineKey = key;
      return;
    }
    if (key === this.#baselineKey) return; // a repaint with no real change
    if (this.#burstTimer === undefined) {
      this.#history.push(this.#baseline);
      if (this.#history.length > HISTORY_LIMIT) this.#history.shift();
      this.#historyTick++;
      this.#future = [];
      this.#futureTick++;
    } else {
      clearTimeout(this.#burstTimer);
    }
    this.#burstTimer = setTimeout(() => {
      this.#burstTimer = undefined;
      this.#baseline = this.#snapshot();
      this.#baselineKey = JSON.stringify(this.#baseline);
    }, HISTORY_BURST_MS);
  }

  #applySnapshot(s: ComposeSnapshot): void {
    this.#restoring = true;
    this.layers = s.layers;
    this.selected = s.selected;
    this.ratio = s.ratio;
    this.#restoring = false;
    this.#baseline = this.#snapshot();
    this.#baselineKey = JSON.stringify(this.#baseline);
    this.scheduleRender();
  }

  undo(): void {
    if (!this.canUndo) return;
    this.#future.push(this.#snapshot());
    this.#futureTick++;
    const prev = this.#history.pop()!;
    this.#historyTick++;
    this.#applySnapshot(prev);
  }

  redo(): void {
    if (!this.canRedo) return;
    this.#history.push(this.#snapshot());
    this.#historyTick++;
    const next = this.#future.pop()!;
    this.#futureTick++;
    this.#applySnapshot(next);
  }

  get aspectRatio(): number {
    const [w, h] = this.ratio.split("/").map(Number);
    return w && h ? w / h : 16 / 9;
  }

  get selectedLayer(): LayerUI | undefined {
    return this.layers[this.selected];
  }

  slotById(id: string | undefined): ImageSlot | undefined {
    return id ? this.slots.find((s) => s.id === id) : undefined;
  }

  // The GPU backend runs the compositor on the main thread (no shader), so
  // for the live preview prefer the worker/CPU path.
  #pref(): BackendPref {
    return ui.settings.backend === "gpu" ? "cpu" : ui.settings.backend;
  }

  select(i: number): void {
    if (i >= 0 && i < this.layers.length) this.selected = i;
  }

  addLayer(source: LayerSource): void {
    this.layers.push(newLayer(source));
    this.selected = this.layers.length - 1;
    this.status = source === "image" ? "Solte uma imagem nesta camada." : "";
    this.scheduleRender();
  }

  removeLayer(i: number): void {
    if (i < 0 || i >= this.layers.length) return;
    this.layers.splice(i, 1);
    if (this.layers.length === 0) this.layers.push(newLayer("base"));
    this.selected = Math.min(this.selected, this.layers.length - 1);
    this.scheduleRender();
  }

  moveLayer(from: number, to: number): void {
    if (
      from === to ||
      from < 0 ||
      to < 0 ||
      from >= this.layers.length ||
      to >= this.layers.length
    ) {
      return;
    }
    const [l] = this.layers.splice(from, 1);
    this.layers.splice(to, 0, l);
    this.selected = to;
    this.scheduleRender();
  }

  toggleLayer(i: number): void {
    this.layers[i].enabled = !this.layers[i].enabled;
    this.scheduleRender();
  }
  setBlend(i: number, m: BlendMode): void {
    this.layers[i].blend = m;
    this.scheduleRender();
  }
  setOpacity(i: number, v: number): void {
    this.layers[i].opacity = clamp01(v);
    this.scheduleRender();
  }
  setFit(i: number, f: FitMode): void {
    this.layers[i].fit = f;
    this.scheduleRender();
  }
  setTransform(i: number, patch: Partial<ComposeTransform>): void {
    const L = this.layers[i];
    Object.assign(L.transform, patch);
    // A "cover" fit has already cropped away part of the source before any
    // transform runs — avoid surprising the user by switching to "contain"
    // the first time they nudge position/scale/rotation.
    if (L.fit === "cover") L.fit = "contain";
    this.scheduleRender();
  }
  resetTransform(i: number): void {
    this.layers[i].transform = identityTransform();
    this.scheduleRender();
  }
  setLayerMask(i: number, mask: MaskParams | undefined): void {
    this.layers[i].mask = mask;
    this.scheduleRender();
  }
  setRatio(r: string): void {
    this.ratio = r;
    this.scheduleRender();
  }

  setSource(i: number, s: LayerSource): void {
    const cur = this.layers[i];
    if (cur.source === s) return;
    const fresh = newLayer(s);
    fresh.id = cur.id;
    fresh.enabled = cur.enabled;
    fresh.blend = cur.blend;
    fresh.opacity = cur.opacity;
    fresh.fit = cur.fit;
    fresh.transform = cur.transform;
    fresh.chain = cur.chain;
    this.layers[i] = fresh;
    this.scheduleRender();
  }

  setGenerator(i: number, name: string): void {
    this.layers[i].generator = name;
    this.layers[i].genParams = defaultGeneratorParams(name);
    this.scheduleRender();
  }

  addStage(i: number, filter: string): void {
    this.layers[i].chain.push({ id: uid("S"), filter, params: defaultParams(filter) });
    this.scheduleRender();
  }
  removeStage(i: number, j: number): void {
    this.layers[i].chain.splice(j, 1);
    this.scheduleRender();
  }
  moveStage(i: number, from: number, to: number): void {
    const ch = this.layers[i].chain;
    if (from === to || to < 0 || to >= ch.length) return;
    const [s] = ch.splice(from, 1);
    ch.splice(to, 0, s);
    this.scheduleRender();
  }
  setStageFilter(i: number, j: number, filter: string): void {
    const st = this.layers[i].chain[j];
    this.layers[i].chain[j] = { id: st.id, filter, params: defaultParams(filter) };
    this.scheduleRender();
  }
  setStageParam(i: number, j: number, key: string, val: unknown): void {
    this.layers[i].chain[j].params[key] = val as number | string | boolean;
    this.scheduleRender();
  }
  setStageMask(i: number, j: number, mask: MaskParams | undefined): void {
    this.layers[i].chain[j].mask = mask;
    this.scheduleRender();
  }

  // --- image slots ---

  async addImageForSelectedLayer(file: File): Promise<void> {
    if (!file.type.startsWith("image/")) {
      showToast(`"${file.name}" não é uma imagem`, "error");
      return;
    }
    let idx = this.selected;
    if (this.layers[idx]?.source !== "image") {
      this.addLayer("image");
      idx = this.selected;
    }
    try {
      const el = await loadImageFile(file);
      const original = imageElementToImageData(el);
      const preview = downscaleImageData(original, ui.settings.previewMaxDim);
      const slot: ImageSlot = { id: uid("img"), name: file.name, original, preview };
      this.slots.push(slot);
      this.layers[idx].slotId = slot.id;
      this.status = "";
      showToast(`${original.width}×${original.height} · ${file.name}`);
      this.scheduleRender();
    } catch (err) {
      console.error(err);
      showToast("Não foi possível decodificar a imagem", "error");
    }
  }

  bindSlot(i: number, slotId: string): void {
    this.layers[i].slotId = slotId;
    this.scheduleRender();
  }

  wantsImageDrop(): boolean {
    return this.selectedLayer?.source === "image";
  }

  recomputePreviews(): void {
    for (const s of this.slots) s.preview = downscaleImageData(s.original, ui.settings.previewMaxDim);
    this.scheduleRender();
  }

  // --- spec assembly ---

  buildSpec(scale: "preview" | "full"): {
    spec: ComposeSpec;
    base: ImageData | null;
    extras: ImageData[];
  } {
    const full = scale === "full";
    const base = full ? ui.original : ui.preview;

    const slotIndex = new Map<string, number>();
    const extras: ImageData[] = [];

    const layers: ComposeLayer[] = this.layers.map((L) => {
      const cl: ComposeLayer = {
        enabled: L.enabled,
        source: L.source,
        fit: L.fit,
        transform: L.transform,
        mask: L.mask,
        blend: L.blend,
        opacity: L.opacity,
        chain: L.chain.map((s) => ({ filter: s.filter, params: s.params, mask: s.mask })),
      };
      if (L.source === "base" && !base) {
        cl.enabled = false; // no base image loaded yet — keep the recipe, draw nothing
      } else if (L.source === "image") {
        const slot = this.slotById(L.slotId);
        if (!slot) {
          cl.enabled = false; // recipe kept, nothing to draw yet
          cl.imageIndex = 0;
        } else {
          let idx = slotIndex.get(slot.id);
          if (idx === undefined) {
            idx = extras.length;
            slotIndex.set(slot.id, idx);
            extras.push(full ? slot.original : slot.preview);
          }
          cl.imageIndex = idx;
        }
      } else if (L.source === "generator") {
        cl.generator = L.generator;
        cl.genParams = L.genParams;
      } else if (L.source === "gradient" || L.source === "noisefield" || L.source === "text") {
        cl.genParams = L.genParams;
      }
      return cl;
    });

    let width = 0;
    let height = 0;
    if (!base) {
      const cap = full ? 1600 : ui.settings.previewMaxDim || 720;
      const d = calculateOutputDimensions(this.ratio, cap);
      width = d.w;
      height = d.h;
    }
    return { spec: { width, height, layers }, base, extras };
  }

  scheduleRender(): void {
    this.#noteEdit();
    this.updateURL();
    const canvas = refs.canvas;
    if (!canvas) return;
    const { spec, base, extras } = this.buildSpec("preview");
    scheduleComposeRender({ canvas, base, spec, extras, pref: this.#pref() });
  }

  async exportPNG(): Promise<void> {
    this.status = "Processando…";
    try {
      const { spec, base, extras } = this.buildSpec("full");
      const backend = await getBackend(this.#pref());
      const out = await backend.renderComposite(base, spec, extras);
      const c = document.createElement("canvas");
      c.width = out.width;
      c.height = out.height;
      c.getContext("2d")!.putImageData(out, 0, 0);
      downloadCanvas(c, `bitbrush-compose-${out.width}x${out.height}.png`);
      this.status = "✓ Salvo!";
      setTimeout(() => (this.status = ""), 1500);
    } catch (err) {
      console.error(err);
      this.status = "Erro ao exportar";
      setTimeout(() => (this.status = ""), 2000);
    }
  }

  // Repaint the shared canvas immediately (no debounce) — used when the
  // canvas element (re)mounts while already in compose mode.
  paintNow(): void {
    this.scheduleRender();
  }

  // --- URL sync ---

  toURLState(): ComposeURLState {
    return {
      ratio: this.ratio,
      layers: this.layers.map((L) => {
        const source =
          L.source === "image"
            ? "img"
            : L.source === "generator"
              ? `gen:${L.generator}`
              : L.source === "gradient"
                ? "grad"
                : L.source === "noisefield"
                  ? "noise"
                  : L.source === "text"
                    ? "text"
                    : "base";
        return {
          source,
          enabled: L.enabled,
          blend: L.blend,
          opacity: L.opacity,
          fit: L.fit,
          transform: [L.transform.offsetX, L.transform.offsetY, L.transform.scale, L.transform.rotation],
          mask: L.mask,
          genParams:
            L.source === "generator" || L.source === "gradient" || L.source === "noisefield" || L.source === "text"
              ? L.genParams
              : undefined,
          chain: L.chain.map((s) => ({ filter: s.filter, params: s.params, mask: s.mask })),
        };
      }),
    };
  }

  updateURL(): void {
    writeComposeStateToURL(this.toURLState());
    ui.query = location.search;
  }

  hydrateFromURL(state: ComposeURLState): void {
    if (state.ratio) this.ratio = state.ratio;
    if (!state.layers?.length) return;
    this.layers = state.layers.map((r) => {
      let source: LayerSource = "base";
      let generator = "truchet";
      if (r.source === "grad") source = "gradient";
      else if (r.source === "noise") source = "noisefield";
      else if (r.source === "text" || r.source === "txt") source = "text";
      else if (r.source.startsWith("gen:")) {
        source = "generator";
        generator = r.source.slice(4) || "truchet";
      } else if (r.source.startsWith("img")) source = "image";

      const L = newLayer(source);
      L.generator = generator;
      if (source === "generator") {
        L.genParams = { ...defaultGeneratorParams(generator), ...(r.genParams ?? {}) };
      } else if (source === "text") {
        L.genParams = { ...defaultTextParams(), ...(r.genParams ?? {}) };
      } else if (r.genParams) {
        L.genParams = r.genParams;
      }
      // Uploaded images never travel in a link — an "img" layer comes back
      // disabled, awaiting a re-drop, but keeps its blend / opacity / chain.
      L.enabled = source === "image" ? false : (r.enabled ?? true);
      if (r.blend) L.blend = r.blend as BlendMode;
      if (typeof r.opacity === "number") L.opacity = r.opacity;
      if (r.fit) L.fit = r.fit as FitMode;
      if (r.transform) {
        const [offsetX, offsetY, scale, rotation] = r.transform;
        L.transform = { offsetX, offsetY, scale, rotation };
      }
      if (r.mask) L.mask = r.mask;
      L.chain = (r.chain ?? []).map((s) => ({
        id: uid("S"),
        filter: s.filter,
        params: s.params as Record<string, number | string | boolean>,
        mask: s.mask,
      }));
      return L;
    });
    this.selected = 0;
  }
}

export const composeStore = new ComposeStore();
