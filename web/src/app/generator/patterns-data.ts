// Presets and metadata for algorithmic pattern generators
import { generators } from "../../ui/generators";

export interface PatternPreset {
  id: string;
  pattern: string;
  name: string;
  description: string;
  params: Record<string, any>;
}

export const PATTERN_PRESETS: PatternPreset[] = [
  // Contours (Living Landscape - 040)
  {
    id: "contours-classic",
    pattern: "contours",
    name: "Mapa Topográfico",
    description: "Relevo sombreado clássico em papel com curvas de nível equidistantes.",
    params: {
      seed: 1337,
      levels: 18,
      indexEvery: 5,
      scale: 1,
      warp: 0.6,
      octaves: 5,
      hillshade: true,
      palette: "paper",
      background: "",
    },
  },
  {
    id: "contours-terrain",
    pattern: "contours",
    name: "Canyon & Montanhas",
    description: "Relevo montanhoso denso com gradiente de elevação em tons de terreno.",
    params: {
      seed: 4242,
      levels: 26,
      indexEvery: 4,
      scale: 1.4,
      warp: 1.1,
      octaves: 6,
      hillshade: true,
      palette: "terrain",
      background: "",
    },
  },
  {
    id: "contours-blueprint",
    pattern: "contours",
    name: "Arquipélago Blueprint",
    description: "Estética técnica de planta náutica com fundo azul e linhas ciano.",
    params: {
      seed: 8812,
      levels: 16,
      indexEvery: 4,
      scale: 0.85,
      warp: 0.7,
      octaves: 4,
      hillshade: false,
      palette: "blueprint",
      background: "",
    },
  },
  {
    id: "contours-night",
    pattern: "contours",
    name: "Expedição Noturna",
    description: "Mapa topográfico luminescente em ambiente noturno profundo.",
    params: {
      seed: 90210,
      levels: 22,
      indexEvery: 5,
      scale: 1.2,
      warp: 0.9,
      octaves: 5,
      hillshade: true,
      palette: "night",
      background: "",
    },
  },

  // Flow Field (Sumi Field - 013)
  {
    id: "flow-sumi",
    pattern: "flowfield",
    name: "Nanquim Tradicional",
    description: "Pinceladas sumi-ê com cerdas secas e textura de papel de arroz.",
    params: {
      seed: 108,
      turbulence: 2.4,
      density: 1.0,
      curl: false,
      palette: "ink",
      grain: 0.4,
      background: "",
    },
  },
  {
    id: "flow-curl",
    pattern: "flowfield",
    name: "Vórtices Curl",
    description: "Campo sem divergência (curl) formando redemoinhos fluidos e contínuos.",
    params: {
      seed: 7041,
      turbulence: 3.5,
      density: 1.3,
      curl: true,
      palette: "ink",
      grain: 0.35,
      background: "",
    },
  },
  {
    id: "flow-indigo",
    pattern: "flowfield",
    name: "Índigo Japonês",
    description: "Tons de anil sobre papel claro com fluxo suave e denso.",
    params: {
      seed: 332,
      turbulence: 1.9,
      density: 1.25,
      curl: false,
      palette: "indigo",
      grain: 0.45,
      background: "",
    },
  },
  {
    id: "flow-vermilion",
    pattern: "flowfield",
    name: "Vermilion Imperial",
    description: "Pigmentos avermelhados intensos inspirados em selos orientais.",
    params: {
      seed: 991,
      turbulence: 2.8,
      density: 1.4,
      curl: false,
      palette: "vermilion",
      grain: 0.3,
      background: "",
    },
  },

  // Truchet
  {
    id: "truchet-arcs",
    pattern: "truchet",
    name: "Arcos de Smith",
    description: "Caminhos circulares interconectados formando tubos elegantes.",
    params: {
      seed: 42,
      tiles: 12,
      style: "arcs",
      lineWidth: 0.18,
      multiScale: false,
      colorful: false,
      background: "#141414",
      ink: "#f2efe6",
    },
  },
  {
    id: "truchet-maze",
    pattern: "truchet",
    name: "Labirinto Diagonal",
    description: "Grade de ladrilhos diagonais formando um labirinto fechado.",
    params: {
      seed: 101,
      tiles: 20,
      style: "maze",
      lineWidth: 0.14,
      multiScale: false,
      colorful: false,
      background: "#121216",
      ink: "#e2ded4",
    },
  },
  {
    id: "truchet-multiscale",
    pattern: "truchet",
    name: "Multi-Escala Quadtree",
    description: "Subdivisão hierárquica variando densidade de ladrilhos.",
    params: {
      seed: 808,
      tiles: 16,
      style: "arcs",
      lineWidth: 0.15,
      multiScale: true,
      colorful: false,
      background: "#0c0d10",
      ink: "#edf0f2",
    },
  },

  // Harmonograph
  {
    id: "harmono-duo",
    pattern: "harmonograph",
    name: "Pêndulo Duplo Harmônico",
    description: "Figuras de ressonância com desafinação sutil entre eixos.",
    params: {
      seed: 12,
      pendulums: 2,
      damping: 0.006,
      freqSpread: 0.012,
      duration: 220,
      steps: 60000,
      lineAlpha: 0.06,
      colorful: false,
      background: "#0e0e12",
      ink: "#e9e6dc",
    },
  },
  {
    id: "harmono-chroma",
    pattern: "harmonograph",
    name: "Fita Cromática",
    description: "Curvas harmônicas coloridas desenhadas com transparência sobreposta.",
    params: {
      seed: 77,
      pendulums: 3,
      damping: 0.005,
      freqSpread: 0.015,
      duration: 280,
      steps: 80000,
      lineAlpha: 0.05,
      colorful: true,
      background: "#08080c",
      ink: "#e9e6dc",
    },
  },

  // Attractor
  {
    id: "attractor-clifford",
    pattern: "attractor",
    name: "Atrator de Clifford",
    description: "Caos determinístico com tone-mapping gama suave.",
    params: {
      type: "clifford",
      seed: 13,
      a: 0,
      b: 0,
      c: 0,
      d: 0,
      iterations: 2000000,
      gamma: 2.2,
      zoom: 1,
      colorBySpeed: false,
      background: "#07070b",
      ink: "#efe7d8",
    },
  },
  {
    id: "attractor-dejong-speed",
    pattern: "attractor",
    name: "De Jong por Velocidade",
    description: "Atrator de Peter de Jong colorido pela magnitude do vetor velocidade.",
    params: {
      type: "dejong",
      seed: 89,
      a: 0,
      b: 0,
      c: 0,
      d: 0,
      iterations: 2500000,
      gamma: 2.5,
      zoom: 1.1,
      colorBySpeed: true,
      background: "#040508",
      ink: "#efe7d8",
    },
  },

  // L-System
  {
    id: "lsystem-plant",
    pattern: "lsystem",
    name: "Samambaia Fractal",
    description: "Gramática formal de Lindenmayer simulando ramificação vegetal.",
    params: {
      preset: "plant",
      axiom: "X",
      rules: "X=F+[[X]-X]-F[-FX]+X;F=FF",
      angleDeg: 25,
      iterations: 5,
      seed: 0,
      jitter: 0,
      lineWidth: 1.4,
      colorByDepth: true,
      background: "#12130f",
      ink: "#dfe7d0",
    },
  },
  {
    id: "lsystem-dragon",
    pattern: "lsystem",
    name: "Curva do Dragão",
    description: "Dobradura fractal clássica de Heighway.",
    params: {
      preset: "dragon",
      axiom: "FX",
      rules: "X=X+YF+;Y=-FX-Y",
      angleDeg: 90,
      iterations: 12,
      seed: 0,
      jitter: 0,
      lineWidth: 1.2,
      colorByDepth: false,
      background: "#0d0f14",
      ink: "#8ecae6",
    },
  },

  // Flame
  {
    id: "flame-nebula",
    pattern: "flame",
    name: "Nebulosa IFS",
    description: "Fractal flame de Draves com 3 transformações não-lineares.",
    params: {
      seed: 42,
      transforms: 3,
      iterations: 2000000,
      gamma: 2.4,
      vibrancy: 0.85,
      hueSpread: 0.7,
      symmetry: 0,
      background: "#050507",
    },
  },
  {
    id: "flame-mandala",
    pattern: "flame",
    name: "Roseta Flamejante",
    description: "Fractal flame com simetria rotacional 6-fold.",
    params: {
      seed: 314,
      transforms: 4,
      iterations: 2000000,
      gamma: 2.2,
      vibrancy: 0.9,
      hueSpread: 0.6,
      symmetry: 6,
      background: "#070409",
    },
  },

  // Chladni Resonance Figures
  {
    id: "chladni-classic",
    pattern: "chladni",
    name: "Ressonância Clássica (3×5)",
    description: "Geometria acústica nodal clássica com grãos de areia claros em placa escura.",
    params: {
      seed: 42,
      n: 3,
      m: 5,
      a: 1.0,
      b: 1.0,
      particles: 80000,
      time: 0,
      glow: true,
      palette: "sand",
    },
  },
  {
    id: "chladni-copper",
    pattern: "chladni",
    name: "Cobre Harmônico (4×4)",
    description: "Modo quadrático perfeitamente simétrico com brilho de bronze aquecido.",
    params: {
      seed: 99,
      n: 4,
      m: 4,
      a: 1.2,
      b: 0.8,
      particles: 100000,
      time: 0,
      glow: true,
      palette: "copper",
    },
  },
  {
    id: "chladni-cyan",
    pattern: "chladni",
    name: "Cimática Elétrica (2×7)",
    description: "Alta frequência longitudinal com partículas ciano luminescentes.",
    params: {
      seed: 512,
      n: 2,
      m: 7,
      a: 1.0,
      b: 1.5,
      particles: 120000,
      time: 0,
      glow: true,
      palette: "cyan",
    },
  },

  // Reaction-Diffusion (Turing)
  {
    id: "rd-coral",
    pattern: "reactiondiffusion",
    name: "Coral Ramificado",
    description: "Morfogênese fractal de coral marinho em paleta bioluminescente.",
    params: {
      seed: 101,
      preset: "coral",
      steps: 800,
      gridSize: 140,
      palette: "bioluminescent",
      invert: false,
    },
  },
  {
    id: "rd-spots",
    pattern: "reactiondiffusion",
    name: "Pele de Leopardo",
    description: "Pontos e manchas biológicas distribuídos organicamente.",
    params: {
      seed: 42,
      preset: "spots",
      steps: 900,
      gridSize: 140,
      palette: "bone",
      invert: false,
    },
  },
  {
    id: "rd-labyrinth",
    pattern: "reactiondiffusion",
    name: "Labirinto Vermiforme",
    description: "Canais contínuos e dobras sinuosas tipo convoluções cerebrais.",
    params: {
      seed: 777,
      preset: "labyrinth",
      steps: 1000,
      gridSize: 150,
      palette: "oxblood",
      invert: false,
    },
  },
  {
    id: "rd-mitosis",
    pattern: "reactiondiffusion",
    name: "Mitose Celular",
    description: "Células esféricas em divisão com núcleos contrastantes.",
    params: {
      seed: 88,
      preset: "mitosis",
      steps: 850,
      gridSize: 130,
      palette: "monochrome",
      invert: false,
    },
  },
];

export function getPatternPresets(patternName: string): PatternPreset[] {
  return PATTERN_PRESETS.filter((p) => p.pattern === patternName);
}

export function buildDefaultParams(): Record<string, Record<string, any>> {
  const map: Record<string, Record<string, any>> = {};
  for (const g of generators) {
    const bag: Record<string, any> = {};
    for (const c of g.controls) {
      bag[c.key] = c.default;
    }
    map[g.name] = bag;
  }
  return map;
}
