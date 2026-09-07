// The live compose-render loop — the compositor twin of app/lib/render.ts.
// Same rAF-ish coalescing: a 50ms debounce, an in-flight gate, and a
// monotone seq guard so a slow stack never paints over a newer one.

import { getBackend, type BackendPref } from "../../../backend";
import type { ComposeSpec } from "../../../wasm";
import { putImageData } from "../../../canvas";

interface ComposeJob {
  canvas: HTMLCanvasElement;
  base: ImageData | null;
  spec: ComposeSpec;
  extras: ImageData[];
  pref: BackendPref;
}

let debounceTimer: number | undefined;
let latest: ComposeJob | null = null;
let queuedJob: ComposeJob | null = null;
let inFlight = false;
let seq = 0;

export function scheduleComposeRender(job: ComposeJob): void {
  latest = job;
  if (inFlight) {
    queuedJob = job;
    return;
  }
  if (debounceTimer !== undefined) clearTimeout(debounceTimer);
  debounceTimer = window.setTimeout(() => {
    debounceTimer = undefined;
    if (latest && !inFlight) void run(latest);
  }, 50);
}

async function run(job: ComposeJob): Promise<void> {
  inFlight = true;
  queuedJob = null;
  const mine = ++seq;
  try {
    const backend = await getBackend(job.pref);
    const out = await backend.renderComposite(job.base, job.spec, job.extras);
    if (mine !== seq) return; // superseded
    putImageData(job.canvas, out);
  } catch (err) {
    console.error(err);
  } finally {
    inFlight = false;
    if (queuedJob) {
      const next = queuedJob;
      queuedJob = null;
      void run(next);
    }
  }
}
