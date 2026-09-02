// Global, per-viewer preferences: which engine runs the pixel work and how
// large the live preview is. These are performance/comfort choices tied to
// the device, not part of the reproducible recipe — so they live in
// localStorage, NOT the shareable URL. A shared link carries the effect and
// its params (see ./state.ts); it must not dictate the recipient's engine
// or preview resolution.

import type { BackendPref } from "./backend";

export interface Settings {
  backend: BackendPref;
  /** Longest side of the live preview in px; 0 means "no cap / full resolution". */
  previewMaxDim: number;
}

const STORE_KEY = "bitbrush.settings";

const DEFAULTS: Settings = {
  backend: "auto",
  previewMaxDim: 720,
};

const BACKEND_LABELS: Record<BackendPref, string> = {
  auto: "Automático",
  cpu: "CPU (WASM)",
  gpu: "GPU (WebGL2)",
};

const PREVIEW_OPTIONS: Array<{ value: number; label: string }> = [
  { value: 480, label: "480 px" },
  { value: 720, label: "720 px" },
  { value: 1080, label: "1080 px" },
  { value: 0, label: "Resolução plena" },
];

export function readSettings(): Settings {
  try {
    const raw = localStorage.getItem(STORE_KEY);
    if (!raw) return { ...DEFAULTS };
    const parsed = JSON.parse(raw) as Partial<Settings>;
    return {
      backend: parsed.backend === "cpu" || parsed.backend === "gpu" ? parsed.backend : "auto",
      previewMaxDim:
        typeof parsed.previewMaxDim === "number" && Number.isFinite(parsed.previewMaxDim) && parsed.previewMaxDim >= 0
          ? parsed.previewMaxDim
          : DEFAULTS.previewMaxDim,
    };
  } catch {
    return { ...DEFAULTS };
  }
}

function writeSettings(s: Settings): void {
  try {
    localStorage.setItem(STORE_KEY, JSON.stringify(s));
  } catch {
    // private mode / storage disabled — settings just don't persist
  }
}

export interface SettingsPanel {
  readonly element: HTMLElement;
}

export function renderSettings(initial: Settings, onChange: (next: Settings) => void): SettingsPanel {
  let current = { ...initial };
  const root = document.createElement("div");
  root.className = "settings";

  const emit = (patch: Partial<Settings>) => {
    current = { ...current, ...patch };
    writeSettings(current);
    onChange({ ...current });
  };

  root.append(
    labelled(
      "Motor",
      select(
        (["auto", "cpu", "gpu"] as BackendPref[]).map((v) => ({ value: v, label: BACKEND_LABELS[v] })),
        current.backend,
        (v) => emit({ backend: v as BackendPref }),
      ),
    ),
    labelled(
      "Resolução do preview",
      select(
        PREVIEW_OPTIONS.map((o) => ({ value: String(o.value), label: o.label })),
        String(current.previewMaxDim),
        (v) => emit({ previewMaxDim: Number(v) }),
      ),
    ),
  );

  return { element: root };
}

function labelled(text: string, control: HTMLElement): HTMLElement {
  const l = document.createElement("label");
  l.className = "control";
  const span = document.createElement("span");
  span.className = "control-label";
  span.textContent = text;
  l.append(span, control);
  return l;
}

function select(
  options: Array<{ value: string; label: string }>,
  value: string,
  onInput: (value: string) => void,
): HTMLSelectElement {
  const el = document.createElement("select");
  for (const o of options) {
    const opt = document.createElement("option");
    opt.value = o.value;
    opt.textContent = o.label;
    el.append(opt);
  }
  el.value = value;
  el.addEventListener("change", () => onInput(el.value));
  return el;
}
