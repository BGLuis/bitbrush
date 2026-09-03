<script lang="ts">
  import { ui } from "./store.svelte";
  import { refs } from "./store.svelte";
  import { openFilePicker, recomputePreview } from "./lib/image";
  import { panel } from "./lib/panel";
  import { renderSettings } from "../settings";

  import { getBackend } from "../backend";
  import { generatorStore } from "./generator/generator-store.svelte";

  let settingsOpen = $state(false);
  let shareLabel = $state("Compartilhar");
  let exportLabel = $state("Exportar PNG");

  const recipe = $derived.by(() => {
    if (ui.mode === "pattern") {
      return `padrão · ${generatorStore.currentPatternUI?.title.toLowerCase() ?? "algorítmico"}`;
    }
    if (ui.mode === "generator") return "gerador · gradiente / ruído";
    if (ui.mode === "palette") return "gerador · paleta";
    const bits = Object.entries(ui.params)
      .slice(0, 4)
      .map(([k, v]) => `${k}=${v}`);
    return [ui.effectName, ...bits].join(" · ");
  });

  function makeSettings() {
    return renderSettings(ui.settings, (next) => {
      const previewChanged = next.previewMaxDim !== ui.settings.previewMaxDim;
      ui.settings = next;
      if (previewChanged) recomputePreview();
    });
  }

  function share() {
    void navigator.clipboard?.writeText(location.href);
    shareLabel = "Link copiado!";
    setTimeout(() => (shareLabel = "Compartilhar"), 1400);
  }

  function downloadCanvas(canvas: HTMLCanvasElement, filename: string) {
    canvas.toBlob((blob) => {
      if (!blob) return;
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 15_000);
    }, "image/png");
  }

  async function exportPng() {
    if (ui.mode === "generator" || ui.mode === "pattern") {
      void generatorStore.exportHighResPNG();
      return;
    }

    const src = ui.original ?? ui.preview;
    if (!src) {
      const canvas = refs.canvas;
      if (!canvas) return;
      downloadCanvas(canvas, `bitbrush-${ui.mode}.png`);
      return;
    }

    exportLabel = "Processando...";
    try {
      const backend = await getBackend(ui.settings.backend);
      const out = await backend.applyFilter(ui.effectName, src, ui.params);
      const tempCanvas = document.createElement("canvas");
      tempCanvas.width = out.width;
      tempCanvas.height = out.height;
      const ctx = tempCanvas.getContext("2d")!;
      ctx.putImageData(out, 0, 0);
      downloadCanvas(tempCanvas, `bitbrush-${ui.effectName}-${out.width}x${out.height}.png`);
      exportLabel = "✓ Salvo!";
      setTimeout(() => (exportLabel = "Exportar PNG"), 1500);
    } catch (err) {
      console.error(err);
      exportLabel = "Erro ao exportar";
      setTimeout(() => (exportLabel = "Exportar PNG"), 2000);
    }
  }
</script>

<header class="top">
  <div class="brand"><span class="mk"></span>BitBrush</div>
  <div class="pill" title="Receita reproduzível — viaja na URL">{recipe}</div>

  <div class="right">
    <button class="btn" onclick={openFilePicker}>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round">
        <path d="M12 15V3M8 7l4-4 4 4M4 15v4a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-4" />
      </svg>
      Carregar imagem
    </button>
    <button class="btn" onclick={share}>{shareLabel}</button>
    <button class="btn" onclick={exportPng}>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round">
        <path d="M12 3v12M8 11l4 4 4-4M4 21h16" />
      </svg>
      {exportLabel}
    </button>
    <div class="gearwrap">
      <button
        class="btn icon"
        aria-label="Configurações"
        aria-expanded={settingsOpen}
        onclick={() => (settingsOpen = !settingsOpen)}
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7">
          <circle cx="12" cy="12" r="3.2" />
          <path d="M12 3v2.5M12 18.5V21M4.2 7l2.2 1.3M17.6 15.7l2.2 1.3M4.2 17l2.2-1.3M17.6 8.3l2.2-1.3" />
        </svg>
      </button>
      {#if settingsOpen}
        <div class="menu">
          <div class="menu-hd">Preferências deste dispositivo</div>
          <div use:panel={makeSettings}></div>
        </div>
      {/if}
    </div>
  </div>
</header>

<style>
  .top {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 0 18px;
    background: var(--s1);
    border-bottom: 1px solid var(--line);
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 9px;
    font-family: var(--disp);
    font-weight: 700;
    font-size: 15.5px;
  }
  .mk {
    width: 16px;
    height: 16px;
    border-radius: 5px;
    background: linear-gradient(150deg, #b58cff, #7b52e0);
  }
  .pill {
    display: flex;
    align-items: center;
    padding: 6px 11px;
    border: 1px solid var(--line);
    border-radius: var(--radius);
    background: var(--s2);
    color: var(--dim);
    font-family: var(--mono);
    font-size: 11.5px;
    max-width: 42ch;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .right {
    margin-left: auto;
    display: flex;
    gap: 8px;
    align-items: center;
  }
  .btn {
    display: flex;
    align-items: center;
    gap: 7px;
    padding: 7px 12px;
    border-radius: var(--radius);
    border: 1px solid #3a3646;
    background: var(--s2);
    color: var(--text);
    font-size: 12.5px;
    font-weight: 500;
    cursor: pointer;
  }
  .btn:hover {
    border-color: var(--accent);
  }
  .btn svg {
    width: 14px;
    height: 14px;
  }
  .btn.icon {
    padding: 7px;
  }
  .btn.icon svg {
    width: 16px;
    height: 16px;
  }
  .gearwrap {
    position: relative;
  }
  .menu {
    position: absolute;
    right: 0;
    top: calc(100% + 6px);
    z-index: 20;
    width: 300px;
    padding: 12px;
    border: 1px solid var(--line);
    border-radius: var(--radius);
    background: var(--s1);
    box-shadow: 0 16px 40px rgba(0, 0, 0, 0.5);
  }
  .menu-hd {
    font-size: 10.5px;
    letter-spacing: 1.2px;
    text-transform: uppercase;
    color: var(--mute);
    font-weight: 600;
    margin-bottom: 10px;
  }
</style>
