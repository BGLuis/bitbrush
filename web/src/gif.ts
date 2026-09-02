// Animated-GIF export UI. The user captures the current filter params as a
// start ("A") and end ("B") keyframe; internal/anim interpolates every
// numeric param between them across N frames, applies the filter to each,
// and encodes a GIF. The render runs on the resolved backend (the Worker,
// normally) so a long encode doesn't freeze the page.
//
// GIFs are small by nature, so this has its own resolution cap independent
// of the live-preview one, and it always renders from the full-resolution
// source (internal/anim does the downscale).

import type { FilterParams, GifKeyframe, GifOptions } from "./wasm";

export interface GifPanelDeps {
  currentEffect: () => string;
  currentParams: () => FilterParams;
  source: () => ImageData | null;
  render: (name: string, img: ImageData, kfs: GifKeyframe[], opt: GifOptions) => Promise<Uint8Array>;
}

export interface GifPanel {
  readonly element: HTMLElement;
  /** Reset captured keyframes — call when the selected effect changes. */
  onEffectChange(): void;
}

export function renderGifPanel(deps: GifPanelDeps): GifPanel {
  let kfA: FilterParams | null = null;
  let kfB: FilterParams | null = null;

  const root = document.createElement("details");
  root.className = "gif-panel";
  const summary = document.createElement("summary");
  summary.textContent = "Exportar GIF animado";
  root.append(summary);

  const frames = numberField("Quadros", 24, 2, 120, 1);
  const fps = numberField("FPS", 15, 2, 30, 1);
  const maxDim = selectField("Tamanho máx.", "480", [
    ["240", "240 px"],
    ["360", "360 px"],
    ["480", "480 px"],
    ["720", "720 px"],
  ]);
  const loop = checkboxField("Repetir em loop", true);
  const pingPong = checkboxField("Ping-pong (A→B→A)", false);
  const animateSeed = checkboxField("Animar a semente (glitch)", true);

  const captureA = button("Capturar início (A)", () => {
    kfA = { ...deps.currentParams() };
    refresh();
  });
  const captureB = button("Capturar fim (B)", () => {
    kfB = { ...deps.currentParams() };
    refresh();
  });
  const clear = button("Limpar keyframes", () => {
    kfA = kfB = null;
    refresh();
  });
  const generate = button("Gerar GIF", generateGif);
  generate.classList.add("gif-generate");

  const status = document.createElement("p");
  status.className = "gif-status";

  const anchor = document.createElement("a");
  anchor.hidden = true;

  const grid = document.createElement("div");
  grid.className = "control-panel";
  grid.append(
    frames.row,
    fps.row,
    maxDim.row,
    loop.row,
    pingPong.row,
    animateSeed.row,
    rowOf(captureA, captureB, clear),
    rowOf(generate),
    status,
    anchor,
  );
  root.append(grid);

  refresh();

  function refresh(): void {
    const parts = [
      kfA ? "A ✓" : "A —",
      kfB ? "B ✓" : "B —",
    ];
    if (!kfA && !kfB) parts.push("(sem keyframes: usa os parâmetros atuais nos dois extremos)");
    status.textContent = parts.join("  ·  ");
  }

  async function generateGif(): Promise<void> {
    const src = deps.source();
    if (!src) {
      status.textContent = "Carregue uma imagem primeiro.";
      return;
    }
    const name = deps.currentEffect();
    const now = deps.currentParams();
    const n = clamp(Math.round(frames.value()), 2, 120);

    let a: FilterParams = { ...(kfA ?? now) };
    let b: FilterParams = { ...(kfB ?? now) };

    // With "animate seed" on, spread the integer seed one step per frame so
    // the noise actually moves. Linear interpolation of seed base..base+n-1
    // rounds to base, base+1, … which is exactly that.
    if (animateSeed.checked() && ("seed" in a || "seed" in b || "seed" in now)) {
      const base = Number(a.seed ?? now.seed ?? 0) || 0;
      a = { ...a, seed: base };
      b = { ...b, seed: base + n - 1 };
    }

    const keyframes: GifKeyframe[] = [
      { t: 0, params: a },
      { t: 1, params: b },
    ];
    const options: GifOptions = {
      frames: n,
      fps: clamp(Math.round(fps.value()), 2, 30),
      loop: loop.checked(),
      pingPong: pingPong.checked(),
      maxDimension: Number(maxDim.value()),
    };

    generate.disabled = true;
    status.textContent = "Gerando…";
    try {
      const bytes = await deps.render(name, src, keyframes, options);
      const buf = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer;
      const url = URL.createObjectURL(new Blob([buf], { type: "image/gif" }));
      anchor.href = url;
      anchor.download = `bitbrush-${name}.gif`;
      anchor.click();
      setTimeout(() => URL.revokeObjectURL(url), 15_000);
      status.textContent = `Pronto — ${(bytes.length / 1024).toFixed(0)} KB · ${n} quadros`;
    } catch (err) {
      console.error(err);
      status.textContent = `Falhou: ${String(err)}`;
    } finally {
      generate.disabled = false;
    }
  }

  return {
    element: root,
    onEffectChange() {
      kfA = kfB = null;
      refresh();
    },
  };
}

// --- tiny DOM helpers (kept local; the effect-panel renderer in ui/ is
// descriptor-driven and not a fit for this one-off form) ---

function clamp(n: number, lo: number, hi: number): number {
  return Math.min(hi, Math.max(lo, n));
}

function rowOf(...els: HTMLElement[]): HTMLElement {
  const d = document.createElement("div");
  d.className = "control";
  d.style.gridTemplateColumns = "repeat(auto-fit, minmax(0, max-content))";
  d.append(...els);
  return d;
}

function labelled(text: string): [HTMLElement, HTMLElement] {
  const row = document.createElement("label");
  row.className = "control";
  const span = document.createElement("span");
  span.className = "control-label";
  span.textContent = text;
  row.append(span);
  return [row, span];
}

function numberField(text: string, def: number, min: number, max: number, step: number) {
  const [row] = labelled(text);
  const input = document.createElement("input");
  input.type = "number";
  input.min = String(min);
  input.max = String(max);
  input.step = String(step);
  input.value = String(def);
  row.append(input);
  return { row, value: () => (input.value === "" ? def : Number(input.value)) };
}

function selectField(text: string, def: string, options: Array<[string, string]>) {
  const [row] = labelled(text);
  const el = document.createElement("select");
  for (const [v, label] of options) {
    const o = document.createElement("option");
    o.value = v;
    o.textContent = label;
    el.append(o);
  }
  el.value = def;
  row.append(el);
  return { row, value: () => el.value };
}

function checkboxField(text: string, def: boolean) {
  const [row, span] = labelled(text);
  const el = document.createElement("input");
  el.type = "checkbox";
  el.checked = def;
  row.insertBefore(el, span);
  return { row, checked: () => el.checked };
}

function button(text: string, onClick: () => void): HTMLButtonElement {
  const b = document.createElement("button");
  b.type = "button";
  b.textContent = text;
  b.addEventListener("click", onClick);
  return b;
}
