<script lang="ts">
  import type { MaskParams } from "../../wasm";

  let {
    mask,
    onchange,
    title = "Máscara",
  }: {
    mask: MaskParams | undefined;
    onchange: (mask: MaskParams | undefined) => void;
    title?: string;
  } = $props();

  function onKindChange(kind: string) {
    if (kind === "none" || !kind) {
      onchange(undefined);
    } else if (kind === "rect" || kind === "ellipse") {
      onchange({ kind, x: 0.2, y: 0.2, w: 0.6, h: 0.6, feather: 0.1, invert: false });
    } else if (kind === "luma") {
      onchange({ kind: "luma", threshold: 0.5, feather: 0.05, invert: false });
    }
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
    </select>
  </div>

  {#if mask}
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
