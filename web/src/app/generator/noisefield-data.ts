// Color math, catalogue and presets for the generative noisefield gradient tool,
// ported and adapted from Gradient Studio (gurade.netlify.app).

export interface Preset {
  name: string;
  jp: string;
  genre: string;
  type: string;
  texture: string;
  direction: number;
  colors: string[];
}

export const GENRES: Array<[string, string]> = [
  ["duotone", "Duotone"],
  ["metallic", "Metálico"],
  ["chrome", "Cromado"],
  ["iridescent", "Irisado"],
  ["holographic", "Holográfico"],
  ["neon", "Neon"],
  ["pastel", "Pastel"],
  ["rainbow", "Arco-íris"],
];

export const TYPES: Array<[string, string]> = [
  ["flow", "Fluxo (Ruído)"],
  ["mesh", "Malha (Multi-ponto)"],
  ["freeform", "Livre"],
  ["linear", "Linear"],
  ["radial", "Radial"],
  ["conic", "Cônico"],
  ["reflected", "Refletido"],
  ["diamond", "Diamante"],
];

export const TYPE_NOTES: Record<string, string> = {
  flow: "Fluxo de cor distorcido por ruído simplex/fBm. Arraste os anéis sobre o canvas para posicionar as cores.",
  mesh: "Gradiente multiponto com difusão suave a partir de cada spot.",
  freeform: "Blocos de cor orgânicos com transições definidas.",
  linear: "Forma linear direcional clássica. Mapeia diretamente para CSS nativo.",
  radial: "Propaga-se do centro para as extremidades.",
  conic: "Gira em torno do eixo central.",
  reflected: "Simétrico, espelhando-se a partir do centro.",
  diamond: "Expande-se em formato de losango/diamante.",
};

export const TEXTURES: Array<[string, string]> = [
  ["smooth", "Lisa"],
  ["grain", "Granulada"],
  ["frosted", "Fosca"],
  ["wave", "Ondas"],
  ["wrinkle", "Amassada"],
  ["paper", "Papel"],
];

export const RATIOS: Array<[string, string]> = [
  ["16:9", "16/9"],
  ["1:1", "1/1"],
  ["4:5", "4/5"],
  ["3:2", "3/2"],
  ["9:16", "9/16"],
  ["21:9", "21/9"],
];

export const COMPASS_ANGLES = [315, 0, 45, 270, null, 90, 225, 180, 135] as const;

export const ORGANIC_TYPES = new Set(["flow", "mesh", "freeform"]);
export const ANGLED_TYPES = new Set(["linear", "reflected", "conic", "flow"]);

export const SPOT_DEFAULTS: Array<[number, number]> = [
  [0.2, 0.26],
  [0.8, 0.22],
  [0.72, 0.8],
  [0.24, 0.76],
  [0.5, 0.5],
  [0.5, 0.1],
  [0.92, 0.55],
  [0.08, 0.5],
];

export const PRESETS: Preset[] = [
  { name: "Nebula", jp: "Nebulosa", genre: "duotone", type: "flow", texture: "smooth", direction: 135, colors: ["#F3D3E4", "#4C5FD5", "#FF8FA3", "#8B6BAE"] },
  { name: "Sunset", jp: "Pôr do Sol", genre: "neon", type: "flow", texture: "smooth", direction: 135, colors: ["#FF5E62", "#FF9966", "#FFD194"] },
  { name: "Aurora", jp: "Aurora Boreal", genre: "neon", type: "flow", texture: "smooth", direction: 45, colors: ["#00F5A0", "#00D9F5", "#7B61FF", "#0B1B3A"] },
  { name: "Twilight", jp: "Crepúsculo", genre: "neon", type: "flow", texture: "smooth", direction: 135, colors: ["#4A00E0", "#8E2DE2", "#FF6FB5"] },
  { name: "Candy", jp: "Doce", genre: "pastel", type: "flow", texture: "smooth", direction: 135, colors: ["#FF0080", "#7928CA", "#00D4FF"] },
  { name: "Sea Glass", jp: "Vidro Marinho", genre: "duotone", type: "flow", texture: "frosted", direction: 135, colors: ["#2E3192", "#1BFFFF", "#C9FFF4"] },
  { name: "Rose Gold", jp: "Ouro Rosa", genre: "metallic", type: "flow", texture: "smooth", direction: 135, colors: ["#B76E79", "#F7CAC9", "#E8B4A0"] },
  { name: "Foil", jp: "Folha Metálica", genre: "chrome", type: "flow", texture: "wrinkle", direction: 90, colors: ["#8E9EAB", "#EEF2F3", "#5C6B78"] },
  { name: "Lavender", jp: "Lavanda", genre: "duotone", type: "mesh", texture: "smooth", direction: 135, colors: ["#A18CD1", "#FBC2EB", "#E4D9FF", "#7B61FF"] },
  { name: "Moss", jp: "Musgo", genre: "duotone", type: "mesh", texture: "grain", direction: 135, colors: ["#2F5D50", "#A3C9A8", "#E9F5DB", "#5A8F6E"] },
  { name: "Bloom", jp: "Floração", genre: "holographic", type: "freeform", texture: "smooth", direction: 90, colors: ["#FA8BFF", "#2BD2FF", "#2BFF88"] },
  { name: "Ember", jp: "Brasa", genre: "neon", type: "radial", texture: "grain", direction: 90, colors: ["#FF512F", "#F09819"] },
  { name: "Golden Hour", jp: "Hora Dourada", genre: "duotone", type: "linear", texture: "smooth", direction: 135, colors: ["#F6D365", "#FDA085"] },
  { name: "Coral", jp: "Coral", genre: "duotone", type: "linear", texture: "smooth", direction: 135, colors: ["#FF6E7F", "#BFE9FF"] },
  { name: "Dusk", jp: "Anoitecer", genre: "duotone", type: "diamond", texture: "smooth", direction: 90, colors: ["#FF5E62", "#FFD194"] },
  { name: "Gold Leaf", jp: "Folha de Ouro", genre: "metallic", type: "linear", texture: "smooth", direction: 135, colors: ["#BF953F", "#FCF6BA"] },
  { name: "Gold Wave", jp: "Onda Dourada", genre: "metallic", type: "linear", texture: "wave", direction: 90, colors: ["#BF953F", "#FCF6BA", "#B38728"] },
  { name: "Sand Dune", jp: "Duna de Areia", genre: "metallic", type: "linear", texture: "paper", direction: 135, colors: ["#C79081", "#DFA579"] },
  { name: "Liquid Metal", jp: "Metal Líquido", genre: "chrome", type: "reflected", texture: "smooth", direction: 135, colors: ["#333B44", "#C9D6DF"] },
  { name: "Ocean", jp: "Oceano", genre: "duotone", type: "linear", texture: "smooth", direction: 135, colors: ["#2E3192", "#1BFFFF"] },
  { name: "Deep Sea", jp: "Mar Profundo", genre: "duotone", type: "linear", texture: "grain", direction: 180, colors: ["#0F2027", "#2C5364"] },
  { name: "Emerald", jp: "Esmeralda", genre: "duotone", type: "linear", texture: "smooth", direction: 135, colors: ["#134E5E", "#71B280"] },
  { name: "Neon", jp: "Neon Roxo", genre: "neon", type: "linear", texture: "smooth", direction: 135, colors: ["#7F00FF", "#E100FF"] },
  { name: "Ink Wash", jp: "Nanquim", genre: "duotone", type: "radial", texture: "paper", direction: 90, colors: ["#1F1C2C", "#928DAB"] },
  { name: "Wine", jp: "Vinho", genre: "duotone", type: "linear", texture: "smooth", direction: 135, colors: ["#4B1248", "#A70000"] },
  { name: "Slate", jp: "Ardósia", genre: "duotone", type: "linear", texture: "smooth", direction: 135, colors: ["#485563", "#29323C"] },
  { name: "Shell", jp: "Concha", genre: "iridescent", type: "conic", texture: "smooth", direction: 90, colors: ["#EE9CA7", "#FFDDE1", "#A8EDEA"] },
  { name: "Prism", jp: "Prisma", genre: "rainbow", type: "conic", texture: "smooth", direction: 0, colors: ["#FF5F6D", "#FFC371", "#FFF240", "#66E46E", "#4FA7FF", "#B06AB3"] },
];

export const HEX_REGEX = /^#[0-9a-fA-F]{6}$/;

export function clamp(n: number, a: number, b: number): number {
  return Math.max(a, Math.min(b, n));
}

export function toRgb(hex: string): { r: number; g: number; b: number } {
  const h = hex.replace("#", "");
  return {
    r: parseInt(h.slice(0, 2), 16) || 0,
    g: parseInt(h.slice(2, 4), 16) || 0,
    b: parseInt(h.slice(4, 6), 16) || 0,
  };
}

export function toHex(r: number, g: number, b: number): string {
  const cr = clamp(Math.round(r), 0, 255).toString(16).padStart(2, "0");
  const cg = clamp(Math.round(g), 0, 255).toString(16).padStart(2, "0");
  const cb = clamp(Math.round(b), 0, 255).toString(16).padStart(2, "0");
  return `#${cr}${cg}${cb}`.toUpperCase();
}

export function mix(a: string, b: string, t: number): string {
  const x = toRgb(a);
  const y = toRgb(b);
  return toHex(x.r + (y.r - x.r) * t, x.g + (y.g - x.g) * t, x.b + (y.b - x.b) * t);
}

export function lighten(hex: string, t: number): string {
  return mix(hex, "#FFFFFF", t);
}

export function darken(hex: string, t: number): string {
  return mix(hex, "#000000", t);
}

export function luma(hex: string): number {
  const c = toRgb(hex);
  return (c.r * 0.299 + c.g * 0.587 + c.b * 0.114) / 255;
}

export function toHsl(hex: string): { h: number; s: number; l: number } {
  const c = toRgb(hex);
  const r = c.r / 255;
  const g = c.g / 255;
  const b = c.b / 255;
  const mx = Math.max(r, g, b);
  const mn = Math.min(r, g, b);
  const l = (mx + mn) / 2;
  let s = 0;
  let h = 0;
  const d = mx - mn;
  if (d > 0) {
    s = l > 0.5 ? d / (2 - mx - mn) : d / (mx + mn);
    if (mx === r) h = (g - b) / d + (g < b ? 6 : 0);
    else if (mx === g) h = (b - r) / d + 2;
    else h = (r - g) / d + 4;
    h *= 60;
  }
  return { h, s, l };
}

export function hslToHex(h: number, s: number, l: number): string {
  const hNorm = ((h % 360) + 360) % 360;
  const c = (1 - Math.abs(2 * l - 1)) * s;
  const x = c * (1 - Math.abs(((hNorm / 60) % 2) - 1));
  const m = l - c / 2;
  let r = 0;
  let g = 0;
  let b = 0;
  if (hNorm < 60) {
    r = c; g = x;
  } else if (hNorm < 120) {
    r = x; g = c;
  } else if (hNorm < 180) {
    g = c; b = x;
  } else if (hNorm < 240) {
    g = x; b = c;
  } else if (hNorm < 300) {
    r = x; b = c;
  } else {
    r = c; b = x;
  }
  return toHex((r + m) * 255, (g + m) * 255, (b + m) * 255);
}

const HUE_NAMES: Array<[string, number]> = [
  ["VERMELHO", 0],
  ["CORAL", 14],
  ["LARANJA", 28],
  ["ÂMBAR", 40],
  ["DOURADO", 50],
  ["LIMA", 72],
  ["VERDE", 115],
  ["MENTA", 150],
  ["VERDE-ÁGUA", 170],
  ["CIANO", 186],
  ["AZUL CÉU", 200],
  ["AZUL", 214],
  ["AZUL ESCURO", 232],
  ["ÍNDIGO", 254],
  ["VIOLETA", 270],
  ["PÚRPURA", 286],
  ["MAGENTA", 306],
  ["ROSA", 330],
  ["ROSE", 346],
  ["VERMELHO", 360],
];

export function colorName(hex: string): string {
  const c = toHsl(hex);
  if (c.s < 0.1 || c.l > 0.96 || c.l < 0.05) {
    return c.l > 0.92
      ? "BRANCO"
      : c.l > 0.7
        ? "PRATA"
        : c.l > 0.45
          ? "CINZA"
          : c.l > 0.2
            ? "GRAFITE"
            : "PRETO";
  }
  let best = HUE_NAMES[0];
  let bd = 999;
  for (const n of HUE_NAMES) {
    const d = Math.abs(n[1] - c.h);
    if (d < bd) {
      bd = d;
      best = n;
    }
  }
  const pre =
    c.l > 0.84
      ? "PÁLIDO "
      : c.l > 0.68
        ? "CLARO "
        : c.l < 0.22
          ? "MUITO ESCURO "
          : c.l < 0.38
            ? "ESCURO "
            : "";
  return `${pre}${best[0]}`;
}

export function sanitizeColors(list: string[]): string[] {
  const out = list
    .filter((v) => HEX_REGEX.test(String(v).trim()))
    .map((v) => v.trim().toUpperCase());
  return out.length ? out : ["#888888", "#CCCCCC"];
}

export function stopsFrom(genre: string, list: string[]): string[] {
  const c = sanitizeColors(list);
  const a = c[0];
  const b = c[1] || c[0];
  const d = c[2] || c[1] || c[0];
  switch (genre.toLowerCase()) {
    case "metallic":
      return [darken(a, 0.55), a, lighten(a, 0.55), a, darken(b, 0.35), lighten(b, 0.35), b];
    case "chrome":
      return [
        darken(a, 0.78),
        lighten(a, 0.88),
        darken(a, 0.18),
        "#FFFFFF",
        darken(b, 0.65),
        lighten(b, 0.82),
        darken(b, 0.22),
      ];
    case "iridescent":
      return [a, lighten(b, 0.22), d, lighten(a, 0.28), b, a];
    case "holographic":
      return [lighten(a, 0.18), b, lighten(d, 0.12), a, lighten(b, 0.2), d];
    case "neon":
      return [darken(a, 0.16), a, lighten(a, 0.24), b, d, lighten(d, 0.22)];
    case "pastel":
      return c.map((x) => lighten(x, 0.48));
    case "duotone":
      return [a, b];
    case "rainbow":
      return c.length >= 5 ? c : [a, b, d, "#FFF240", "#66E46E", "#4FA7FF"];
    default:
      return c;
  }
}

export function geoStops(type: string, genre: string, colors: string[]): string[] {
  const s = stopsFrom(genre, colors);
  return type.toLowerCase() === "reflected"
    ? s.slice().reverse().concat(s.slice(1))
    : s;
}

export function defaultSpots(n: number): Array<[number, number]> {
  return SPOT_DEFAULTS.slice(0, n).map((p) => [p[0], p[1]]);
}

export function calculateOutputDimensions(ratioStr: string, maxDim = 1600): { w: number; h: number } {
  const parts = ratioStr.split("/").map(Number);
  const rw = parts[0] || 16;
  const rh = parts[1] || 9;
  if (rw >= rh) {
    return { w: maxDim, h: Math.round((maxDim * rh) / rw) };
  }
  return { w: Math.round((maxDim * rw) / rh), h: maxDim };
}

export function shuffleColorsAndSpots(currentCount: number): {
  colors: string[];
  spots: Array<[number, number]>;
  seed: number;
} {
  const n = clamp(currentCount, 2, 8);
  const baseHue = Math.random() * 360;
  const schemes = [
    [0, 25, 50, 75],
    [0, 180, 20, 200],
    [0, 120, 240, 60],
    [0, 150, 210, 30],
    [0, 15, -15, 30],
    [0, 40, 200, 220],
  ];
  const sch = schemes[Math.floor(Math.random() * schemes.length)];
  const cols: string[] = [];
  for (let i = 0; i < n; i++) {
    const h = baseHue + sch[i % sch.length] + (i >= sch.length ? 23 * i : 0);
    const s = 0.5 + Math.random() * 0.4;
    const l = i % 2 ? 0.66 + Math.random() * 0.22 : 0.36 + Math.random() * 0.24;
    cols.push(hslToHex(h, s, l));
  }
  const spots = defaultSpots(n).map(([x, y]) => [
    clamp(x + (Math.random() - 0.5) * 0.28, 0.06, 0.94),
    clamp(y + (Math.random() - 0.5) * 0.28, 0.06, 0.94),
  ] as [number, number]);

  return {
    colors: cols,
    spots,
    seed: Math.floor(Math.random() * 100000),
  };
}

export function generateNoiseCSS(params: {
  field: string;
  style: string;
  texture: string;
  direction: number;
  colors: string[];
  spots: Array<[number, number]>;
  ratio: string;
  scale: number;
  distortion: number;
}): string {
  const ar = params.ratio.replace("/", " / ");
  const t = params.field.toLowerCase();
  const isOrganic = ORGANIC_TYPES.has(t);

  if (isOrganic) {
    const validCols = sanitizeColors(params.colors);
    const cols = validCols.map((c) => (params.style === "pastel" ? lighten(c, 0.42) : c));
    const base = cols.slice().sort((a, b) => luma(a) - luma(b))[0];
    const reach = t === "freeform" ? "48%" : "62%";
    const layers = params.spots.map((sp, i) => {
      const col = cols[i % cols.length];
      const px = Math.round(sp[0] * 100);
      const py = Math.round(sp[1] * 100);
      return `radial-gradient(at ${px}% ${py}%, ${col} 0px, transparent ${reach})`;
    });

    let out = `/* ${params.field.toUpperCase()} é gerado por fragment shader WebGL determinístico. */\n`;
    out += `/* O código abaixo é uma aproximação CSS pura em camadas de radial-gradient: */\n\n`;
    out += `.gradient {\n`;
    out += `  background-color: ${base};\n`;
    out += `  background-image:\n    ${layers.join(",\n    ")};\n`;
    out += `  aspect-ratio: ${ar};\n`;
    out += `}\n\n`;
    out += `/* Estilo: ${params.style} | Textura: ${params.texture} | Escala: ${params.scale}% | Distorção: ${params.distortion}% */\n`;
    out += `/* Paleta original: ${params.colors.join(", ")} */\n`;
    return out;
  }

  const stops = geoStops(t, params.style, params.colors);
  const fmt = stops
    .map((v, i) => `${v} ${Math.round((i / Math.max(stops.length - 1, 1)) * 100)}%`)
    .join(", ");
  let bg = "";
  if (t === "linear" || t === "reflected") {
    bg = `linear-gradient(${params.direction}deg, ${fmt})`;
  } else if (t === "radial") {
    bg = `radial-gradient(circle at 50% 50%, ${fmt})`;
  } else if (t === "conic") {
    bg = `conic-gradient(from ${params.direction}deg at 50% 50%, ${fmt})`;
  } else if (t === "diamond") {
    bg = `radial-gradient(circle at 50% 50%, ${fmt}) /* Diamante aproximado por radial */`;
  }

  let out = `.gradient {\n`;
  out += `  background: ${bg};\n`;
  out += `  aspect-ratio: ${ar};\n`;
  out += `}\n\n`;
  out += `/* Estilo: ${params.style} | Textura: ${params.texture} */\n`;
  out += `/* Paleta: ${stops.join(", ")} */\n`;
  return out;
}

export function render2DPreview(
  ctx: CanvasRenderingContext2D,
  w: number,
  h: number,
  type: string,
  style: string,
  direction: number,
  colors: string[],
  spots: Array<[number, number]>,
): void {
  const rad = ((direction || 0) * Math.PI) / 180;
  const dx = Math.sin(rad);
  const dy = -Math.cos(rad);
  const cx = w / 2;
  const cy = h / 2;
  const t = type.toLowerCase();

  if (!ORGANIC_TYPES.has(t)) {
    const stops = geoStops(t, style, colors);
    let g: CanvasGradient;
    if (t === "radial" || t === "diamond") {
      g = ctx.createRadialGradient(cx, cy, 0, cx, cy, Math.hypot(w, h) / 2);
    } else if (t === "conic" && "createConicGradient" in ctx) {
      g = (ctx as any).createConicGradient(rad - Math.PI / 2, cx, cy);
    } else {
      const len = (Math.abs(dx) * w) / 2 + (Math.abs(dy) * h) / 2;
      g = ctx.createLinearGradient(cx - dx * len, cy - dy * len, cx + dx * len, cy + dy * len);
    }
    stops.forEach((s, i) => {
      g.addColorStop(i / Math.max(stops.length - 1, 1), s);
    });
    ctx.fillStyle = g;
    ctx.fillRect(0, 0, w, h);
    return;
  }

  const validCols = sanitizeColors(colors);
  ctx.fillStyle = darken(validCols[0], 0.25);
  ctx.fillRect(0, 0, w, h);

  spots.forEach((sp, i) => {
    const hex = validCols[i % validCols.length];
    const rgb = toRgb(hex);
    const sx = w * sp[0];
    const sy = h * sp[1];
    const gr = ctx.createRadialGradient(sx, sy, 0, sx, sy, Math.max(w, h) * 0.45);
    gr.addColorStop(0, `rgba(${rgb.r},${rgb.g},${rgb.b},0.95)`);
    gr.addColorStop(1, `rgba(${rgb.r},${rgb.g},${rgb.b},0)`);
    ctx.fillStyle = gr;
    ctx.fillRect(0, 0, w, h);
  });
}
