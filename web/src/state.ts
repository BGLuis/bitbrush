// Serialises {effect, params} into the URL query string so a result — a
// glitch's seed included — is reproducible from a shared link, with no
// server involved. Only the recipe travels in the URL, never image data.

import type { FilterParams } from "./wasm";

const EFFECT_KEY = "fx";
const PARAM_PREFIX = "p.";

export interface URLState {
  effect: string | null;
  params: FilterParams;
}

export function readStateFromURL(): URLState {
  const q = new URLSearchParams(location.search);
  const params: FilterParams = {};
  for (const [key, raw] of q) {
    if (!key.startsWith(PARAM_PREFIX)) continue;
    params[key.slice(PARAM_PREFIX.length)] = decodeValue(raw);
  }
  return { effect: q.get(EFFECT_KEY), params };
}

// Replaces the current URL's query string wholesale with the given effect
// and params — never appends to browser history, so dragging a slider
// doesn't fill up the back button.
export function writeStateToURL(effect: string, params: FilterParams): void {
  const q = new URLSearchParams();
  q.set(EFFECT_KEY, effect);
  for (const [key, value] of Object.entries(params)) {
    q.set(PARAM_PREFIX + key, String(value));
  }
  const url = `${location.pathname}?${q.toString()}${location.hash}`;
  history.replaceState(null, "", url);
}

function decodeValue(raw: string): number | string | boolean {
  if (raw === "true") return true;
  if (raw === "false") return false;
  if (raw.trim() === "") return raw;
  const n = Number(raw);
  return Number.isNaN(n) ? raw : n;
}
