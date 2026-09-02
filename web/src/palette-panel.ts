// Palette panel: extract a palette from the loaded image (median-cut /
// k-means, internal/palette) or generate one from a base colour by harmony
// rule. Results show as clickable swatches.

import type { PaletteExtractOptions, PaletteHarmonyOptions } from "./wasm";
import {
  labelledRow,
  rowOf,
  numberInput,
  selectInput,
  colorInput,
  button,
  swatchStrip,
} from "./ui/widgets";

export interface PalettePanelDeps {
  extractPalette(img: ImageData, options: PaletteExtractOptions): Promise<string[]>;
  genPalette(options: PaletteHarmonyOptions): Promise<string[]>;
  /** The full-resolution loaded image, or null. */
  source(): ImageData | null;
}

export interface PalettePanel {
  readonly element: HTMLElement;
}

const METHODS: Array<[string, string]> = [
  ["median-cut", "Median-cut"],
  ["kmeans", "k-means"],
];
const SPACES: Array<[string, string]> = [
  ["oklab", "OKLab"],
  ["srgb", "sRGB"],
];
const SORTS: Array<[string, string]> = [
  ["population", "por população"],
  ["luma", "por luminância"],
  ["hue", "por matiz"],
  ["none", "sem ordenar"],
];
const RULES: Array<[string, string]> = [
  ["complementary", "Complementar"],
  ["analogous", "Análoga"],
  ["triadic", "Tríade"],
  ["tetradic", "Tétrade"],
  ["split", "Split-complementar"],
  ["monochromatic", "Monocromática"],
];
const WHEELS: Array<[string, string]> = [
  ["oklch", "OKLCh (perceptual)"],
  ["hsl", "HSL"],
];

export function renderPalettePanel(deps: PalettePanelDeps): PalettePanel {
  const root = document.createElement("details");
  root.className = "pal-panel";
  const summary = document.createElement("summary");
  summary.textContent = "Paletas";
  root.append(summary);

  const body = document.createElement("div");
  body.className = "control-panel";
  root.append(body);

  const lr = (label: string, ...els: HTMLElement[]) => {
    const { row, slot } = labelledRow(label);
    slot.append(...els);
    return row;
  };

  // ---------- extract ----------
  const exCount = numberInput(8, 2, 32, 1);
  const exMethod = selectInput(METHODS, "median-cut");
  const exSpace = selectInput(SPACES, "oklab");
  const exSort = selectInput(SORTS, "population");
  const exSeed = numberInput(1, 0, 999999, 1);
  const exSeedRow = lr("Semente (k-means)", exSeed.el);
  exSeedRow.hidden = true;
  exMethod.el.addEventListener("change", () => {
    exSeedRow.hidden = exMethod.get() !== "kmeans";
  });

  const exStatus = document.createElement("p");
  exStatus.className = "gif-status";
  const exSwatches = swatchStrip();

  const exBtn = button("Extrair da imagem", async () => {
    const img = deps.source();
    if (!img) {
      exStatus.textContent = "Carregue uma imagem primeiro.";
      return;
    }
    exBtn.disabled = true;
    exStatus.textContent = "Extraindo…";
    try {
      const cols = await deps.extractPalette(img, {
        count: exCount.get(),
        method: exMethod.get(),
        space: exSpace.get(),
        sort: exSort.get(),
        alphaThreshold: 0,
        seed: exSeed.get(),
      });
      exSwatches.update(cols);
      exStatus.textContent = `${cols.length} cores`;
    } catch (err) {
      console.error(err);
      exStatus.textContent = `Falhou: ${String(err)}`;
    } finally {
      exBtn.disabled = false;
    }
  });

  // ---------- harmony ----------
  const hmBase = colorInput("#3b82f6");
  const hmRule = selectInput(RULES, "analogous");
  const hmN = numberInput(5, 1, 12, 1);
  const hmWheel = selectInput(WHEELS, "oklch");
  const hmSpread = numberInput(30, 5, 90, 1);
  const hmSpreadRow = lr("Passo de matiz", hmSpread.el);
  const hmSwatches = swatchStrip();

  const syncSpread = () => {
    hmSpreadRow.hidden = !["analogous", "split"].includes(hmRule.get());
  };
  hmRule.el.addEventListener("change", syncSpread);
  syncSpread();

  const runHarmony = async () => {
    try {
      const cols = await deps.genPalette({
        base: hmBase.get(),
        rule: hmRule.get(),
        n: hmN.get(),
        wheel: hmWheel.get(),
        spread: hmSpread.get(),
      });
      hmSwatches.update(cols);
    } catch (err) {
      console.error(err);
    }
  };
  for (const el of [hmBase.el, hmRule.el, hmN.el, hmWheel.el, hmSpread.el]) {
    el.addEventListener("input", () => void runHarmony());
    el.addEventListener("change", () => void runHarmony());
  }

  const heading = (text: string) => {
    const h = document.createElement("p");
    h.className = "control-label";
    h.style.fontWeight = "600";
    h.style.margin = "0.25rem 0 0";
    h.textContent = text;
    return h;
  };

  body.append(
    heading("Extrair de imagem"),
    lr("Cores", exCount.el),
    lr("Método", exMethod.el),
    lr("Espaço", exSpace.el),
    lr("Ordenar", exSort.el),
    exSeedRow,
    rowOf(exBtn),
    exStatus,
    exSwatches.element,
    heading("Gerar por harmonia"),
    lr("Cor base", hmBase.el),
    lr("Regra", hmRule.el),
    lr("Quantidade", hmN.el),
    lr("Roda de matiz", hmWheel.el),
    hmSpreadRow,
    hmSwatches.element,
  );

  root.addEventListener("toggle", () => {
    if (root.open && hmSwatches.element.childElementCount === 0) void runHarmony();
  });

  return { element: root };
}
