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

let pendingRaf = false;
let latest: Job | null = null;
let seq = 0;

/** Ask for a render. Rapid calls (slider drags) collapse to one rAF that runs
 *  with the freshest job; a slow async result that lands after a newer request
 *  is dropped rather than painted. */
export function scheduleFilterRender(job: Job): void {
  latest = job;
  if (pendingRaf) return;
  pendingRaf = true;
  requestAnimationFrame(() => {
    pendingRaf = false;
    if (latest) void run(latest);
  });
}

async function run(job: Job): Promise<void> {
  const mine = ++seq;
  try {
    const backend = await getBackend(job.pref);
    const out = await backend.applyFilter(job.name, job.src, job.params);
    if (mine !== seq) return; // a newer render already superseded this one
    putImageData(job.canvas, out);
  } catch (err) {
    console.error(err);
  }
}
