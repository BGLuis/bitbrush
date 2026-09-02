// Gradient generators: the multi-stop / CSS tool (internal/gradient) and the
// generative noise-field tool (internal/noisefield). Both draw straight to
// the main canvas and need no input image. Hand-built controls (stop lists,
// spot lists, live CSS, an animation loop) rather than the descriptor panel.

import type {
  GradientParams,
  GradientStop,
  GradientCSSOptions,
  NoiseFieldParams,
  NoiseFieldSpot,
} from "./wasm";
import {
  labelledRow,
  rowOf,
  numberInput,
  selectInput,
  checkboxInput,
  colorInput,
  textInput,
  button,
} from "./ui/widgets";

export interface GeneratorPanelDeps {
  renderGradient(params: GradientParams): Promise<ImageData>;
  gradientCSS(params: GradientParams, options: GradientCSSOptions): Promise<string>;
  renderNoiseField(params: NoiseFieldParams, w: number, h: number): Promise<ImageData>;
  /** Paint an ImageData onto the shared canvas. */
  show(img: ImageData): void;
}

export interface GeneratorPanel {
  readonly element: HTMLElement;
}

const SPACES: Array<[string, string]> = [
  ["oklab", "OKLab"],
  ["oklch", "OKLCh"],
  ["lab", "CIE Lab"],
  ["lch", "CIE LCh"],
  ["hsl", "HSL"],
  ["linear", "RGB linear"],
  ["srgb", "sRGB"],
];
const CYLINDRICAL = new Set(["hsl", "lch", "oklch"]);
const HUE_ARCS: Array<[string, string]> = [
  ["shorter", "arco curto"],
  ["longer", "arco longo"],
  ["increasing", "crescente"],
  ["decreasing", "decrescente"],
];
const CSS_KINDS: Array<[string, string]> = [
  ["linear", "linear"],
  ["radial", "radial"],
  ["conic", "conic"],
];
const FIELDS: Array<[string, string]> = [
  ["linear", "Linear"],
  ["radial", "Radial"],
  ["conic", "Cônico"],
  ["reflected", "Refletido"],
  ["diamond", "Diamante"],
  ["mesh", "Malha (multi-ponto)"],
  ["freeform", "Livre"],
  ["flow", "Fluxo (ruído)"],
];
const ORGANIC = new Set(["mesh", "freeform", "flow"]);
const STYLES: Array<[string, string]> = [
  ["duotone", "Duotone"],
  ["metallic", "Metálico"],
  ["chrome", "Cromado"],
  ["iridescent", "Irisado"],
  ["holographic", "Holográfico"],
  ["neon", "Neon"],
  ["pastel", "Pastel"],
  ["rainbow", "Arco-íris"],
];
const TEXTURES: Array<[string, string]> = [
  ["smooth", "Lisa"],
  ["grain", "Granulada"],
  ["frosted", "Fosca"],
  ["wave", "Onda"],
  ["wrinkle", "Amassada"],
  ["paper", "Papel"],
];
const SIZES: Array<[string, string]> = [
  ["640x360", "640×360"],
  ["960x540", "960×540"],
  ["1280x720", "1280×720"],
  ["1024x1024", "1024×1024"],
];

const reduceMotion =
  typeof matchMedia === "function" && matchMedia("(prefers-reduced-motion: reduce)").matches;

export function renderGeneratorPanel(deps: GeneratorPanelDeps): GeneratorPanel {
  const root = document.createElement("details");
  root.className = "gen-panel";
  const summary = document.createElement("summary");
  summary.textContent = "Gerar gradiente";
  root.append(summary);

  const body = document.createElement("div");
  body.className = "control-panel";
  root.append(body);

  const mode = selectInput(
    [
      ["multistop", "Multi-stop / CSS"],
      ["noise", "Generativo (ruído)"],
    ],
    "multistop",
  );
  const size = selectInput(SIZES, "960x540");
  const dims = () => {
    const [w, h] = size.get().split("x").map(Number);
    return { w, h };
  };

  // one shared rAF-coalesced render
  let pending = false;
  const schedule = () => {
    if (pending) return;
    pending = true;
    requestAnimationFrame(() => {
      pending = false;
      void draw();
    });
  };
  const bindLive = (el: HTMLElement) => {
    el.addEventListener("input", schedule);
    el.addEventListener("change", schedule);
  };

  // ---------- multi-stop ----------
  const msBox = document.createElement("div");
  const stopsHost = document.createElement("div");
  stopsHost.className = "stop-list";
  type StopRow = { color: ReturnType<typeof colorInput>; pos: ReturnType<typeof numberInput>; el: HTMLElement };
  const stopRows: StopRow[] = [];
  const addStop = (hex = "#8b5cf6", pos = stopRows.length === 0 ? 0 : 1) => {
    const color = colorInput(hex);
    const pos_ = numberInput(pos, 0, 1, 0.01);
    const remove = button("✕", () => {
      const i = stopRows.findIndex((r) => r.el === el);
      if (i >= 0 && stopRows.length > 2) {
        stopRows.splice(i, 1);
        el.remove();
        schedule();
      }
    });
    remove.className = "stop-remove";
    const el = rowOf(color.el, pos_.el, remove);
    bindLive(color.el);
    bindLive(pos_.el);
    stopRows.push({ color: color, pos: pos_, el });
    stopsHost.append(el);
  };
  addStop("#0ea5e9", 0);
  addStop("#8b5cf6", 1);

  const msSpace = selectInput(SPACES, "oklab");
  const msHue = selectInput(HUE_ARCS, "shorter");
  const msHueRow = labelledRow("Arco de matiz");
  msHueRow.slot.append(msHue.el);
  const msEasing = textInput("linear");
  msEasing.el.placeholder = "linear · ease-in-out · cubic-bezier(.4,0,.2,1) · steps(4)";
  const msKind = selectInput(CSS_KINDS, "linear");
  const msAngle = numberInput(90, 0, 360, 1);
  const msNative = checkboxInput(false);
  const msSamples = numberInput(16, 2, 64, 1);
  const msSamplesRow = labelledRow("Amostras (CSS assado)");
  msSamplesRow.slot.append(msSamples.el);

  for (const el of [
    msSpace.el,
    msHue.el,
    msEasing.el,
    msKind.el,
    msAngle.el,
    msNative.el,
    msSamples.el,
  ])
    bindLive(el);

  const addStopBtn = button("+ stop", () => {
    addStop();
    schedule();
  });

  const cssOut = document.createElement("textarea");
  cssOut.className = "css-out";
  cssOut.readOnly = true;
  cssOut.rows = 3;
  const copyCss = button("Copiar CSS", () => {
    void navigator.clipboard?.writeText(cssOut.value);
    copyCss.textContent = "Copiado!";
    setTimeout(() => (copyCss.textContent = "Copiar CSS"), 1200);
  });

  const lr = (label: string, ...els: HTMLElement[]) => {
    const { row, slot } = labelledRow(label);
    slot.append(...els);
    return row;
  };

  msBox.append(
    lr("Espaço de cor", msSpace.el),
    msHueRow.row,
    lr("Easing", msEasing.el),
    lr("Tipo CSS", msKind.el),
    lr("Ângulo", msAngle.el),
    lr("CSS nativo (in oklch)", msNative.el),
    msSamplesRow.row,
    stopsHost,
    rowOf(addStopBtn),
    cssOut,
    rowOf(copyCss),
  );

  // ---------- noise field ----------
  const nfBox = document.createElement("div");
  nfBox.hidden = true;
  const nfField = selectInput(FIELDS, "flow");
  const nfStyle = selectInput(STYLES, "holographic");
  const nfTexture = selectInput(TEXTURES, "smooth");
  const nfAngle = numberInput(135, 0, 360, 1);
  const nfScale = numberInput(50, 0, 100, 1);
  const nfDistortion = numberInput(55, 0, 100, 1);
  const nfSeed = numberInput(7, 0, 999999, 1);
  const nfAnimate = checkboxInput(!reduceMotion);
  for (const el of [
    nfField.el,
    nfStyle.el,
    nfTexture.el,
    nfAngle.el,
    nfScale.el,
    nfDistortion.el,
    nfSeed.el,
  ])
    bindLive(el);
  nfField.el.addEventListener("change", () => syncNoiseVisibility());
  nfAnimate.el.addEventListener("change", () => {
    if (nfAnimate.get()) startAnim();
    else stopAnim();
  });

  // geometric stops (reuse a compact editor)
  const nfStopsHost = document.createElement("div");
  nfStopsHost.className = "stop-list";
  type ColorRow = { color: ReturnType<typeof colorInput>; el: HTMLElement };
  const nfStopRows: ColorRow[] = [];
  const addNfStop = (hex = "#f59e0b") => {
    const color = colorInput(hex);
    bindLive(color.el);
    const remove = button("✕", () => {
      const i = nfStopRows.findIndex((r) => r.el === el);
      if (i >= 0 && nfStopRows.length > 1) {
        nfStopRows.splice(i, 1);
        el.remove();
        schedule();
      }
    });
    remove.className = "stop-remove";
    const el = rowOf(color.el, remove);
    nfStopRows.push({ color, el });
    nfStopsHost.append(el);
  };
  addNfStop("#0ea5e9");
  addNfStop("#8b5cf6");
  const addNfStopBtn = button("+ cor", () => {
    addNfStop();
    schedule();
  });
  const nfStopsRow = document.createElement("div");
  nfStopsRow.append(nfStopsHost, rowOf(addNfStopBtn));

  // organic spots: colour + x + y
  const nfSpotsHost = document.createElement("div");
  nfSpotsHost.className = "stop-list";
  type SpotRow = {
    color: ReturnType<typeof colorInput>;
    x: ReturnType<typeof numberInput>;
    y: ReturnType<typeof numberInput>;
    el: HTMLElement;
  };
  const nfSpotRows: SpotRow[] = [];
  const DEF_SPOTS: Array<[string, number, number]> = [
    ["#f472b6", 0.2, 0.26],
    ["#38bdf8", 0.8, 0.22],
    ["#a78bfa", 0.72, 0.8],
    ["#4ade80", 0.24, 0.76],
  ];
  const addSpot = (hex = "#ffffff", x = 0.5, y = 0.5) => {
    const color = colorInput(hex);
    const xi = numberInput(x, 0, 1, 0.01);
    const yi = numberInput(y, 0, 1, 0.01);
    const remove = button("✕", () => {
      const i = nfSpotRows.findIndex((r) => r.el === el);
      if (i >= 0 && nfSpotRows.length > 1) {
        nfSpotRows.splice(i, 1);
        el.remove();
        schedule();
      }
    });
    remove.className = "stop-remove";
    const el = rowOf(color.el, xi.el, yi.el, remove);
    for (const c of [color.el, xi.el, yi.el]) bindLive(c);
    nfSpotRows.push({ color, x: xi, y: yi, el });
    nfSpotsHost.append(el);
  };
  for (const [h, x, y] of DEF_SPOTS) addSpot(h, x, y);
  const addSpotBtn = button("+ ponto", () => {
    addSpot();
    schedule();
  });
  const nfSpotsRow = document.createElement("div");
  nfSpotsRow.append(nfSpotsHost, rowOf(addSpotBtn));

  function syncNoiseVisibility(): void {
    const organic = ORGANIC.has(nfField.get());
    nfStopsRow.hidden = organic;
    nfSpotsRow.hidden = !organic;
    schedule();
  }

  nfBox.append(
    lr("Campo", nfField.el),
    lr("Estilo", nfStyle.el),
    lr("Textura", nfTexture.el),
    lr("Ângulo", nfAngle.el),
    lr("Escala", nfScale.el),
    lr("Distorção", nfDistortion.el),
    lr("Semente", nfSeed.el),
    lr("Animar", nfAnimate.el),
    nfStopsRow,
    nfSpotsRow,
  );

  // ---------- animation loop (noise only) ----------
  let raf = 0;
  let last = 0;
  let clock = 0;
  const frame = (now: number) => {
    raf = 0;
    if (!nfAnimate.get() || mode.get() !== "noise" || document.hidden) return;
    clock += Math.min(0.1, (now - last) / 1000);
    last = now;
    void draw();
    raf = requestAnimationFrame(frame);
  };
  const startAnim = () => {
    if (raf || mode.get() !== "noise") return;
    last = performance.now();
    raf = requestAnimationFrame(frame);
  };
  const stopAnim = () => {
    if (raf) cancelAnimationFrame(raf);
    raf = 0;
  };
  document.addEventListener("visibilitychange", () => {
    if (document.hidden) stopAnim();
    else if (nfAnimate.get()) startAnim();
  });

  // ---------- params + draw ----------
  const gradientParams = (): GradientParams => {
    const { w, h } = dims();
    const stops: GradientStop[] = stopRows
      .map((r) => ({ color: r.color.get(), pos: r.pos.get() }))
      .sort((a, b) => a.pos - b.pos);
    return {
      stops,
      space: msSpace.get(),
      hue: msHue.get(),
      easing: msEasing.get().trim() || "linear",
      width: w,
      height: h,
      angle: msAngle.get(),
    };
  };
  const noiseParams = (): NoiseFieldParams => {
    const organic = ORGANIC.has(nfField.get());
    const spots: NoiseFieldSpot[] = nfSpotRows.map((r) => ({
      color: r.color.get(),
      x: r.x.get(),
      y: r.y.get(),
    }));
    return {
      field: nfField.get(),
      style: nfStyle.get(),
      texture: nfTexture.get(),
      stops: organic ? [] : nfStopRows.map((r) => r.color.get()),
      spots: organic ? spots : [],
      angle: nfAngle.get(),
      scale: nfScale.get(),
      distortion: nfDistortion.get(),
      seed: nfSeed.get(),
      time: clock,
    };
  };

  let drawSeq = 0;
  async function draw(): Promise<void> {
    const seq = ++drawSeq;
    try {
      if (mode.get() === "multistop") {
        const p = gradientParams();
        const [img, css] = await Promise.all([
          deps.renderGradient(p),
          deps
            .gradientCSS(p, {
              kind: msKind.get(),
              angle: msAngle.get(),
              native: msNative.get(),
              samples: msSamples.get(),
              selector: ".gradient",
              property: "background",
            })
            .catch((e) => `/* ${String(e)} */`),
        ]);
        if (seq !== drawSeq) return;
        deps.show(img);
        cssOut.value = css;
      } else {
        const { w, h } = dims();
        const img = await deps.renderNoiseField(noiseParams(), w, h);
        if (seq !== drawSeq) return;
        deps.show(img);
      }
    } catch (err) {
      console.error(err);
    }
  }

  const pngBtn = button("Baixar PNG", () => {
    const { w, h } = dims();
    void (mode.get() === "multistop"
      ? deps.renderGradient(gradientParams())
      : deps.renderNoiseField(noiseParams(), w, h)
    ).then((img) => downloadPng(img, `bitbrush-${mode.get()}.png`));
  });

  // ---------- mode toggle ----------
  const syncMode = () => {
    const noise = mode.get() === "noise";
    msBox.hidden = noise;
    nfBox.hidden = !noise;
    msHueRow.row.hidden = !CYLINDRICAL.has(msSpace.get());
    msSamplesRow.row.hidden = msNative.get();
    if (noise) {
      syncNoiseVisibility();
      if (nfAnimate.get()) startAnim();
    } else {
      stopAnim();
    }
    schedule();
  };
  mode.el.addEventListener("change", syncMode);
  msSpace.el.addEventListener("change", () => {
    msHueRow.row.hidden = !CYLINDRICAL.has(msSpace.get());
  });
  msNative.el.addEventListener("change", () => {
    msSamplesRow.row.hidden = msNative.get();
  });
  root.addEventListener("toggle", () => {
    if (root.open) syncMode();
    else stopAnim();
  });

  body.append(lr("Ferramenta", mode.el), lr("Tamanho", size.el), msBox, nfBox, rowOf(pngBtn));
  bindLive(size.el);
  syncMode();

  return { element: root };
}

function downloadPng(img: ImageData, filename: string): void {
  const c = document.createElement("canvas");
  c.width = img.width;
  c.height = img.height;
  c.getContext("2d")!.putImageData(img, 0, 0);
  c.toBlob((blob) => {
    if (!blob) return;
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 15_000);
  }, "image/png");
}
