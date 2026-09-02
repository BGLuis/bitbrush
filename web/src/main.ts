// Entry point: register the pixel-work backends (side-effecting imports, order
// unchanged from before the Svelte port), warm one up, then mount the shell.

import { mount } from "svelte";
import "./style.css";
import "./backends/cpu"; // main-thread CPU backend (reference / last-resort)
import "./backends/worker"; // Web Worker backend (preferred)
import "./backends/gpu"; // WebGL2 accelerator (opt-in via settings)
import { getBackend } from "./backend";
import App from "./app/App.svelte";
import { ui } from "./app/store.svelte";

// Warm the resolved backend so the first render isn't also paying for WASM
// instantiation.
void getBackend(ui.settings.backend);

const target = document.getElementById("app");
if (!target) throw new Error("#app element missing from index.html");

mount(App, { target });

// Handy during bring-up: `bitbrush.getOriginal()`, `bitbrush.applyFilter(...)`.
Object.assign(window, {
  bitbrush: {
    getOriginal: () => ui.original,
    applyFilter: async (name: string, img: ImageData, params: Record<string, unknown>) =>
      (await getBackend(ui.settings.backend)).applyFilter(name, img, params as never),
  },
});
