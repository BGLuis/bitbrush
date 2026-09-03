// Runes-based reactive store for the Gradient & Noise generator,
// managing state, presets, animation, canvas rendering, spots and export.

import { ui, refs } from "../store.svelte";
import { getBackend } from "../../backend";
import { putImageData } from "../../canvas";
import { writeGeneratorStateToURL, type GeneratorURLState } from "../../state";
import type { GradientParams, NoiseFieldParams } from "../../wasm";
import { generators, type GeneratorUI } from "../../ui/generators";
import {
  type PatternPreset,
  getPatternPresets,
  buildDefaultParams,
} from "./patterns-data";
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
  generatorMode = $state<"noise" | "multistop" | "patterns">("noise");

  // Algorithmic pattern parameters
  selectedPattern = $state("contours");
  patternParams = $state<Record<string, Record<string, any>>>(buildDefaultParams());

  // Generative noise gradient parameters
  field = $state("flow");
  style = $state("duotone");
  texture = $state("smooth");
  direction = $state(135);
  scale = $state(50);
  distortion = $state(55);
  seed = $state(0);
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
  #patternClock = 0;
  #lastTime = 0;
  #rafId = 0;
  #pending = false;
  #inFlight = false;
  #queuedRender = false;
  #drawSeq = 0;
  #statusTimer = 0;

  constructor() {
    if (typeof document !== "undefined") {
      document.addEventListener("visibilitychange", () => {
        if (document.hidden) {
          this.stopAnimation();
        } else if (
          this.animate &&
          ((this.isOrganic && this.generatorMode === "noise") ||
            (this.patternHasTime && this.generatorMode === "patterns"))
        ) {
          this.startAnimation();
        }
      });
    }
  }

  hydrateFromURL(gen: GeneratorURLState) {
    if (gen.tool) this.generatorMode = gen.tool;
    if (gen.tool === "patterns") {
      if (gen.pattern) this.selectedPattern = gen.pattern;
      if (gen.ratio) this.ratio = gen.ratio;
      if (gen.patternParams) {
        if (!this.patternParams[this.selectedPattern]) {
          this.patternParams[this.selectedPattern] = {};
        }
        Object.assign(this.patternParams[this.selectedPattern], gen.patternParams);
      }
    } else if (gen.tool === "noise") {
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
    if (ui.mode !== "generator" && ui.mode !== "pattern") return;
    if (this.generatorMode === "patterns" || ui.mode === "pattern") {
      writeGeneratorStateToURL({
        tool: "patterns",
        pattern: this.selectedPattern,
        ratio: this.ratio,
        patternParams: this.patternParams[this.selectedPattern],
      });
    } else if (this.generatorMode === "noise") {
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

  currentPatternUI = $derived.by((): GeneratorUI => {
    return generators.find((g) => g.name === this.selectedPattern) ?? generators[0];
  });

  currentPatternParams = $derived.by(() => {
    return this.patternParams[this.selectedPattern] ?? {};
  });

  currentPatternPresets = $derived.by(() => {
    return getPatternPresets(this.selectedPattern);
  });

  patternHasTime = $derived.by(() => {
    return this.currentPatternUI.controls.some((c) => c.key === "time");
  });

  aspectRatio = $derived.by(() => {
    const parts = this.ratio.split("/").map(Number);
    return (parts[0] || 16) / (parts[1] || 9);
  });

  previewDimensions = $derived.by(() => {
    const cap = ui.settings.previewMaxDim > 0 ? ui.settings.previewMaxDim : 720;
    return calculateOutputDimensions(this.ratio, cap);
  });

  exportDimensions = $derived.by(() => {
    return calculateOutputDimensions(this.ratio, 1600);
  });

  patternDimensions = $derived.by(() => {
    const defW = this.currentPatternUI.defaultSize.width;
    const defH = this.currentPatternUI.defaultSize.height;
    const cap = ui.settings.previewMaxDim > 0 ? ui.settings.previewMaxDim : 720;
    const scale = Math.min(1, cap / Math.max(defW, defH));
    return {
      w: Math.max(64, Math.round(defW * scale)),
      h: Math.max(64, Math.round(defH * scale)),
    };
  });

  patternExportDimensions = $derived.by(() => {
    const defW = this.currentPatternUI.defaultSize.width;
    const defH = this.currentPatternUI.defaultSize.height;
    const target = 1600;
    const scale = target / Math.max(defW, defH);
    return {
      w: Math.round(defW * scale),
      h: Math.round(defH * scale),
    };
  });

  selectPattern(name: string) {
    this.selectedPattern = name;
    this.ratio = name === "contours" || name === "flowfield" ? "16/10" : "1/1";
    this.#patternClock = 0;
    this.scheduleRender();
    this.syncAnimation();
  }

  setPatternParam(key: string, val: any) {
    if (!this.patternParams[this.selectedPattern]) {
      this.patternParams[this.selectedPattern] = {};
    }
    this.patternParams[this.selectedPattern][key] = val;
    this.scheduleRender();
  }

  rollPatternSeed() {
    const p = this.patternParams[this.selectedPattern];
    if (p && "seed" in p) {
      p.seed = Math.floor(Math.random() * 1_000_000);
      this.setStatus(`Nova semente: ${p.seed}`);
      this.scheduleRender();
    }
  }

  applyPatternPreset(preset: PatternPreset) {
    this.selectedPattern = preset.pattern;
    if (!this.patternParams[preset.pattern]) {
      this.patternParams[preset.pattern] = {};
    }
    Object.assign(this.patternParams[preset.pattern], preset.params);
    this.setStatus(`Preset aplicado: ${preset.name}`);
    this.scheduleRender();
    this.syncAnimation();
  }

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
    this.seed = 0;
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
    if (!this.animate) {
      this.#patternClock = 0;
    }
    this.syncAnimation();
  }

  startAnimation() {
    const canAnimate =
      (this.generatorMode === "noise" && this.isOrganic) ||
      (this.generatorMode === "patterns" && this.patternHasTime);
    if (this.#rafId || !this.animate || !canAnimate) return;
    this.#lastTime = performance.now();
    const frame = async (now: number) => {
      this.#rafId = 0;
      const canLoop =
        (this.generatorMode === "noise" && this.isOrganic) ||
        (this.generatorMode === "patterns" && this.patternHasTime);
      if (!this.animate || !canLoop || (typeof document !== "undefined" && document.hidden)) {
        return;
      }
      const delta = Math.min(0.1, (now - this.#lastTime) / 1000);
      this.#clock += delta;
      this.#lastTime = now;
      if (this.generatorMode === "patterns" && this.patternHasTime) {
        this.#patternClock += delta * 0.4;
      }
      if (!this.#inFlight) {
        await this.renderFrame();
      }
      const stillActive =
        (this.generatorMode === "noise" && this.isOrganic) ||
        (this.generatorMode === "patterns" && this.patternHasTime);
      if (this.animate && stillActive && !(typeof document !== "undefined" && document.hidden)) {
        this.#rafId = requestAnimationFrame(frame);
      }
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
    const canAnimate =
      (this.generatorMode === "noise" && this.isOrganic) ||
      (this.generatorMode === "patterns" && this.patternHasTime);
    if (this.animate && canAnimate) {
      this.startAnimation();
    } else {
      this.stopAnimation();
    }
  }

  scheduleRender() {
    this.updateURL();
    if (this.#inFlight) {
      this.#queuedRender = true;
      return;
    }
    if (this.#pending) return;
    this.#pending = true;
    requestAnimationFrame(() => {
      this.#pending = false;
      if (!this.#inFlight) {
        void this.renderFrame();
      }
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
            return { color: c, x: sp[0], y: 1.0 - sp[1] };
          })
        : [],
      angle: this.direction,
      scale: this.scale,
      distortion: this.distortion,
      seed: Math.round(this.seed),
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
    if (this.#inFlight) {
      this.#queuedRender = true;
      return;
    }
    this.#inFlight = true;
    this.#queuedRender = false;
    const seq = ++this.#drawSeq;
    try {
      // Prioritize WebGL2 (GPU) in "auto" mode for noisefield rendering when available
      const pref = ui.settings.backend === "auto" ? "gpu" : ui.settings.backend;
      let backend;
      try {
        backend = await getBackend(pref);
      } catch {
        backend = await getBackend("auto");
      }
      const canvas = refs.canvas;
      if (!canvas) return;

      if (this.generatorMode === "patterns") {
        const { w, h } = this.patternDimensions;
        const raw = this.patternParams[this.selectedPattern] ?? {};
        const params = { ...raw };
        if (this.animate && this.patternHasTime && "time" in params) {
          params.time = Number(((params.time ?? 0) + this.#patternClock).toFixed(3));
        }
        // Run pattern calculations on the background worker pool to keep the UI thread 100% free
        const workerBackend = await getBackend("cpu").catch(() => backend);
        const img = await workerBackend.renderGenerator(
          this.selectedPattern,
          params,
          w,
          h,
        );
        if (seq !== this.#drawSeq) return;
        putImageData(canvas, img);
      } else if (this.generatorMode === "noise") {
        const { w, h } = this.previewDimensions;
        if (backend.renderNoiseFieldToCanvas) {
          backend.renderNoiseFieldToCanvas(canvas, this.getNoiseParams(), w, h);
        } else {
          const img = await backend.renderNoiseField(this.getNoiseParams(), w, h);
          if (seq !== this.#drawSeq) return;
          putImageData(canvas, img);
        }
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
    } finally {
      this.#inFlight = false;
      if (this.#queuedRender) {
        this.#queuedRender = false;
        void this.renderFrame();
      }
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
      let img: ImageData;
      let filename: string;

      if (this.generatorMode === "patterns") {
        const { w, h } = this.patternExportDimensions;
        const raw = this.patternParams[this.selectedPattern] ?? {};
        const params = { ...raw };
        if (this.animate && this.patternHasTime && "time" in params) {
          params.time = Number(((params.time ?? 0) + this.#patternClock).toFixed(3));
        }
        img = await backend.renderGenerator(
          this.selectedPattern,
          params,
          w,
          h,
        );
        filename = `bitbrush-${this.selectedPattern}-${w}x${h}.png`;
      } else if (this.generatorMode === "noise") {
        const { w, h } = this.exportDimensions;
        img = await backend.renderNoiseField(this.getNoiseParams(), w, h);
        filename = `bitbrush-gradient-${this.field}-${this.style}-${this.texture}-${w}x${h}.png`;
      } else {
        const { w, h } = this.exportDimensions;
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
        this.setStatus(`PNG ${img.width}×${img.height} px salvo com sucesso!`);
      }, "image/png");
    } catch (err) {
      console.error(err);
      this.setStatus("Erro ao exportar PNG.");
    }
  }

  exportSVG(): void {
    const { w, h } = this.exportDimensions;
    let svgContent = "";

    if (this.generatorMode === "multistop") {
      const rad = ((this.msAngle - 90) * Math.PI) / 180;
      const x1 = Math.round(50 - Math.cos(rad) * 50);
      const y1 = Math.round(50 - Math.sin(rad) * 50);
      const x2 = Math.round(50 + Math.cos(rad) * 50);
      const y2 = Math.round(50 + Math.sin(rad) * 50);

      const stopsXML = [...this.msStops]
        .sort((a, b) => a.pos - b.pos)
        .map((s) => `      <stop offset="${(s.pos * 100).toFixed(1)}%" stop-color="${s.color}" />`)
        .join("\n");

      svgContent = `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="0 0 ${w} ${h}">
  <defs>
    <linearGradient id="bitbrush-grad" x1="${x1}%" y1="${y1}%" x2="${x2}%" y2="${y2}%">
${stopsXML}
    </linearGradient>
  </defs>
  <rect width="100%" height="100%" fill="url(#bitbrush-grad)" />
</svg>`;
    } else {
      const colors = sanitizeColors(this.colors);
      const stopsXML = colors
        .map((c, i) => {
          const pct = ((i / Math.max(1, colors.length - 1)) * 100).toFixed(1);
          return `      <stop offset="${pct}%" stop-color="${c}" />`;
        })
        .join("\n");

      const rad = ((this.direction - 90) * Math.PI) / 180;
      const x1 = Math.round(50 - Math.cos(rad) * 50);
      const y1 = Math.round(50 - Math.sin(rad) * 50);
      const x2 = Math.round(50 + Math.cos(rad) * 50);
      const y2 = Math.round(50 + Math.sin(rad) * 50);

      svgContent = `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="0 0 ${w} ${h}">
  <defs>
    <linearGradient id="bitbrush-grad" x1="${x1}%" y1="${y1}%" x2="${x2}%" y2="${y2}%">
${stopsXML}
    </linearGradient>
  </defs>
  <rect width="100%" height="100%" fill="url(#bitbrush-grad)" />
</svg>`;
    }

    const blob = new Blob([svgContent], { type: "image/svg+xml;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `bitbrush-gradient-${this.generatorMode}.svg`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(url), 10_000);
    this.setStatus("Vetor SVG baixado com sucesso!");
  }
}

export const generatorStore = new GeneratorStore();
