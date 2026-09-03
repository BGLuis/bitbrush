<script lang="ts">
  // Descriptor-driven control form for the active filter — the Svelte port of
  // the old ui/panel.ts renderControls(). One entry per Control in the effect's
  // descriptor; `showIf` toggles visibility without dropping the value (filters
  // ignore params they don't read).
  import { ui, effectByName } from "./store.svelte";
  import type { Control } from "../ui/controls";
  import { getBackend } from "../backend";

  const controls = $derived(effectByName(ui.effectName)?.controls ?? []);

  const shown = (c: Control): boolean => (c.showIf ? c.showIf(ui.params) : true);

  function setNum(c: Control, raw: string): void {
    ui.params[c.key] = raw === "" ? Number(c.default) : Number(raw);
  }
  function rollSeed(c: Control): void {
    ui.params[c.key] = Math.floor(Math.random() * 1_000_000);
  }

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

<div class="params">
  {#each controls as c (c.key)}
    {#if shown(c)}
      {#if c.type === "checkbox"}
        <label class="ctl chk">
          <input
            type="checkbox"
            checked={Boolean(ui.params[c.key])}
            onchange={(e) => (ui.params[c.key] = e.currentTarget.checked)}
          />
          <span>{c.label}</span>
        </label>
      {:else if c.type === "select"}
        <label class="ctl stack">
          <span class="lab">{c.label}</span>
          <select
            value={String(ui.params[c.key])}
            onchange={(e) => (ui.params[c.key] = e.currentTarget.value)}
          >
            {#each c.options ?? [] as opt (opt)}
              <option value={opt}>{opt}</option>
            {/each}
          </select>
        </label>
      {:else if c.type === "text"}
        <label class="ctl stack">
          <span class="lab">{c.label}</span>
          <input
            type="text"
            value={String(ui.params[c.key])}
            oninput={(e) => (ui.params[c.key] = e.currentTarget.value)}
          />
        </label>
      {:else}
        <div class="ctl stack">
          <div class="lab">
            <span>{c.label}</span>
            <span class="val">{ui.params[c.key]}</span>
          </div>
          <div class="row">
            <input
              type={c.type === "range" ? "range" : "number"}
              min={c.min}
              max={c.max}
              step={c.step}
              value={Number(ui.params[c.key])}
              oninput={(e) => setNum(c, e.currentTarget.value)}
            />
            {#if c.type === "seed"}
              <button
                type="button"
                class="dice"
                title="Semente aleatória"
                aria-label="Semente aleatória"
                onclick={() => rollSeed(c)}>⚄</button
              >
            {/if}
          </div>
        </div>
      {/if}
    {/if}
  {/each}

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
</div>

<style>
  .params {
    display: flex;
    flex-direction: column;
    gap: 13px;
  }
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
  .ctl.stack {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .lab {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 12px;
    color: var(--dim);
  }
  .val {
    color: var(--text);
    font-family: var(--mono);
    font-size: 11.5px;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .row input[type="number"] {
    width: 6rem;
  }
  .chk {
    display: flex;
    align-items: center;
    gap: 9px;
    font-size: 12px;
    color: var(--dim);
  }
  .dice {
    flex: none;
    width: 32px;
    height: 30px;
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    background: var(--s2);
    cursor: pointer;
    font-size: 14px;
    line-height: 1;
  }
  .dice:hover {
    border-color: var(--accent);
  }
</style>
