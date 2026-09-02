// Runes-based reactive store for the Gradient & Noise generator,
// managing state, presets, animation, canvas rendering, spots and export.

import { ui, refs } from "../store.svelte";
import { getBackend } from "../../backend";
import { putImageData } from "../../canvas";
import { writeGeneratorStateToURL, type GeneratorURLState } from "../../state";
import type { GradientParams, NoiseFieldParams } from "../../wasm";
import {
  PRESETS,
  type Preset,
  ORGANIC_TYPES,
  ANGLED_TYPES,
  defaultSpots,
  calculateOutputDimensions,
  shuffleColorsAndSpots,
  generateNoiseCSS,
  sanitizeColors,
  lighten,
  colorName,
} from "./noisefield-data";

export interface StopRow {
  color: string;
  pos: number;
}

const reduceMotion =
  typeof matchMedia === "function" && matchMedia("(prefers-reduced-motion: reduce)").matches;

class GeneratorStore {
  // Mode selection
  generatorMode = $state<"noise" | "multistop">("noise");

  // Generative noise gradient parameters
  field = $state("flow");
  style = $state("duotone");
  texture = $state("smooth");
  direction = $state(135);
  scale = $state(50);
  distortion = $state(55);
  seed = $state(7.3);
  animate = $state(!reduceMotion);
  ratio = $state("16/9");
  colors = $state<string[]>([...PRESETS[0].colors]);
  spots = $state<Array<[number, number]>>(defaultSpots(PRESETS[0].colors.length));

  // Multi-stop / CSS gradient parameters
  msSpace = $state("oklab");
  msHue = $state("shorter");
  msEasing = $state("linear");
  msKind = $state("linear");
  msAngle = $state(90);
  msNative = $state(false);
  msSamples = $state(16);
  msStops = $state<StopRow[]>([
    { color: "#0ea5e9", pos: 0 },
    { color: "#8b5cf6", pos: 1 },
  ]);
  msCSS = $state("");

  // Status & copy feedback
  statusMessage = $state("");
  copiedCSS = $state(false);

  // Internal clock & render state
  #clock = 0;
  #lastTime = 0;
  #rafId = 0;
  #pending = false;
  #drawSeq = 0;
  #statusTimer = 0;

  constructor() {
    if (typeof document !== "undefined") {
      document.addEventListener("visibilitychange", () => {
        if (document.hidden) {
          this.stopAnimation();
        } else if (this.animate && this.isOrganic && this.generatorMode === "noise") {
          this.startAnimation();
        }
      });
    }
  }

  hydrateFromURL(gen: GeneratorURLState) {
    if (gen.tool) this.generatorMode = gen.tool;
    if (gen.tool === "noise") {
      if (gen.field) this.field = gen.field;
      if (gen.style) this.style = gen.style;
      if (gen.texture) this.texture = gen.texture;
      if (gen.angle !== undefined) this.direction = gen.angle;
      if (gen.scale !== undefined) this.scale = gen.scale;
      if (gen.distortion !== undefined) this.distortion = gen.distortion;
      if (gen.seed !== undefined) this.seed = gen.seed;
      if (gen.ratio) this.ratio = gen.ratio;
      if (gen.colors && gen.colors.length) this.colors = [...gen.colors];
      if (gen.spots && gen.spots.length) this.spots = [...gen.spots];
    } else {
      if (gen.space) this.msSpace = gen.space;
      if (gen.hue) this.msHue = gen.hue;
      if (gen.easing) this.msEasing = gen.easing;
      if (gen.kind) this.msKind = gen.kind;
      if (gen.angle !== undefined) this.msAngle = gen.angle;
      if (gen.stops && gen.stops.length) this.msStops = [...gen.stops];
    }
  }

  updateURL() {
    if (ui.mode !== "generator") return;
    if (this.generatorMode === "noise") {
      writeGeneratorStateToURL({
        tool: "noise",
        field: this.field,
        style: this.style,
        texture: this.texture,
        angle: this.direction,
        scale: this.scale,
        distortion: this.distortion,
        seed: this.seed,
        ratio: this.ratio,
        colors: this.colors,
        spots: this.spots,
      });
    } else {
      writeGeneratorStateToURL({
        tool: "multistop",
        space: this.msSpace,
        hue: this.msHue,
        easing: this.msEasing,
        kind: this.msKind,
        angle: this.msAngle,
        stops: this.msStops,
      });
    }
    ui.query = location.search;
  }

  // Derived properties
  isOrganic = $derived(ORGANIC_TYPES.has(this.field.toLowerCase()));
  isAngled = $derived(ANGLED_TYPES.has(this.field.toLowerCase()));

  aspectRatio = $derived.by(() => {
    const parts = this.ratio.split("/").map(Number);
    return (parts[0] || 16) / (parts[1] || 9);
  });

  previewDimensions = $derived.by(() => {
    return calculateOutputDimensions(this.ratio, 960);
  });

  exportDimensions = $derived.by(() => {
    return calculateOutputDimensions(this.ratio, 1600);
  });

  cssOutput = $derived.by(() => {
    if (this.generatorMode === "multistop") {
      return this.msCSS;
    }
    return generateNoiseCSS({
      field: this.field,
      style: this.style,
      texture: this.texture,
      direction: this.direction,
      colors: this.colors,
      spots: this.spots,
      ratio: this.ratio,
      scale: this.scale,
      distortion: this.distortion,
    });
  });

  matchedPreset = $derived.by((): Preset | null => {
    for (const p of PRESETS) {
      if (
        p.genre.toLowerCase() === this.style.toLowerCase() &&
        p.type.toLowerCase() === this.field.toLowerCase() &&
        p.texture.toLowerCase() === this.texture.toLowerCase() &&
        p.direction === this.direction &&
        p.colors.length === this.colors.length
      ) {
        const ok = p.colors.every(
          (c, idx) => c.toUpperCase() === (this.colors[idx] || "").toUpperCase(),
        );
        if (ok) return p;
      }
    }
    return null;
  });

  colorNames = $derived.by(() => {
    return this.colors.map((c) => colorName(c));
  });

  // Actions
  setStatus(msg: string) {
    this.statusMessage = msg;
    clearTimeout(this.#statusTimer);
    if (msg) {
      this.#statusTimer = window.setTimeout(() => {
        this.statusMessage = "";
      }, 4000);
    }
  }

  applyPreset(preset: Preset) {
    this.field = preset.type.toLowerCase();
    this.style = preset.genre.toLowerCase();
    this.texture = preset.texture.toLowerCase();
    this.direction = preset.direction;
    this.colors = [...preset.colors];
    this.spots = defaultSpots(preset.colors.length);
    this.scale = 50;
    this.distortion = 55;
    this.seed = 7.3;
    this.scheduleRender();
    this.syncAnimation();
  }

  shuffle() {
    const res = shuffleColorsAndSpots(this.colors.length);
    this.colors = res.colors;
    this.spots = res.spots;
    this.seed = res.seed;
    this.setStatus("Cores e spots sorteados harmonicamente.");
    this.scheduleRender();
  }

  addColor(hex?: string) {
    if (this.colors.length >= 8) return;
    const n = this.colors.length;
    const defaultColor = hex ?? lighten(this.colors[n - 1] ?? "#888888", 0.3);
    this.colors.push(defaultColor);
    const def = defaultSpots(8)[n] ?? [0.5, 0.5];
    this.spots.push([def[0], def[1]]);
    this.scheduleRender();
  }

  removeColor(index: number) {
    if (this.colors.length <= 2) return;
    this.colors.splice(index, 1);
    this.spots.splice(index, 1);
    this.scheduleRender();
  }

  updateColor(index: number, hex: string) {
    this.colors[index] = hex;
    this.scheduleRender();
  }

  updateSpot(index: number, x: number, y: number) {
    if (index >= 0 && index < this.spots.length) {
      this.spots[index] = [
        Math.max(0.02, Math.min(0.98, x)),
        Math.max(0.02, Math.min(0.98, y)),
      ];
      this.scheduleRender();
    }
  }

  setField(f: string) {
    this.field = f.toLowerCase();
    this.scheduleRender();
    this.syncAnimation();
  }

  setStyle(s: string) {
    this.style = s.toLowerCase();
    this.scheduleRender();
  }

  setTexture(t: string) {
    this.texture = t.toLowerCase();
    this.scheduleRender();
  }

  setDirection(deg: number) {
    this.direction = deg;
    this.scheduleRender();
  }

  setRatio(r: string) {
    this.ratio = r;
    this.scheduleRender();
  }

  setScale(sc: number) {
    this.scale = sc;
    this.scheduleRender();
  }

  setDistortion(dist: number) {
    this.distortion = dist;
    this.scheduleRender();
  }

  toggleAnimation() {
    this.animate = !this.animate;
    this.syncAnimation();
  }

  startAnimation() {
    if (this.#rafId || !this.animate || !this.isOrganic || this.generatorMode !== "noise") return;
    this.#lastTime = performance.now();
    const frame = (now: number) => {
      this.#rafId = 0;
      if (!this.animate || !this.isOrganic || this.generatorMode !== "noise" || document.hidden) {
        return;
      }
      this.#clock += Math.min(0.1, (now - this.#lastTime) / 1000);
      this.#lastTime = now;
      void this.renderFrame();
      this.#rafId = requestAnimationFrame(frame);
    };
    this.#rafId = requestAnimationFrame(frame);
  }

  stopAnimation() {
    if (this.#rafId) {
      cancelAnimationFrame(this.#rafId);
      this.#rafId = 0;
    }
  }

  syncAnimation() {
    if (this.animate && this.isOrganic && this.generatorMode === "noise") {
      this.startAnimation();
    } else {
      this.stopAnimation();
    }
  }

  scheduleRender() {
    this.updateURL();
    if (this.#pending) return;
    this.#pending = true;
    requestAnimationFrame(() => {
      this.#pending = false;
      void this.renderFrame();
    });
  }

  getNoiseParams(clockTime?: number): NoiseFieldParams {
    const isOrganic = this.isOrganic;
    const cleanColors = sanitizeColors(this.colors);
    return {
      field: this.field.toLowerCase(),
      style: this.style.toLowerCase(),
      texture: this.texture.toLowerCase(),
      stops: isOrganic ? [] : cleanColors,
      spots: isOrganic
        ? cleanColors.map((c, i) => {
            const sp = this.spots[i] ?? [0.5, 0.5];
            return { color: c, x: sp[0], y: sp[1] };
          })
        : [],
      angle: this.direction,
      scale: this.scale,
      distortion: this.distortion,
      seed: this.seed,
      time: clockTime !== undefined ? clockTime : this.#clock,
    };
  }

  getGradientParams(w: number, h: number): GradientParams {
    const stops = [...this.msStops].sort((a, b) => a.pos - b.pos);
    return {
      stops,
      space: this.msSpace,
      hue: this.msHue,
      easing: this.msEasing.trim() || "linear",
      width: w,
      height: h,
      angle: this.msAngle,
    };
  }

  async renderFrame(): Promise<void> {
    const seq = ++this.#drawSeq;
    try {
      const backend = await getBackend(ui.settings.backend);
      const canvas = refs.canvas;
      if (!canvas) return;

      if (this.generatorMode === "noise") {
        const { w, h } = this.previewDimensions;
        const img = await backend.renderNoiseField(this.getNoiseParams(), w, h);
        if (seq !== this.#drawSeq) return;
        putImageData(canvas, img);
      } else {
        const { w, h } = this.previewDimensions;
        const p = this.getGradientParams(w, h);
        const [img, css] = await Promise.all([
          backend.renderGradient(p),
          backend
            .gradientCSS(p, {
              kind: this.msKind,
              angle: this.msAngle,
              native: this.msNative,
              samples: this.msSamples,
              selector: ".gradient",
              property: "background",
            })
            .catch((e) => `/* ${String(e)} */`),
        ]);
        if (seq !== this.#drawSeq) return;
        putImageData(canvas, img);
        this.msCSS = css;
      }
    } catch (err) {
      console.error("Generator render error:", err);
    }
  }

  async copyCSS(): Promise<void> {
    try {
      await navigator.clipboard.writeText(this.cssOutput);
      this.copiedCSS = true;
      this.setStatus("CSS copiado para a área de transferência!");
      setTimeout(() => {
        this.copiedCSS = false;
      }, 1800);
    } catch {
      this.setStatus("Não foi possível copiar o CSS.");
    }
  }

  async exportHighResPNG(): Promise<void> {
    this.setStatus("Gerando PNG em alta resolução (1600px)...");
    try {
      const backend = await getBackend(ui.settings.backend);
      const { w, h } = this.exportDimensions;

      let img: ImageData;
      let filename: string;

      if (this.generatorMode === "noise") {
        img = await backend.renderNoiseField(this.getNoiseParams(), w, h);
        filename = `bitbrush-gradient-${this.field}-${this.style}-${this.texture}-${w}x${h}.png`;
      } else {
        img = await backend.renderGradient(this.getGradientParams(w, h));
        filename = `bitbrush-gradient-perceptual-${w}x${h}.png`;
      }

      const tempCanvas = document.createElement("canvas");
      tempCanvas.width = img.width;
      tempCanvas.height = img.height;
      const ctx = tempCanvas.getContext("2d")!;
      ctx.putImageData(img, 0, 0);

      tempCanvas.toBlob((blob) => {
        if (!blob) {
          this.setStatus("Falha ao criar o arquivo PNG.");
          return;
        }
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        a.remove();
        setTimeout(() => URL.revokeObjectURL(url), 10_000);
        this.setStatus(`PNG ${w}×${h} px salvo com sucesso!`);
      }, "image/png");
    } catch (err) {
      console.error(err);
      this.setStatus("Erro ao exportar PNG.");
    }
  }
}

export const generatorStore = new GeneratorStore();
