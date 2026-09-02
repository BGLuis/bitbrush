<script lang="ts">
  import { effects } from "../ui/controls";
  import { ui, selectEffect, selectGenerator } from "./store.svelte";
</script>

{#snippet ico(name: string)}
  {#if name === "pixelate"}
    <svg viewBox="0 0 24 24"><rect x="3.5" y="3.5" width="7" height="7" /><rect x="13.5" y="3.5" width="7" height="7" /><rect x="3.5" y="13.5" width="7" height="7" /><rect x="13.5" y="13.5" width="7" height="7" /></svg>
  {:else if name === "sobel"}
    <svg viewBox="0 0 24 24"><path d="M4 20 20 4M4 20V9M4 20h11" /></svg>
  {:else if name === "quantize"}
    <svg viewBox="0 0 24 24"><path d="M4 19h4v-8H4zM10 19h4V6h-4zM16 19h4v-5h-4z" /></svg>
  {:else if name === "dither"}
    <svg viewBox="0 0 24 24"><circle cx="6" cy="6" r="1.3" /><circle cx="12" cy="7" r="1.3" /><circle cx="18" cy="6" r="1.3" /><circle cx="7" cy="13" r="1.3" /><circle cx="14" cy="14" r="1.3" /><circle cx="9" cy="19" r="1.3" /><circle cx="19" cy="18" r="1.3" /></svg>
  {:else if name === "glitch"}
    <svg viewBox="0 0 24 24"><rect x="3" y="6" width="13" height="4" /><rect x="8" y="14" width="13" height="4" /></svg>
  {:else if name === "ascii"}
    <svg viewBox="0 0 24 24"><rect x="4" y="4" width="16" height="16" rx="1.5" /><path d="M9 16l3-8 3 8M10.2 13h3.6" /></svg>
  {:else if name === "gradient"}
    <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="8" /><path d="M12 4a8 8 0 0 1 0 16z" fill="currentColor" stroke="none" /></svg>
  {:else if name === "palette"}
    <svg viewBox="0 0 24 24"><rect x="3.5" y="4" width="4.5" height="16" /><rect x="9.7" y="4" width="4.5" height="16" /><rect x="16" y="4" width="4.5" height="16" /></svg>
  {/if}
{/snippet}

<nav class="rail">
  <h4>Filtros</h4>
  {#each effects as e (e.name)}
    <button
      class="tool"
      class:on={ui.mode === "filter" && ui.effectName === e.name}
      onclick={() => selectEffect(e.name)}
    >
      {@render ico(e.name)}
      <span>{e.title}</span>
    </button>
  {/each}

  <div class="sep"></div>
  <h4>Geradores</h4>
  <button
    class="tool"
    class:on={ui.mode === "generator"}
    onclick={() => selectGenerator("generator")}
  >
    {@render ico("gradient")}
    <span>Gradiente + Ruído</span>
  </button>
  <button
    class="tool"
    class:on={ui.mode === "palette"}
    onclick={() => selectGenerator("palette")}
  >
    {@render ico("palette")}
    <span>Paleta</span>
  </button>
</nav>

<style>
  .rail {
    background: var(--s1);
    border-right: 1px solid var(--line);
    padding: 14px 12px;
    display: flex;
    flex-direction: column;
    gap: 1px;
    overflow-y: auto;
  }
  h4 {
    margin: 8px 8px 7px;
    font-size: 10.5px;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    color: var(--mute);
    font-weight: 600;
  }
  .tool {
    display: flex;
    align-items: center;
    gap: 11px;
    padding: 8px 9px;
    border: 0;
    border-radius: 7px;
    background: transparent;
    color: var(--dim);
    font-weight: 500;
    font-size: 13px;
    text-align: left;
    cursor: pointer;
  }
  .tool:hover {
    background: var(--s2);
    color: var(--text);
  }
  .tool.on {
    background: var(--tint);
    color: #fff;
    box-shadow: inset 2px 0 0 var(--accent);
  }
  .tool :global(svg) {
    width: 16px;
    height: 16px;
    flex: none;
    stroke: currentColor;
    fill: none;
    stroke-width: 1.6;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .sep {
    height: 1px;
    background: var(--line-soft);
    margin: 12px 8px;
  }
</style>
