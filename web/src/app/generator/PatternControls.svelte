<script lang="ts">
  import { generatorStore } from "./generator-store.svelte";
  import type { Control } from "../../ui/controls";

  const controls = $derived(generatorStore.currentPatternUI.controls);
  const params = $derived(generatorStore.currentPatternParams);

  const shown = (c: Control): boolean => (c.showIf ? c.showIf(params) : true);

  function setNum(c: Control, raw: string): void {
    const val = raw === "" ? Number(c.default) : Number(raw);
    generatorStore.setPatternParam(c.key, val);
  }

  function rollSeed(c: Control): void {
    generatorStore.setPatternParam(c.key, Math.floor(Math.random() * 1_000_000));
  }
</script>

<div class="params">
  {#each controls as c (c.key)}
    {#if shown(c)}
      {#if c.type === "checkbox"}
        <label class="ctl chk">
          <input
            type="checkbox"
            checked={Boolean(params[c.key])}
            onchange={(e) => generatorStore.setPatternParam(c.key, e.currentTarget.checked)}
          />
          <span>{c.label}</span>
        </label>
      {:else if c.type === "select"}
        <label class="ctl stack">
          <span class="lab">{c.label}</span>
          <select
            value={String(params[c.key] ?? c.default)}
            onchange={(e) => generatorStore.setPatternParam(c.key, e.currentTarget.value)}
          >
            {#each c.options ?? [] as opt (opt)}
              <option value={opt}>{opt}</option>
            {/each}
          </select>
        </label>
      {:else if c.type === "text"}
        <label class="ctl stack">
          <div class="lab">
            <span>{c.label}</span>
            {#if c.key.toLowerCase().includes("color") || c.key === "background" || c.key === "ink"}
              {#if String(params[c.key]).startsWith("#")}
                <span class="color-swatch" style="background: {params[c.key]}"></span>
              {/if}
            {/if}
          </div>
          <div class="row">
            <input
              type="text"
              value={String(params[c.key] ?? c.default ?? "")}
              placeholder={String(c.default ?? "")}
              oninput={(e) => generatorStore.setPatternParam(c.key, e.currentTarget.value)}
            />
            {#if c.key.toLowerCase().includes("color") || c.key === "background" || c.key === "ink"}
              <input
                type="color"
                class="color-picker-input"
                value={String(params[c.key]).startsWith("#") && String(params[c.key]).length === 7 ? params[c.key] : "#ffffff"}
                oninput={(e) => generatorStore.setPatternParam(c.key, e.currentTarget.value)}
                title="Escolher cor"
              />
            {/if}
          </div>
        </label>
      {:else}
        <div class="ctl stack">
          <div class="lab">
            <span>{c.label}</span>
            <span class="val">{params[c.key] ?? c.default}</span>
          </div>
          <div class="row">
            <input
              type={c.type === "range" ? "range" : "number"}
              min={c.min}
              max={c.max}
              step={c.step}
              value={Number(params[c.key] ?? c.default)}
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
  .ctl {
    display: flex;
    font-size: var(--text-sm);
  }
  .chk {
    flex-direction: row;
    align-items: center;
    gap: 9px;
    cursor: pointer;
    user-select: none;
    color: var(--txt);
    padding: 2px 0;
  }
  .chk input {
    accent-color: var(--accent);
    width: 15px;
    height: 15px;
    cursor: pointer;
  }
  .stack {
    flex-direction: column;
    gap: 5px;
  }
  .lab {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: var(--text-xs);
    color: var(--dim);
    letter-spacing: 0.2px;
  }
  .val {
    font-family: var(--font-mono);
    color: var(--txt);
    font-variant-numeric: tabular-nums;
  }
  .color-swatch {
    display: inline-block;
    width: 14px;
    height: 14px;
    border-radius: 3px;
    border: 1px solid var(--line);
  }
  .row {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .row input[type="range"] {
    flex: 1;
    accent-color: var(--accent);
    cursor: pointer;
  }
  .row input[type="number"],
  .row input[type="text"],
  select {
    width: 100%;
    background: var(--s2);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    color: var(--txt);
    padding: 6px 9px;
    font-size: var(--text-sm);
    font-family: inherit;
    box-sizing: border-box;
  }
  select {
    cursor: pointer;
  }
  .row input[type="text"] {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }
  .color-picker-input {
    width: 30px;
    height: 30px;
    padding: 0;
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    background: transparent;
    cursor: pointer;
    flex-shrink: 0;
  }
  .dice {
    flex-shrink: 0;
    width: 32px;
    height: 30px;
    border: 1px solid var(--line);
    background: var(--s2);
    border-radius: var(--radius-sm);
    color: var(--txt);
    cursor: pointer;
    font-size: 15px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.15s, border-color 0.15s;
  }
  .dice:hover {
    background: var(--s3);
    border-color: var(--dim);
  }
</style>
