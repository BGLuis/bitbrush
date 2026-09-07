<script lang="ts">
  import { composeStore } from "./compose-store.svelte";
  import LayerRow from "./LayerRow.svelte";

  // Show the stack top-first (last composited = top of the list).
  const rows = $derived(
    composeStore.layers.map((layer, index) => ({ layer, index })).reverse(),
  );
</script>

<div class="list">
  {#each rows as { layer, index } (layer.id)}
    <LayerRow {layer} {index} count={composeStore.layers.length} />
  {/each}
</div>

<style>
  .list {
    display: flex;
    flex-direction: column;
    gap: 5px;
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    padding: 5px;
    background: var(--s1);
  }
</style>
