<script lang="ts">
  // Descriptor-driven control form for the active filter — the Svelte port of
  // the old ui/panel.ts renderControls(). One entry per Control in the effect's
  // descriptor; `showIf` toggles visibility without dropping the value (filters
  // ignore params they don't read).
  import { ui, effectByName } from "./store.svelte";
  import type { Control } from "../ui/controls";

  const controls = $derived(effectByName(ui.effectName)?.controls ?? []);

  const shown = (c: Control): boolean => (c.showIf ? c.showIf(ui.params) : true);

  function setNum(c: Control, raw: string): void {
    ui.params[c.key] = raw === "" ? Number(c.default) : Number(raw);
  }
  function rollSeed(c: Control): void {
    ui.params[c.key] = Math.floor(Math.random() * 1_000_000);
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
</div>

<style>
  .params {
    display: flex;
    flex-direction: column;
    gap: 13px;
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
