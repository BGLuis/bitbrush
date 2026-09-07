<script lang="ts">
  import { composeStore, type LayerUI } from "./compose-store.svelte";
  import { effects } from "../../ui/controls";
  import ParamForm from "../ParamForm.svelte";

  const controlsFor = (name: string) => effects.find((e) => e.name === name)?.controls ?? [];

  let { layer, index }: { layer: LayerUI; index: number } = $props();

  let addPick = $state("");
  let open = $state<Record<string, boolean>>({});

  function add() {
    if (!addPick) return;
    composeStore.addStage(index, addPick);
    addPick = "";
  }
</script>

<div class="chain">
  <div class="cap">Cadeia de filtros ({layer.chain.length})</div>

  {#each layer.chain as st, j (st.id)}
    <div class="stage">
      <div class="stage-hd">
        <div class="ord">
          <button
            type="button"
            title="Subir"
            disabled={j === 0}
            onclick={() => composeStore.moveStage(index, j, j - 1)}>▲</button
          >
          <button
            type="button"
            title="Descer"
            disabled={j === layer.chain.length - 1}
            onclick={() => composeStore.moveStage(index, j, j + 1)}>▼</button
          >
        </div>
        <select
          value={st.filter}
          onchange={(e) => composeStore.setStageFilter(index, j, e.currentTarget.value)}
        >
          {#each effects as e (e.name)}
            <option value={e.name}>{e.title}</option>
          {/each}
        </select>
        <button
          type="button"
          class="exp"
          title="Ajustes"
          onclick={() => (open[st.id] = !open[st.id])}>{open[st.id] ? "▾" : "▸"}</button
        >
        <button
          type="button"
          class="del"
          title="Remover filtro"
          onclick={() => composeStore.removeStage(index, j)}>✕</button
        >
      </div>
      {#if open[st.id]}
        <div class="stage-body">
          <ParamForm
            controls={controlsFor(st.filter)}
            params={st.params}
            onchange={() => composeStore.scheduleRender()}
          />
        </div>
      {/if}
    </div>
  {/each}

  <div class="add">
    <select bind:value={addPick}>
      <option value="">+ adicionar filtro…</option>
      {#each effects as e (e.name)}
        <option value={e.name}>{e.title}</option>
      {/each}
    </select>
    <button type="button" disabled={!addPick} onclick={add}>Add</button>
  </div>
</div>

<style>
  .chain {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .cap {
    font-size: 10.5px;
    letter-spacing: 1px;
    text-transform: uppercase;
    color: var(--mute);
    font-weight: 600;
  }
  .stage {
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    background: var(--s1);
  }
  .stage-hd {
    display: grid;
    grid-template-columns: auto 1fr auto auto;
    align-items: center;
    gap: 6px;
    padding: 5px 6px;
  }
  .ord {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .ord button {
    border: 0;
    background: transparent;
    color: var(--dim);
    cursor: pointer;
    font-size: 8px;
    line-height: 1;
  }
  .ord button:disabled {
    opacity: 0.25;
  }
  .exp,
  .del {
    border: 0;
    background: transparent;
    color: var(--mute);
    cursor: pointer;
    font-size: 11px;
  }
  .stage-body {
    padding: 8px 8px 10px;
    border-top: 1px solid var(--line);
  }
  .add {
    display: flex;
    gap: 6px;
  }
  .add select {
    flex: 1;
  }
  .add button {
    border: 1px solid var(--line);
    background: var(--s2);
    color: var(--text);
    border-radius: var(--radius-sm);
    padding: 0 10px;
    cursor: pointer;
    font-size: 12px;
  }
  .add button:disabled {
    opacity: 0.4;
    cursor: default;
  }
</style>
