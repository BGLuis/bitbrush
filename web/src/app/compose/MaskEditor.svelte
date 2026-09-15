<script lang="ts">
  import type { MaskParams } from "../../wasm";
  import { selectionStore, type SelectionTool } from "./selection-store.svelte";

  let {
    mask,
    onchange,
    title = "Máscara",
  }: {
    mask: MaskParams | undefined;
    onchange: (mask: MaskParams | undefined) => void;
    title?: string;
  } = $props();

  let myToken = $state<number | null>(null);
  const isArming = $derived(selectionStore.active && selectionStore.token === myToken);

  function onKindChange(kind: string) {
    if (kind === "none" || !kind) {
      onchange(undefined);
    } else if (kind === "rect" || kind === "ellipse") {
      onchange({ kind, x: 0.2, y: 0.2, w: 0.6, h: 0.6, feather: 0.1, invert: false });
    } else if (kind === "luma") {
      onchange({ kind: "luma", threshold: 0.5, feather: 0.05, invert: false });
    } else if (kind === "polygon") {
      onchange({ kind: "polygon", points: [], feather: 0, invert: false });
      draw("polygon"); // an empty polygon hides the layer/stage entirely — jump straight into drawing
    }
  }

  function draw(tool: SelectionTool) {
    myToken = selectionStore.arm(tool, onchange);
  }

  function cancelDraw() {
    selectionStore.cancel();
    myToken = null;
  }

  function patch(field: keyof MaskParams, value: number | boolean) {
    if (!mask) return;
    onchange({ ...mask, [field]: value });
  }
</script>

<div class="mask-box">
  <div class="mask-hdr">
    <span class="mask-title">{title}</span>
    <select class="mask-sel" value={mask?.kind ?? "none"} onchange={(e) => onKindChange(e.currentTarget.value)}>
      <option value="none">Nenhuma (total)</option>
      <option value="rect">Retângulo</option>
      <option value="ellipse">Elipse</option>
      <option value="luma">Luminância</option>
      <option value="polygon">Polígono (laço)</option>
    </select>
  </div>

  {#if mask}
    {#if mask.kind !== "luma"}
      <div class="mask-draw">
        {#if isArming}
          <span class="drawing">Desenhando no canvas…</span>
          <button type="button" onclick={cancelDraw}>Cancelar</button>
        {:else}
          <button type="button" onclick={() => draw(mask.kind as SelectionTool)}>Desenhar no canvas</button>
        {/if}
      </div>
    {/if}

    <div class="mask-controls">
      {#if mask.kind === "rect" || mask.kind === "ellipse"}
        <label class="mask-row">
          <span>Posição X · {Math.round((mask.x ?? 0.2) * 100)}%</span>
          <input
            type="range"
            min="0"
            max="1"
            step="0.01"
            value={mask.x ?? 0.2}
            oninput={(e) => patch("x", Number(e.currentTarget.value))}
          />
        </label>
        <label class="mask-row">
          <span>Posição Y · {Math.round((mask.y ?? 0.2) * 100)}%</span>
          <input
            type="range"
            min="0"
            max="1"
            step="0.01"
            value={mask.y ?? 0.2}
            oninput={(e) => patch("y", Number(e.currentTarget.value))}
          />
        </label>
        <label class="mask-row">
          <span>Largura · {Math.round((mask.w ?? 0.6) * 100)}%</span>
          <input
            type="range"
            min="0.05"
            max="1"
            step="0.01"
            value={mask.w ?? 0.6}
            oninput={(e) => patch("w", Number(e.currentTarget.value))}
          />
        </label>
        <label class="mask-row">
          <span>Altura · {Math.round((mask.h ?? 0.6) * 100)}%</span>
          <input
            type="range"
            min="0.05"
            max="1"
            step="0.01"
            value={mask.h ?? 0.6}
            oninput={(e) => patch("h", Number(e.currentTarget.value))}
          />
        </label>
        <label class="mask-row">
          <span>Suavização · {Math.round((mask.feather ?? 0.1) * 100)}%</span>
          <input
            type="range"
            min="0"
            max="0.5"
            step="0.01"
            value={mask.feather ?? 0.1}
            oninput={(e) => patch("feather", Number(e.currentTarget.value))}
          />
        </label>
      {:else if mask.kind === "luma"}
        <label class="mask-row">
          <span>Limiar (luma) · {Math.round((mask.threshold ?? 0.5) * 100)}%</span>
          <input
            type="range"
            min="0"
            max="1"
            step="0.01"
            value={mask.threshold ?? 0.5}
            oninput={(e) => patch("threshold", Number(e.currentTarget.value))}
          />
        </label>
        <label class="mask-row">
          <span>Suavização · {Math.round((mask.feather ?? 0.05) * 100)}%</span>
          <input
            type="range"
            min="0"
            max="0.5"
            step="0.01"
            value={mask.feather ?? 0.05}
            oninput={(e) => patch("feather", Number(e.currentTarget.value))}
          />
        </label>
      {:else if mask.kind === "polygon"}
        <p class="hint">
          {mask.points?.length ? `${mask.points.length} pontos` : "Nenhum ponto — desenhe no canvas"}
        </p>
        <button
          type="button"
          class="clear-pts"
          disabled={!mask.points?.length}
          onclick={() => onchange({ ...mask, points: [] })}
        >
          Limpar pontos
        </button>
        <label class="mask-row">
          <span>Suavização · {Math.round((mask.feather ?? 0) * 100)}%</span>
          <input
            type="range"
            min="0"
            max="0.5"
            step="0.01"
            value={mask.feather ?? 0}
            oninput={(e) => patch("feather", Number(e.currentTarget.value))}
          />
        </label>
      {/if}

      <label class="mask-check">
        <input
          type="checkbox"
          checked={mask.invert ?? false}
          onchange={(e) => patch("invert", e.currentTarget.checked)}
        />
        <span>Inverter seleção</span>
      </label>
    </div>
  {/if}
</div>

<style>
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
  .mask-draw {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 11px;
  }
  .mask-draw button {
    border: 1px solid var(--line);
    background: var(--s2);
    color: var(--text);
    padding: 4px 8px;
    border-radius: var(--radius-sm);
    font-size: 11px;
    cursor: pointer;
  }
  .mask-draw button:hover {
    border-color: var(--accent);
  }
  .drawing {
    color: var(--accent);
    font-weight: 500;
  }
  .hint {
    margin: 0;
    font-size: 11px;
    color: var(--mute);
  }
  .clear-pts {
    align-self: flex-start;
    border: 1px solid var(--line);
    background: transparent;
    color: var(--dim);
    padding: 3px 8px;
    border-radius: var(--radius-sm);
    font-size: 11px;
    cursor: pointer;
  }
  .clear-pts:disabled {
    opacity: 0.4;
    cursor: default;
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
