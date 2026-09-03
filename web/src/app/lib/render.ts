// The live filter-render loop: rAF-coalesced, out-of-order-guarded, backend-
// driven. Lifted out of the old main.ts so App.svelte only has to say "params
// changed" and this decides when/whether to actually run.

import { getBackend, type BackendPref } from "../../backend";
import type { FilterParams } from "../../wasm";
import { putImageData } from "../../canvas";

interface Job {
  canvas: HTMLCanvasElement;
  name: string;
  src: ImageData;
  params: FilterParams;
  pref: BackendPref;
}

let debounceTimer: number | undefined;
let latest: Job | null = null;
let queuedJob: Job | null = null;
let inFlight = false;
let seq = 0;

/** Ask for a render. Rapid calls (slider drags) collapse into an in-flight gate
 *  with a gentle 50ms debounce: slider movements remain fluid while heavy
 *  filters wait for the user to pause/settle before computing. */
export function scheduleFilterRender(job: Job): void {
  latest = job;
  if (inFlight) {
    queuedJob = job;
    return;
  }
  if (debounceTimer !== undefined) {
    clearTimeout(debounceTimer);
  }
  debounceTimer = window.setTimeout(() => {
    debounceTimer = undefined;
    if (latest && !inFlight) {
      void run(latest);
    }
  }, 50);
}

async function run(job: Job): Promise<void> {
  inFlight = true;
  queuedJob = null;
  const mine = ++seq;
  try {
    const backend = await getBackend(job.pref);
    const out = await backend.applyFilter(job.name, job.src, job.params);
    if (mine !== seq) return; // a newer render already superseded this one
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

