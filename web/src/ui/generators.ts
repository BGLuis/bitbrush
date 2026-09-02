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
      { key: "background", label: "Fundo (hex)", type: "text", default: "#0e0e12" },
      { key: "ink", label: "Traço (hex)", type: "text", default: "#e9e6dc" },
    ],
  },
];
