// Declarative descriptors for each effect's adjustable variables. Adding an
// effect is three touch points: a Go file in internal/filters/, one
// Register(...) in its init(), and one entry here. No per-effect DOM code —
// app/ParamControls.svelte turns any of these into a live form.

export interface Control {
  /** Param key sent to the filter (matches p.Int/Float/String/Bool in Go). */
  key: string;
  label: string;
  /** range/number/seed -> numeric input; select -> dropdown; checkbox -> bool; text -> free text. */
  type: "range" | "number" | "select" | "checkbox" | "seed" | "text";
  min?: number;
  max?: number;
  step?: number;
  default: number | string | boolean;
  /** Required for type "select". */
  options?: string[];
  /**
   * When set, the control is only shown (and only present in `values`) while
   * this returns true for the current values. Lets one descriptor carry
   * mode-specific params — e.g. dither's `levels` only under mode "levels".
   */
  showIf?: (values: Record<string, number | string | boolean>) => boolean;
}

export interface EffectUI {
  /** Registry name in Go. */
  name: string;
  /** Human label for the effect picker. */
  title: string;
  controls: Control[];
}

export const effects: EffectUI[] = [
  {
    name: "pixelate",
    title: "Pixelização",
    controls: [
      {
        key: "blockSize",
        label: "Tamanho do bloco",
        type: "range",
        min: 2,
        max: 64,
        step: 1,
        default: 8,
      },
    ],
  },
  {
    name: "sobel",
    title: "Detecção de bordas (Sobel)",
    controls: [
      {
        key: "threshold",
        label: "Limiar (0 = contínuo)",
        type: "range",
        min: 0,
        max: 255,
        step: 1,
        default: 0,
      },
      { key: "normalize", label: "Normalizar", type: "checkbox", default: true },
      { key: "invert", label: "Inverter", type: "checkbox", default: false },
    ],
  },
  {
    name: "quantize",
    title: "Quantização de cores (poster)",
    controls: [
      {
        key: "colors",
        label: "Número de cores",
        type: "range",
        min: 2,
        max: 64,
        step: 1,
        default: 8,
      },
      {
        key: "space",
        label: "Métrica de cor",
        type: "select",
        options: ["oklab", "srgb"],
        default: "oklab",
      },
    ],
  },
  {
    name: "dither",
    title: "Dithering (difusão de erro / ordenado)",
    controls: [
      {
        key: "mode",
        label: "Modo",
        type: "select",
        options: ["levels", "palette", "ordered"],
        default: "levels",
      },
      {
        key: "kernel",
        label: "Núcleo de difusão",
        type: "select",
        options: ["floyd-steinberg", "atkinson", "stucki", "jarvis", "sierra", "burkes"],
        default: "floyd-steinberg",
        showIf: (v) => v.mode === "levels" || v.mode === "palette",
      },
      {
        key: "matrix",
        label: "Matriz de Bayer",
        type: "select",
        options: ["2", "4", "8"],
        default: "4",
        showIf: (v) => v.mode === "ordered",
      },
      {
        key: "levels",
        label: "Níveis por canal",
        type: "range",
        min: 2,
        max: 16,
        step: 1,
        default: 2,
        showIf: (v) => v.mode === "levels" || v.mode === "ordered",
      },
      {
        key: "grayscale",
        label: "Escala de cinza",
        type: "checkbox",
        default: true,
        showIf: (v) => v.mode === "levels" || v.mode === "ordered",
      },
      {
        key: "colors",
        label: "Cores",
        type: "range",
        min: 2,
        max: 64,
        step: 1,
        default: 8,
        showIf: (v) => v.mode === "palette",
      },
      {
        key: "space",
        label: "Métrica de cor",
        type: "select",
        options: ["oklab", "srgb"],
        default: "oklab",
        showIf: (v) => v.mode === "palette",
      },
      { key: "serpentine", label: "Varredura serpentina", type: "checkbox", default: true },
    ],
  },
  {
    name: "halftone",
    title: "Meio-tom (Halftone)",
    controls: [
      { key: "cellSize", label: "Tamanho da célula (px)", type: "range", min: 2, max: 64, step: 1, default: 6 },
      { key: "angleDeg", label: "Ângulo da trama (°)", type: "range", min: 0, max: 90, step: 1, default: 45 },
      {
        key: "shape",
        label: "Forma do ponto",
        type: "select",
        options: ["circle", "square", "diamond", "line"],
        default: "circle",
      },
      {
        key: "channels",
        label: "Canais",
        type: "select",
        options: ["mono", "cmyk", "rgb"],
        default: "mono",
      },
      { key: "gamma", label: "Gama", type: "range", min: 0.2, max: 3, step: 0.05, default: 1 },
      { key: "invert", label: "Inverter", type: "checkbox", default: false },
      {
        key: "ink",
        label: "Tinta (hex)",
        type: "text",
        default: "#000000",
        showIf: (v) => v.channels === "mono",
      },
      {
        key: "paper",
        label: "Papel (hex)",
        type: "text",
        default: "#ffffff",
        showIf: (v) => v.channels === "mono",
      },
    ],
  },
  {
    name: "glitch",
    title: "Glitch / RGB Shift",
    controls: [
      { key: "seed", label: "Semente", type: "seed", default: 0 },
      {
        key: "rgbShift",
        label: "Deslocamento RGB (px)",
        type: "range",
        min: 0,
        max: 40,
        step: 1,
        default: 4,
      },
      {
        key: "sliceCount",
        label: "Número de faixas",
        type: "range",
        min: 0,
        max: 40,
        step: 1,
        default: 8,
      },
      {
        key: "maxSliceShift",
        label: "Deslocamento máx. da faixa (px)",
        type: "range",
        min: 0,
        max: 120,
        step: 1,
        default: 20,
      },
      {
        key: "sliceProbability",
        label: "Probabilidade de deslocar",
        type: "range",
        min: 0,
        max: 1,
        step: 0.05,
        default: 0.5,
      },
      { key: "scanlines", label: "Scanlines", type: "checkbox", default: false },
    ],
  },
  {
    name: "ascii",
    title: "ASCII colorido",
    controls: [
      {
        key: "cellWidth",
        label: "Largura da célula (px)",
        type: "range",
        min: 4,
        max: 24,
        step: 1,
        default: 8,
      },
      {
        key: "cellHeight",
        label: "Altura da célula (px)",
        type: "range",
        min: 6,
        max: 32,
        step: 1,
        default: 12,
      },
      {
        key: "ramp",
        label: "Rampa de caracteres (escuro → claro)",
        type: "text",
        default: " .:-=+*#%@",
      },
      { key: "colored", label: "Colorido", type: "checkbox", default: true },
      { key: "invert", label: "Inverter rampa", type: "checkbox", default: false },
      { key: "background", label: "Cor de fundo (hex)", type: "text", default: "#000000" },
    ],
  },
];
