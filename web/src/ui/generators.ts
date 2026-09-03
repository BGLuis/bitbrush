// Declarative descriptors for the algorithmic generators (the Go
// internal/generators registry). Same shape and philosophy as
// ./controls.ts: adding a generator is three touch points — a Go package
// under internal/, one generators.Register(...) in its init(), and one
// entry here — with no bespoke DOM code.
//
// Reuses the Control type from ./controls.ts. A generator produces a whole
// image from parameters alone (no input image), so it also carries a
// default output size.

import type { Control } from "./controls";

export interface GeneratorUI {
  /** Registry name in Go (internal/generators). */
  name: string;
  /** Human label for the generator picker. */
  title: string;
  /** Canvas size to use when this generator is first selected. */
  defaultSize: { width: number; height: number };
  controls: Control[];
}

export const generators: GeneratorUI[] = [
  {
    name: "truchet",
    title: "Ladrilhos de Truchet",
    defaultSize: { width: 1024, height: 1024 },
    controls: [
      { key: "seed", label: "Semente", type: "seed", default: 0 },
      { key: "tiles", label: "Ladrilhos na largura", type: "range", min: 2, max: 64, step: 1, default: 12 },
      {
        key: "style",
        label: "Estilo",
        type: "select",
        options: ["arcs", "lines", "maze", "triangles"],
        default: "arcs",
      },
      { key: "lineWidth", label: "Espessura do traço", type: "range", min: 0.03, max: 0.5, step: 0.01, default: 0.18 },
      { key: "multiScale", label: "Multi-escala", type: "checkbox", default: false },
      { key: "colorful", label: "Colorido", type: "checkbox", default: false },
      { key: "background", label: "Fundo (hex)", type: "text", default: "#141414" },
      { key: "ink", label: "Traço (hex)", type: "text", default: "#f2efe6" },
    ],
  },
  {
    name: "harmonograph",
    title: "Harmonógrafo",
    defaultSize: { width: 1024, height: 1024 },
    controls: [
      { key: "seed", label: "Semente", type: "seed", default: 0 },
      { key: "pendulums", label: "Pêndulos por eixo", type: "range", min: 1, max: 4, step: 1, default: 2 },
      { key: "damping", label: "Amortecimento", type: "range", min: 0, max: 0.05, step: 0.001, default: 0.006 },
      { key: "freqSpread", label: "Desafinação", type: "range", min: 0, max: 0.2, step: 0.002, default: 0.012 },
      { key: "duration", label: "Duração", type: "range", min: 20, max: 2000, step: 10, default: 220 },
      { key: "steps", label: "Amostras", type: "range", min: 2000, max: 400000, step: 1000, default: 60000 },
      { key: "lineAlpha", label: "Opacidade do traço", type: "range", min: 0.005, max: 1, step: 0.005, default: 0.06 },
      { key: "colorful", label: "Colorido", type: "checkbox", default: false },
      { key: "time", label: "Tempo (fase)", type: "range", min: 0, max: 60, step: 0.1, default: 0 },
      { key: "background", label: "Fundo (hex)", type: "text", default: "#0e0e12" },
      { key: "ink", label: "Traço (hex)", type: "text", default: "#e9e6dc" },
    ],
  },
  {
    name: "attractor",
    title: "Atrator estranho",
    defaultSize: { width: 1024, height: 1024 },
    controls: [
      {
        key: "type",
        label: "Mapa",
        type: "select",
        options: ["clifford", "dejong", "svensson"],
        default: "clifford",
      },
      { key: "seed", label: "Semente (coefs. se a=b=c=d=0)", type: "seed", default: 0 },
      { key: "a", label: "a", type: "number", step: 0.001, default: 0 },
      { key: "b", label: "b", type: "number", step: 0.001, default: 0 },
      { key: "c", label: "c", type: "number", step: 0.001, default: 0 },
      { key: "d", label: "d", type: "number", step: 0.001, default: 0 },
      { key: "iterations", label: "Iterações", type: "range", min: 50000, max: 20000000, step: 50000, default: 2000000 },
      { key: "gamma", label: "Gama (tone map)", type: "range", min: 0.5, max: 6, step: 0.1, default: 2.2 },
      { key: "zoom", label: "Zoom", type: "range", min: 0.2, max: 5, step: 0.05, default: 1 },
      { key: "colorBySpeed", label: "Cor pela velocidade", type: "checkbox", default: false },
      { key: "background", label: "Fundo (hex)", type: "text", default: "#07070b" },
      { key: "ink", label: "Tinta (hex)", type: "text", default: "#efe7d8" },
    ],
  },
  {
    name: "contours",
    title: "Curvas de nível (topográfico)",
    defaultSize: { width: 1280, height: 800 },
    controls: [
      { key: "seed", label: "Semente", type: "seed", default: 1337 },
      { key: "levels", label: "Níveis", type: "range", min: 4, max: 48, step: 1, default: 18 },
      { key: "indexEvery", label: "Curva-mestra a cada", type: "range", min: 0, max: 12, step: 1, default: 5 },
      { key: "scale", label: "Escala do ruído", type: "range", min: 0.2, max: 4, step: 0.05, default: 1 },
      { key: "warp", label: "Distorção de domínio", type: "range", min: 0, max: 2, step: 0.05, default: 0.6 },
      { key: "octaves", label: "Oitavas (fBm)", type: "range", min: 1, max: 8, step: 1, default: 5 },
      { key: "time", label: "Tempo (anima)", type: "range", min: 0, max: 60, step: 0.1, default: 0 },
      { key: "hillshade", label: "Relevo sombreado", type: "checkbox", default: true },
      {
        key: "palette",
        label: "Paleta",
        type: "select",
        options: ["paper", "blueprint", "terrain"],
        default: "paper",
      },
      { key: "background", label: "Fundo (hex, opcional)", type: "text", default: "" },
    ],
  },
  {
    name: "flowfield",
    title: "Campo de fluxo (tinta sumi)",
    defaultSize: { width: 1280, height: 800 },
    controls: [
      { key: "seed", label: "Semente", type: "seed", default: 0 },
      { key: "turbulence", label: "Turbulência", type: "range", min: 0.2, max: 8, step: 0.1, default: 2.4 },
      { key: "density", label: "Densidade de traços", type: "range", min: 0.15, max: 4, step: 0.05, default: 1 },
      { key: "curl", label: "Campo curl (sem divergência)", type: "checkbox", default: false },
      {
        key: "palette",
        label: "Paleta",
        type: "select",
        options: ["ink", "indigo", "vermilion"],
        default: "ink",
      },
      { key: "grain", label: "Grão do papel", type: "range", min: 0, max: 1, step: 0.05, default: 0.4 },
      { key: "time", label: "Tempo (fluxo)", type: "range", min: 0, max: 60, step: 0.1, default: 0 },
      { key: "background", label: "Papel (hex, opcional)", type: "text", default: "" },
    ],
  },
  {
    name: "lsystem",
    title: "Sistema-L (turtle graphics)",
    defaultSize: { width: 1024, height: 1024 },
    controls: [
      {
        key: "preset",
        label: "Preset",
        type: "select",
        options: ["plant", "koch", "dragon", "sierpinski", "tree", "custom"],
        default: "plant",
      },
      { key: "axiom", label: "Axioma (custom)", type: "text", default: "X", showIf: (v) => v.preset === "custom" },
      {
        key: "rules",
        label: "Regras (custom)",
        type: "text",
        default: "X=F+[[X]-X]-F[-FX]+X;F=FF",
        showIf: (v) => v.preset === "custom",
      },
      { key: "angleDeg", label: "Ângulo (°)", type: "range", min: 0, max: 180, step: 1, default: 25 },
      { key: "iterations", label: "Iterações", type: "range", min: 0, max: 18, step: 1, default: 5 },
      { key: "seed", label: "Semente", type: "seed", default: 0 },
      { key: "jitter", label: "Tremor aleatório", type: "range", min: 0, max: 1, step: 0.02, default: 0 },
      { key: "lineWidth", label: "Espessura", type: "range", min: 0.5, max: 6, step: 0.1, default: 1.4 },
      { key: "colorByDepth", label: "Cor por profundidade", type: "checkbox", default: false },
      { key: "background", label: "Fundo (hex)", type: "text", default: "#12130f" },
      { key: "ink", label: "Traço (hex)", type: "text", default: "#dfe7d0" },
    ],
  },
  {
    name: "flame",
    title: "Fractal flame (IFS)",
    defaultSize: { width: 1024, height: 1024 },
    controls: [
      { key: "seed", label: "Semente (deriva o conjunto)", type: "seed", default: 0 },
      { key: "transforms", label: "Transformações", type: "range", min: 2, max: 6, step: 1, default: 3 },
      { key: "iterations", label: "Iterações", type: "range", min: 100000, max: 20000000, step: 100000, default: 2000000 },
      { key: "gamma", label: "Gama (tone map)", type: "range", min: 0.5, max: 6, step: 0.1, default: 2.4 },
      { key: "vibrancy", label: "Vibração da cor", type: "range", min: 0, max: 1, step: 0.05, default: 0.85 },
      { key: "hueSpread", label: "Espalhamento de matiz", type: "range", min: 0, max: 1, step: 0.05, default: 0.7 },
      { key: "symmetry", label: "Simetria rotacional", type: "range", min: 0, max: 12, step: 1, default: 0 },
      { key: "background", label: "Fundo (hex)", type: "text", default: "#050507" },
    ],
  },
  {
    name: "chladni",
    title: "Figuras de Chladni (Cimática)",
    defaultSize: { width: 1024, height: 1024 },
    controls: [
      { key: "seed", label: "Semente", type: "seed", default: 42 },
      { key: "n", label: "Harmônico X (n)", type: "range", min: 1, max: 12, step: 1, default: 3 },
      { key: "m", label: "Harmônico Y (m)", type: "range", min: 1, max: 12, step: 1, default: 5 },
      { key: "a", label: "Amplitude a", type: "range", min: 0.2, max: 3, step: 0.1, default: 1 },
      { key: "b", label: "Amplitude b", type: "range", min: 0.2, max: 3, step: 0.1, default: 1 },
      { key: "particles", label: "Grãos de areia", type: "range", min: 10000, max: 300000, step: 10000, default: 80000 },
      { key: "time", label: "Tempo (fase)", type: "range", min: 0, max: 60, step: 0.1, default: 0 },
      { key: "glow", label: "Ondas estacionárias", type: "checkbox", default: true },
      {
        key: "palette",
        label: "Paleta",
        type: "select",
        options: ["sand", "copper", "monochrome", "cyan"],
        default: "sand",
      },
      { key: "background", label: "Fundo (hex, opcional)", type: "text", default: "" },
      { key: "ink", label: "Areia (hex, opcional)", type: "text", default: "" },
    ],
  },
  {
    name: "reactiondiffusion",
    title: "Reação-Difusão (Turing)",
    defaultSize: { width: 1024, height: 1024 },
    controls: [
      { key: "seed", label: "Semente", type: "seed", default: 101 },
      {
        key: "preset",
        label: "Preset Turing",
        type: "select",
        options: ["coral", "mitosis", "spots", "labyrinth", "rings", "custom"],
        default: "coral",
      },
      { key: "feed", label: "Feed F (custom)", type: "range", min: 0.01, max: 0.09, step: 0.001, default: 0.0545, showIf: (v) => v.preset === "custom" },
      { key: "kill", label: "Kill k (custom)", type: "range", min: 0.04, max: 0.07, step: 0.001, default: 0.0620, showIf: (v) => v.preset === "custom" },
      { key: "steps", label: "Passos de evolução", type: "range", min: 200, max: 2000, step: 50, default: 750 },
      { key: "gridSize", label: "Resolução interna", type: "range", min: 80, max: 220, step: 10, default: 140 },
      {
        key: "palette",
        label: "Paleta",
        type: "select",
        options: ["bioluminescent", "bone", "oxblood", "monochrome"],
        default: "bioluminescent",
      },
      { key: "invert", label: "Inverter padrão", type: "checkbox", default: false },
    ],
  },
];
