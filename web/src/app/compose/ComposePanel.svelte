<script lang="ts">
  import { composeStore } from "./compose-store.svelte";
  import LayerList from "./LayerList.svelte";
  import LayerEditor from "./LayerEditor.svelte";
  import { RATIOS } from "../generator/noisefield-data";
  import type { LayerSource } from "../../wasm";

  const ADD: Array<{ source: LayerSource; label: string }> = [
    { source: "image", label: "Imagem" },
    { source: "generator", label: "Gerador" },
    { source: "gradient", label: "Gradiente" },
    { source: "noisefield", label: "Ruído" },
    { source: "base", label: "Imagem base" },
  ];

  const sel = $derived(composeStore.selectedLayer);
</script>

<div class="compose">
  <div class="hist">
    <button type="button" disabled={!composeStore.canUndo} onclick={() => composeStore.undo()}>
      ↶ Desfazer
    </button>
    <button type="button" disabled={!composeStore.canRedo} onclick={() => composeStore.redo()}>
      ↷ Refazer
    </button>
  </div>

  <LayerList />

  <div class="add">
    <span class="cap">Adicionar camada</span>
    <div class="add-grid">
      {#each ADD as a (a.source)}
        <button type="button" onclick={() => composeStore.addLayer(a.source)}>{a.label}</button>
      {/each}
    </div>
  </div>

  <label class="ratio">
    <span class="cap">Proporção do quadro (sem imagem base)</span>
    <select value={composeStore.ratio} onchange={(e) => composeStore.setRatio(e.currentTarget.value)}>
      {#each RATIOS as [label, value] (value)}
        <option {value}>{label}</option>
      {/each}
    </select>
  </label>

  {#if sel}
    <LayerEditor layer={sel} index={composeStore.selected} />
  {/if}

  {#if composeStore.status}
    <div class="status">{composeStore.status}</div>
  {/if}
</div>

<style>
  .compose {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  .hist {
    display: flex;
    gap: 6px;
  }
  .hist button {
    flex: 1;
    border: 1px solid var(--line);
    background: var(--s2);
    color: var(--text);
    padding: 7px;
    border-radius: var(--radius-sm);
    font-size: 12px;
    cursor: pointer;
  }
  .hist button:hover:not(:disabled) {
    border-color: var(--accent);
    background: var(--tint);
  }
  .hist button:disabled {
    color: var(--mute);
    cursor: default;
    opacity: 0.6;
  }
  .cap {
    font-size: 10.5px;
    letter-spacing: 1px;
    text-transform: uppercase;
    color: var(--mute);
    font-weight: 600;
  }
  .add {
    display: flex;
    flex-direction: column;
    gap: 7px;
  }
  .add-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 6px;
  }
  .add-grid button {
    border: 1px solid var(--line);
    background: var(--s2);
    color: var(--text);
    padding: 7px;
    border-radius: var(--radius-sm);
    font-size: 12px;
    cursor: pointer;
  }
  .add-grid button:hover {
    border-color: var(--accent);
    background: var(--tint);
  }
  .ratio {
    display: flex;
    flex-direction: column;
    gap: 7px;
  }
  .ratio select {
    width: 100%;
  }
  .status {
    font-size: 11.5px;
    font-family: var(--mono);
    color: var(--accent);
  }
</style>
