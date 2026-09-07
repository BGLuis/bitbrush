<script lang="ts">
  // Descriptor-driven control form for the active filter — the descriptor
  // rendering now lives in the shared <ParamForm>; this wrapper binds it to
  // the store's single-effect params and keeps the ASCII "copy as text"
  // export box (filter mode only).
  import { ui, effectByName } from "./store.svelte";
  import ParamForm from "./ParamForm.svelte";
  import { getBackend } from "../backend";

  const controls = $derived(effectByName(ui.effectName)?.controls ?? []);

  let asciiStatus = $state("");

  async function copyAsciiText() {
    const src = ui.original ?? ui.preview;
    if (!src) {
      asciiStatus = "Carregue uma imagem primeiro.";
      return;
    }
    asciiStatus = "Gerando texto...";
    try {
      const backend = await getBackend(ui.settings.backend);
      const text = await backend.asciiText(src, ui.params);
      await navigator.clipboard.writeText(text);
      asciiStatus = "✓ Texto copiado!";
      setTimeout(() => (asciiStatus = ""), 2000);
    } catch (err) {
      console.error(err);
      asciiStatus = "Erro ao extrair texto.";
    }
  }

  async function downloadAsciiTxt() {
    const src = ui.original ?? ui.preview;
    if (!src) {
      asciiStatus = "Carregue uma imagem primeiro.";
      return;
    }
    asciiStatus = "Gerando arquivo...";
    try {
      const backend = await getBackend(ui.settings.backend);
      const text = await backend.asciiText(src, ui.params);
      const blob = new Blob([text], { type: "text/plain;charset=utf-8" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "bitbrush-ascii.txt";
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 15_000);
      asciiStatus = "✓ Arquivo .txt salvo!";
      setTimeout(() => (asciiStatus = ""), 2000);
    } catch (err) {
      console.error(err);
      asciiStatus = "Erro ao salvar .txt.";
    }
  }
</script>

<ParamForm {controls} params={ui.params} />

{#if ui.effectName === "ascii"}
  <div class="ascii-box">
    <div class="ascii-title">Exportar Arte em Texto</div>
    <div class="ascii-actions">
      <button type="button" class="btn-ascii" onclick={copyAsciiText}>
        📋 Copiar Texto
      </button>
      <button type="button" class="btn-ascii" onclick={downloadAsciiTxt}>
        💾 Baixar .txt
      </button>
    </div>
    {#if asciiStatus}
      <div class="ascii-status">{asciiStatus}</div>
    {/if}
  </div>
{/if}

<style>
  .ascii-box {
    margin-top: 10px;
    padding: 10px;
    background: var(--s2);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .ascii-title {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.8px;
    color: var(--dim);
  }
  .ascii-actions {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 6px;
  }
  .btn-ascii {
    border: 1px solid var(--line);
    background: var(--s3);
    color: var(--text);
    padding: 6px;
    border-radius: var(--radius-sm);
    font-size: 11.5px;
    font-weight: 500;
    cursor: pointer;
    text-align: center;
    transition: all 0.12s ease;
  }
  .btn-ascii:hover {
    border-color: var(--accent);
    background: var(--tint);
  }
  .ascii-status {
    font-size: 11px;
    font-family: var(--mono);
    color: var(--accent);
  }
</style>
