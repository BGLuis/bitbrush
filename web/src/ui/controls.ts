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
    name: "grayscale",
    title: "Preto e branco",
    controls: [
      {
        key: "method",
        label: "Ponderação do tom",
        type: "select",
        options: ["luma", "luminance", "average", "bt601", "lightness"],
        default: "luma",
      },
      { key: "brightness", label: "Brilho", type: "range", min: -100, max: 100, step: 1, default: 0 },
      { key: "contrast", label: "Contraste", type: "range", min: -100, max: 100, step: 1, default: 0 },
      {
        key: "threshold",
        label: "Limiar 1-bit (0 = contínuo)",
        type: "range",
        min: 0,
        max: 255,
        step: 1,
        default: 0,
      },
      { key: "invert", label: "Inverter", type: "checkbox", default: false },
    ],
  },
  {
    name: "paper",
    title: "Simulação de papel",
    controls: [
      { key: "seed", label: "Semente", type: "seed", default: 0 },
      { key: "paper", label: "Cor do papel (hex)", type: "text", default: "#f3ecd8" },
      { key: "age", label: "Envelhecimento (sépia)", type: "range", min: 0, max: 1, step: 0.05, default: 0.5 },
      { key: "grain", label: "Grão das fibras", type: "range", min: 0, max: 1, step: 0.05, default: 0.35 },
      { key: "mottle", label: "Manchas amplas", type: "range", min: 0, max: 1, step: 0.05, default: 0.3 },
      { key: "fibers", label: "Estrias verticais", type: "range", min: 0, max: 1, step: 0.05, default: 0.2 },
      { key: "scale", label: "Escala da textura", type: "range", min: 0.5, max: 8, step: 0.1, default: 2 },
      { key: "vignette", label: "Vinheta", type: "range", min: 0, max: 1, step: 0.05, default: 0.25 },
    ],
  },
  {
    name: "pencil",
    title: "Desenho a lápis preto",
    controls: [
      { key: "blur", label: "Raio do traço", type: "range", min: 1, max: 40, step: 1, default: 8 },
      { key: "strength", label: "Intensidade do esboço", type: "range", min: 0, max: 1, step: 0.05, default: 1 },
      { key: "darkness", label: "Peso do grafite", type: "range", min: 0.3, max: 4, step: 0.05, default: 1 },
      { key: "hatch", label: "Hachura nas sombras", type: "range", min: 0, max: 1, step: 0.05, default: 0.3 },
      { key: "seed", label: "Semente da hachura", type: "seed", default: 0 },
      { key: "graphite", label: "Cor do grafite (hex)", type: "text", default: "#1b1b1b" },
      { key: "paper", label: "Cor do papel (hex)", type: "text", default: "#f6f3ea" },
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
        options: ["floyd-steinberg", "atkinson", "stucki", "jarvis", "sierra", "sierra-lite", "burkes"],
        default: "floyd-steinberg",
        showIf: (v) => v.mode === "levels" || v.mode === "palette",
      },
      {
        key: "pattern",
        label: "Padrão ordenado",
        type: "select",
        options: [
          "bayer",
          "cluster",
          "radial",
          "lines-h",
          "lines-v",
          "lines-d",
          "white-noise",
          "blue-noise",
        ],
        default: "bayer",
        showIf: (v) => v.mode === "ordered",
      },
      {
        key: "matrix",
        label: "Matriz de Bayer",
        type: "select",
        options: ["2", "4", "8", "16"],
        default: "4",
        showIf: (v) => v.mode === "ordered" && (v.pattern === "bayer" || v.pattern === undefined),
      },
      {
        key: "seed",
        label: "Semente (ruído branco)",
        type: "seed",
        default: 0,
        showIf: (v) => v.mode === "ordered" && v.pattern === "white-noise",
      },
      {
        key: "palette",
        label: "Paleta retrô",
        type: "select",
        options: [
          "custom",
          "gameboy",
          "cga0",
          "cga1",
          "pico8",
          "c64",
          "nes",
          "grey2",
          "grey3",
          "rgb3",
          "sepia",
        ],
        default: "custom",
        showIf: (v) => v.mode === "palette",
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
        showIf: (v) => v.mode === "palette" && (v.palette === "custom" || v.palette === undefined),
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
    name: "stipple",
    title: "Pontilhado Voronoi (stippling)",
    controls: [
      { key: "points", label: "Número de pontos", type: "range", min: 100, max: 20000, step: 100, default: 4000 },
      { key: "iterations", label: "Iterações (Lloyd)", type: "range", min: 0, max: 80, step: 1, default: 30 },
      { key: "seed", label: "Semente", type: "seed", default: 0 },
      { key: "minRadius", label: "Raio mín.", type: "range", min: 0.2, max: 4, step: 0.1, default: 0.6 },
      { key: "maxRadius", label: "Raio máx.", type: "range", min: 0.5, max: 8, step: 0.1, default: 2.4 },
      { key: "gamma", label: "Gama da densidade", type: "range", min: 0.2, max: 3, step: 0.05, default: 1 },
      { key: "invert", label: "Inverter densidade", type: "checkbox", default: false },
      { key: "ink", label: "Tinta (hex)", type: "text", default: "#111111" },
      { key: "paper", label: "Papel (hex)", type: "text", default: "#f5f2ea" },
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
  {
    name: "blocks",
    title: "Caracteres de bloco",
    controls: [
      { key: "cellWidth", label: "Largura da célula (px)", type: "range", min: 2, max: 24, step: 1, default: 6 },
      { key: "cellHeight", label: "Altura da célula (px)", type: "range", min: 2, max: 24, step: 1, default: 6 },
      {
        key: "mode",
        label: "Modo",
        type: "select",
        options: ["quadrant", "halves", "shade"],
        default: "quadrant",
      },
      { key: "colored", label: "Colorido", type: "checkbox", default: true },
      { key: "invert", label: "Inverter", type: "checkbox", default: false },
      { key: "background", label: "Cor de fundo (hex)", type: "text", default: "#000000" },
    ],
  },
  {
    name: "glyphscreen",
    title: "Trama de glifos (cruz / diagonal / losango / linhas)",
    controls: [
      { key: "cell", label: "Tamanho da célula (px)", type: "range", min: 3, max: 32, step: 1, default: 10 },
      {
        key: "shape",
        label: "Forma",
        type: "select",
        options: ["cross", "diagonal", "backslash", "vertical", "horizontal", "plus", "diamond", "dot"],
        default: "cross",
      },
      { key: "weight", label: "Espessura do traço (px)", type: "range", min: 0.5, max: 6, step: 0.5, default: 2 },
      { key: "colored", label: "Colorido", type: "checkbox", default: true },
      { key: "invert", label: "Inverter", type: "checkbox", default: false },
      { key: "background", label: "Cor de fundo (hex)", type: "text", default: "#000000" },
    ],
  },
  {
    name: "mosaic",
    title: "Mosaico de fotos",
    controls: [
      { key: "cellWidth", label: "Largura do ladrilho (px)", type: "range", min: 2, max: 64, step: 1, default: 16 },
      { key: "cellHeight", label: "Altura do ladrilho (px)", type: "range", min: 2, max: 64, step: 1, default: 16 },
      { key: "gap", label: "Rejunte (px)", type: "range", min: 0, max: 8, step: 1, default: 0 },
      { key: "gapColor", label: "Cor do rejunte (hex)", type: "text", default: "#101010" },
      {
        key: "palette",
        label: "Paleta retrô",
        type: "select",
        options: ["none", "gameboy", "cga0", "cga1", "pico8", "c64", "nes", "grey2", "grey3", "rgb3", "sepia"],
        default: "none",
      },
      { key: "bayerOverlay", label: "Textura de Bayer", type: "checkbox", default: false },
    ],
  },
  {
    name: "pixelart",
    title: "Pixel art (paleta / bits)",
    controls: [
      { key: "cell", label: "Tamanho da célula (px)", type: "range", min: 1, max: 48, step: 1, default: 8 },
      {
        key: "palette",
        label: "Paleta retrô",
        type: "select",
        options: ["none", "gameboy", "cga0", "cga1", "pico8", "c64", "nes", "grey2", "grey3", "rgb3", "sepia"],
        default: "none",
      },
      {
        key: "bits",
        label: "Bits por canal",
        type: "range",
        min: 1,
        max: 8,
        step: 1,
        default: 8,
        showIf: (v) => v.palette === "none" || v.palette === undefined,
      },
      { key: "outline", label: "Contorno entre blocos", type: "checkbox", default: false },
    ],
  },
  {
    name: "lego",
    title: "Blocos de montar",
    controls: [
      { key: "cell", label: "Tamanho da peça (px)", type: "range", min: 4, max: 48, step: 1, default: 16 },
      { key: "colors", label: "Cores (poster)", type: "range", min: 2, max: 64, step: 1, default: 16 },
      { key: "studContrast", label: "Relevo do pino", type: "range", min: 0, max: 1, step: 0.05, default: 0.35 },
      { key: "outline", label: "Contorno da peça", type: "checkbox", default: true },
    ],
  },
  {
    name: "voxel",
    title: "Cubos voxel",
    controls: [
      { key: "cell", label: "Tamanho da célula (px)", type: "range", min: 4, max: 64, step: 1, default: 20 },
      { key: "topShade", label: "Face superior", type: "range", min: 0, max: 2, step: 0.05, default: 1.0 },
      { key: "leftShade", label: "Face esquerda", type: "range", min: 0, max: 2, step: 0.05, default: 0.75 },
      { key: "rightShade", label: "Face direita", type: "range", min: 0, max: 2, step: 0.05, default: 0.55 },
      { key: "outline", label: "Contorno (px)", type: "range", min: 0, max: 4, step: 0.5, default: 1 },
      { key: "outlineColor", label: "Cor do contorno (hex)", type: "text", default: "#0a0a0a" },
      { key: "heightFromLuma", label: "Altura pela luminância", type: "checkbox", default: false },
      { key: "lightFlip", label: "Inverter luz", type: "checkbox", default: false },
      { key: "background", label: "Cor de fundo (hex)", type: "text", default: "#000000" },
    ],
  },
  {
    name: "vignette",
    title: "Vinheta",
    controls: [
      { key: "strength", label: "Intensidade", type: "range", min: 0, max: 1, step: 0.05, default: 0.6 },
      { key: "inner", label: "Raio interno", type: "range", min: 0, max: 2, step: 0.05, default: 0.4 },
      { key: "outer", label: "Raio externo", type: "range", min: 0, max: 2, step: 0.05, default: 1.0 },
      { key: "roundness", label: "Formato", type: "range", min: 0.1, max: 4, step: 0.05, default: 1.0 },
      { key: "color", label: "Cor da borda (hex)", type: "text", default: "#000000" },
    ],
  },
  {
    name: "scanlines",
    title: "Scanlines",
    controls: [
      { key: "spacing", label: "Espaçamento (px)", type: "range", min: 2, max: 20, step: 1, default: 3 },
      { key: "thickness", label: "Espessura (px)", type: "range", min: 1, max: 10, step: 1, default: 1 },
      { key: "darkness", label: "Escurecimento", type: "range", min: 0, max: 1, step: 0.05, default: 0.5 },
      { key: "opacity", label: "Opacidade", type: "range", min: 0, max: 1, step: 0.05, default: 1.0 },
    ],
  },
  {
    name: "crt",
    title: "Tela CRT",
    controls: [
      { key: "curvature", label: "Curvatura", type: "range", min: 0, max: 0.5, step: 0.01, default: 0.15 },
      { key: "zoom", label: "Zoom", type: "range", min: 0.5, max: 1.5, step: 0.01, default: 1.0 },
      { key: "scanline", label: "Scanlines", type: "range", min: 0, max: 1, step: 0.05, default: 0.4 },
      { key: "mask", label: "Máscara de fósforo", type: "range", min: 0, max: 1, step: 0.05, default: 0.2 },
      { key: "vignette", label: "Vinheta", type: "range", min: 0, max: 1, step: 0.05, default: 0.3 },
    ],
  },
  {
    name: "chromatic",
    title: "Aberração cromática",
    controls: [
      { key: "strength", label: "Deslocamento (px)", type: "range", min: 0, max: 20, step: 0.5, default: 3 },
      { key: "radial", label: "Radial (cresce nas bordas)", type: "checkbox", default: true },
    ],
  },
  {
    name: "blur",
    title: "Desfoque",
    controls: [
      { key: "radius", label: "Raio (px)", type: "range", min: 0, max: 40, step: 1, default: 4 },
      { key: "type", label: "Tipo", type: "select", options: ["gaussian", "box"], default: "gaussian" },
    ],
  },
  {
    name: "bloom",
    title: "Bloom",
    controls: [
      { key: "threshold", label: "Limiar de brilho", type: "range", min: 0, max: 1, step: 0.05, default: 0.75 },
      { key: "radius", label: "Raio do halo (px)", type: "range", min: 0, max: 40, step: 1, default: 12 },
      { key: "intensity", label: "Intensidade", type: "range", min: 0, max: 2, step: 0.05, default: 0.8 },
    ],
  },
  {
    name: "grain",
    title: "Grão de filme",
    controls: [
      { key: "amount", label: "Quantidade", type: "range", min: 0, max: 1, step: 0.02, default: 0.12 },
      { key: "size", label: "Tamanho do grão (px)", type: "range", min: 1, max: 8, step: 1, default: 1 },
      { key: "seed", label: "Semente", type: "seed", default: 1 },
      { key: "mono", label: "Monocromático", type: "checkbox", default: true },
    ],
  },
  {
    name: "dust",
    title: "Poeira e riscos de filme",
    controls: [
      { key: "density", label: "Densidade", type: "range", min: 0, max: 0.02, step: 0.0005, default: 0.002 },
      { key: "seed", label: "Semente", type: "seed", default: 1 },
      { key: "scratches", label: "Riscos verticais", type: "checkbox", default: true },
    ],
  },
  {
    name: "coloroverlay",
    title: "Sobreposição de cor",
    controls: [
      { key: "color", label: "Cor (hex)", type: "text", default: "#ff8800" },
      {
        key: "mode",
        label: "Modo de mesclagem",
        type: "select",
        options: ["normal", "multiply", "screen", "overlay", "color"],
        default: "normal",
      },
      { key: "opacity", label: "Opacidade", type: "range", min: 0, max: 1, step: 0.05, default: 0.3 },
    ],
  },
  {
    name: "chroma-key",
    title: "Chroma Key (remover cor)",
    controls: [
      { key: "color", label: "Cor alvo (hex)", type: "text", default: "#00ff00" },
      { key: "tolerance", label: "Tolerância", type: "range", min: 0, max: 1, step: 0.01, default: 0.3 },
      { key: "feather", label: "Suavização de borda", type: "range", min: 0, max: 0.5, step: 0.01, default: 0.05 },
      { key: "invert", label: "Inverter (manter a cor)", type: "checkbox", default: false },
    ],
  },
  {
    name: "luma-key",
    title: "Luma Key (remover por luminosidade)",
    controls: [
      { key: "threshold", label: "Limiar de luminância", type: "range", min: 0, max: 1, step: 0.01, default: 0.1 },
      { key: "feather", label: "Suavização de borda", type: "range", min: 0, max: 0.5, step: 0.01, default: 0.05 },
      { key: "invert", label: "Inverter (remover brilho)", type: "checkbox", default: false },
    ],
  },
];
