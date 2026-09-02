// Shared app state for the Estúdio shell, as Svelte 5 runes. Components import
// `ui` and read/mutate it directly instead of prop-drilling through the shell.
//
// Framework-agnostic modules (backend, state URL codec, settings, wasm, the
// effect descriptors) stay where they are — this only wires them to the UI.

import { effects, type EffectUI } from "../ui/controls";
import { readStateFromURL } from "../state";
import { readSettings, type Settings } from "../settings";
import type { FilterParams } from "../wasm";

export type Mode = "filter" | "generator" | "palette";

export function effectByName(name: string): EffectUI | undefined {
  return effects.find((e) => e.name === name);
}

/** Build a params object from an effect's descriptors, optionally seeded from
 *  values carried in on a shared link. Reset always targets these defaults. */
export function defaultParams(name: string, seed?: FilterParams): FilterParams {
  const eff = effectByName(name);
  const out: FilterParams = {};
  if (!eff) return out;
  for (const c of eff.controls) {
    out[c.key] = seed && c.key in seed ? seed[c.key]! : c.default;
  }
  return out;
}

const url = readStateFromURL();
const startEffect = url.effect && effectByName(url.effect) ? url.effect : effects[0].name;

export const ui = $state({
  mode: "filter" as Mode,
  effectName: startEffect,
  params: defaultParams(startEffect, url.effect ? url.params : undefined),
  original: null as ImageData | null,
  preview: null as ImageData | null,
  settings: readSettings() as Settings,
  /** Live query string, refreshed whenever the recipe is written to the URL. */
  query: location.search,
});

/** DOM handles a few parts need but that aren't reactive state. */
export const refs: { canvas: HTMLCanvasElement | null } = { canvas: null };

export function selectEffect(name: string): void {
  ui.mode = "filter";
  ui.effectName = name;
  // Switching effects starts from that effect's own defaults.
  ui.params = defaultParams(name);
}

export function selectGenerator(mode: "generator" | "palette"): void {
  ui.mode = mode;
}

export function resetParams(): void {
  ui.params = defaultParams(ui.effectName);
}
