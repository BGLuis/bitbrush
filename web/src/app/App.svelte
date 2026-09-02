<script lang="ts">
  import Topbar from "./Topbar.svelte";
  import ToolRail from "./ToolRail.svelte";
  import Stage from "./Stage.svelte";
  import Inspector from "./Inspector.svelte";
  import Statusbar from "./Statusbar.svelte";
  import { ui, refs } from "./store.svelte";
  import { scheduleFilterRender } from "./lib/render";
  import { writeStateToURL } from "../state";
  import type { FilterParams } from "../wasm";

  // The live filter loop: any change to effect / params / preview / engine
  // rewrites the URL recipe and asks render.ts for a (coalesced) repaint.
  // Generator and palette modes draw the canvas through their own panels.
  $effect(() => {
    if (ui.mode !== "filter") return;
    const name = ui.effectName;
    const params = $state.snapshot(ui.params) as FilterParams;
    const src = ui.preview;
    const pref = ui.settings.backend;

    writeStateToURL(name, params);
    ui.query = location.search;

    const canvas = refs.canvas;
    if (!src || !canvas) return;
    scheduleFilterRender({ canvas, name, src, params, pref });
  });
</script>

<div class="shell">
  <Topbar />
  <main class="body">
    <ToolRail />
    <Stage />
    <Inspector />
  </main>
  <Statusbar />
</div>

<style>
  .shell {
    height: 100%;
    min-width: 960px;
    display: grid;
    grid-template-rows: 52px 1fr 30px;
    background: var(--bg);
  }
  .body {
    display: grid;
    grid-template-columns: 248px 1fr 320px;
    min-height: 0;
  }
</style>
