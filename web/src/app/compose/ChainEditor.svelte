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

  function onMaskKindChange(stageIdx: number, kind: string) {
    if (kind === "none" || !kind) {
      composeStore.setStageMask(index, stageIdx, undefined);
      return;
    }
    if (kind === "rect") {
      composeStore.setStageMask(index, stageIdx, {
        kind: "rect",
        x: 0.2,
        y: 0.2,
        w: 0.6,
        h: 0.6,
        feather: 0.1,
        invert: false,
      });
    } else if (kind === "ellipse") {
      composeStore.setStageMask(index, stageIdx, {
        kind: "ellipse",
        x: 0.2,
        y: 0.2,
        w: 0.6,
        h: 0.6,
        feather: 0.1,
        invert: false,
      });
    } else if (kind === "luma") {
      composeStore.setStageMask(index, stageIdx, {
        kind: "luma",
        threshold: 0.5,
        feather: 0.05,
        invert: false,
      });
    }
  }

  function touchMask() {
    composeStore.scheduleRender();
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

          <div class="mask-box">
            <div class="mask-hdr">
              <span class="mask-title">Máscara regional</span>
              <select
                class="mask-sel"
                value={st.mask?.kind ?? "none"}
                onchange={(e) => onMaskKindChange(j, e.currentTarget.value)}
              >
                <option value="none">Nenhuma (total)</option>
                <option value="rect">Retângulo</option>
                <option value="ellipse">Elipse</option>
                <option value="luma">Luminância</option>
              </select>
            </div>

            {#if st.mask}
              <div class="mask-controls">
                {#if st.mask.kind === "rect" || st.mask.kind === "ellipse"}
                  <label class="mask-row">
                    <span>Posição X · {Math.round((st.mask.x ?? 0.2) * 100)}%</span>
                    <input
                      type="range"
                      min="0"
                      max="1"
                      step="0.01"
                      value={st.mask.x ?? 0.2}
                      oninput={(e) => {
                        st.mask!.x = Number(e.currentTarget.value);
                        touchMask();
                      }}
                    />
                  </label>
                  <label class="mask-row">
                    <span>Posição Y · {Math.round((st.mask.y ?? 0.2) * 100)}%</span>
                    <input
                      type="range"
                      min="0"
                      max="1"
                      step="0.01"
                      value={st.mask.y ?? 0.2}
                      oninput={(e) => {
                        st.mask!.y = Number(e.currentTarget.value);
                        touchMask();
                      }}
                    />
                  </label>
                  <label class="mask-row">
                    <span>Largura · {Math.round((st.mask.w ?? 0.6) * 100)}%</span>
                    <input
                      type="range"
                      min="0.05"
                      max="1"
                      step="0.01"
                      value={st.mask.w ?? 0.6}
                      oninput={(e) => {
                        st.mask!.w = Number(e.currentTarget.value);
                        touchMask();
                      }}
                    />
                  </label>
                  <label class="mask-row">
                    <span>Altura · {Math.round((st.mask.h ?? 0.6) * 100)}%</span>
                    <input
                      type="range"
                      min="0.05"
                      max="1"
                      step="0.01"
                      value={st.mask.h ?? 0.6}
                      oninput={(e) => {
                        st.mask!.h = Number(e.currentTarget.value);
                        touchMask();
                      }}
                    />
                  </label>
                  <label class="mask-row">
                    <span>Suavização · {Math.round((st.mask.feather ?? 0.1) * 100)}%</span>
                    <input
                      type="range"
                      min="0"
                      max="0.5"
                      step="0.01"
                      value={st.mask.feather ?? 0.1}
                      oninput={(e) => {
                        st.mask!.feather = Number(e.currentTarget.value);
                        touchMask();
                      }}
                    />
                  </label>
                {:else if st.mask.kind === "luma"}
                  <label class="mask-row">
                    <span>Limiar (luma) · {Math.round((st.mask.threshold ?? 0.5) * 100)}%</span>
                    <input
                      type="range"
                      min="0"
                      max="1"
                      step="0.01"
                      value={st.mask.threshold ?? 0.5}
                      oninput={(e) => {
                        st.mask!.threshold = Number(e.currentTarget.value);
                        touchMask();
                      }}
                    />
                  </label>
                  <label class="mask-row">
                    <span>Suavização · {Math.round((st.mask.feather ?? 0.05) * 100)}%</span>
                    <input
                      type="range"
                      min="0"
                      max="0.5"
                      step="0.01"
                      value={st.mask.feather ?? 0.05}
                      oninput={(e) => {
                        st.mask!.feather = Number(e.currentTarget.value);
                        touchMask();
                      }}
                    />
                  </label>
                {/if}

                <label class="mask-check">
                  <input
                    type="checkbox"
                    checked={st.mask.invert ?? false}
                    onchange={(e) => {
                      st.mask!.invert = e.currentTarget.checked;
                      touchMask();
                    }}
                  />
                  <span>Inverter seleção</span>
                </label>
              </div>
            {/if}
          </div>
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
  .mask-box {
    margin-top: 10px;
    padding-top: 8px;
    border-top: 1px dashed var(--line);
    display: flex;
    flex-direction: column;
    gap: 7px;
  }
  .mask-hdr {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 6px;
  }
  .mask-title {
    font-size: 11px;
    color: var(--dim);
    font-weight: 500;
  }
  .mask-sel {
    font-size: 11px;
    padding: 2px 4px;
    max-width: 140px;
  }
  .mask-controls {
    display: flex;
    flex-direction: column;
    gap: 6px;
    background: var(--s2);
    padding: 7px;
    border-radius: var(--radius-sm);
  }
  .mask-row {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: 10.5px;
    color: var(--mute);
  }
  .mask-check {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: var(--dim);
    margin-top: 2px;
    cursor: pointer;
  }
</style>
