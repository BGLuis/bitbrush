<script lang="ts">
  import { ui, effectByName, resetParams } from "./store.svelte";
  import ParamControls from "./ParamControls.svelte";
  import GeneratorPanel from "./generator/GeneratorPanel.svelte";
  import PatternPanel from "./generator/PatternPanel.svelte";
  import ComposePanel from "./compose/ComposePanel.svelte";
  import { generatorStore } from "./generator/generator-store.svelte";
  import { panel } from "./lib/panel";
  import { makePalettePanel } from "./lib/wrapped";

  const title = $derived(
    ui.mode === "filter" ? (effectByName(ui.effectName)?.title ?? "Parâmetros") : "",
  );
</script>

<aside class="insp">
  {#if ui.mode === "filter"}
    <div class="hd">
      <h3>{title}</h3>
      <button class="reset" onclick={resetParams} title="Restaurar padrões">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7">
          <path d="M3 12a9 9 0 1 0 3-6.7M3 4v4h4" />
        </svg>
      </button>
    </div>
    <ParamControls />
  {:else if ui.mode === "pattern"}
    <div class="hd">
      <h3>{generatorStore.currentPatternUI.title}</h3>
    </div>
    <PatternPanel />
  {:else if ui.mode === "generator"}
    <div class="hd">
      <h3>Gradiente + Ruído</h3>
    </div>
    <GeneratorPanel />
  {:else if ui.mode === "compose"}
    <div class="hd">
      <h3>Compor</h3>
    </div>
    <ComposePanel />
  {:else}
    <div class="hd"><h3>Paleta</h3></div>
    <div class="wrapped" use:panel={makePalettePanel}></div>
  {/if}
</aside>

<style>
  .insp {
    background: var(--s1);
    border-left: 1px solid var(--line);
    padding: 15px;
    overflow-y: auto;
  }
  .hd {
    display: flex;
    align-items: center;
    gap: 9px;
    margin-bottom: 12px;
  }
  h3 {
    margin: 0;
    font-family: var(--disp);
    font-weight: 600;
    font-size: 13.5px;
  }
  .reset {
    margin-left: auto;
    width: 28px;
    height: 28px;
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    background: var(--s2);
    color: var(--dim);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .reset:hover {
    border-color: var(--accent);
    color: var(--text);
  }
  .reset svg {
    width: 15px;
    height: 15px;
  }
</style>
