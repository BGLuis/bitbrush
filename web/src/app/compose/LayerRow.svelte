<script lang="ts">
  import { composeStore, BLEND_MODES, SOURCE_LABELS, type LayerUI } from "./compose-store.svelte";
  import { generators } from "../../ui/generators";
  import type { BlendMode } from "../../wasm";

  let { layer, index, count }: { layer: LayerUI; index: number; count: number } = $props();

  const selected = $derived(composeStore.selected === index);

  const label = $derived.by(() => {
    if (layer.source === "generator") {
      return generators.find((g) => g.name === layer.generator)?.title ?? "Gerador";
    }
    if (layer.source === "image") {
      const slot = composeStore.slotById(layer.slotId);
      return slot ? slot.name : "Imagem (solte um arquivo)";
    }
    return SOURCE_LABELS[layer.source];
  });
</script>

<div class="row" class:sel={selected} class:off={!layer.enabled}>
  <div class="moves">
    <button
      type="button"
      title="Mover para cima"
      disabled={index === count - 1}
      onclick={() => composeStore.moveLayer(index, index + 1)}>▲</button
    >
    <button
      type="button"
      title="Mover para baixo"
      disabled={index === 0}
      onclick={() => composeStore.moveLayer(index, index - 1)}>▼</button
    >
  </div>

  <input
    type="checkbox"
    title="Ativar / desativar camada"
    checked={layer.enabled}
    onchange={() => composeStore.toggleLayer(index)}
  />

  <button type="button" class="name" onclick={() => composeStore.select(index)}>
    <span class="ttl">{label}</span>
    {#if layer.chain.length}
      <span class="badge">{layer.chain.length}⛓</span>
    {/if}
  </button>

  <select
    class="blend"
    title="Modo de mesclagem"
    value={layer.blend}
    onchange={(e) => composeStore.setBlend(index, e.currentTarget.value as BlendMode)}
  >
    {#each BLEND_MODES as m (m)}
      <option value={m}>{m}</option>
    {/each}
  </select>

  <input
    class="op"
    type="range"
    min="0"
    max="1"
    step="0.01"
    title={`Opacidade ${Math.round(layer.opacity * 100)}%`}
    value={layer.opacity}
    oninput={(e) => composeStore.setOpacity(index, Number(e.currentTarget.value))}
  />

  <button
    type="button"
    class="del"
    title="Remover camada"
    onclick={() => composeStore.removeLayer(index)}>✕</button
  >
</div>

<style>
  .row {
    display: grid;
    grid-template-columns: auto auto 1fr auto 70px auto;
    align-items: center;
    gap: 7px;
    padding: 5px 6px;
    border-radius: var(--radius-sm);
    background: var(--s2);
  }
  .row.sel {
    box-shadow: inset 0 0 0 1px var(--accent);
  }
  .row.off {
    opacity: 0.5;
  }
  .moves {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .moves button {
    border: 0;
    background: transparent;
    color: var(--dim);
    cursor: pointer;
    font-size: 8px;
    line-height: 1;
    padding: 1px 2px;
  }
  .moves button:disabled {
    opacity: 0.25;
    cursor: default;
  }
  .name {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    border: 0;
    background: transparent;
    color: var(--text);
    font-size: 12px;
    text-align: left;
    cursor: pointer;
    padding: 2px 0;
  }
  .ttl {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .badge {
    flex: none;
    font-size: 9.5px;
    color: var(--mute);
  }
  .blend {
    font-size: 10.5px;
    max-width: 92px;
  }
  .op {
    width: 70px;
  }
  .del {
    border: 0;
    background: transparent;
    color: var(--mute);
    cursor: pointer;
    font-size: 11px;
  }
  .del:hover {
    color: var(--danger, #f87171);
  }
</style>
