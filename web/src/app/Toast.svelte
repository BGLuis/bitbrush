<script lang="ts">
  import { toastState, dismissToast } from "./lib/toast.svelte";
</script>

<div class="toasts" role="status" aria-live="polite">
  {#each toastState.items as t (t.id)}
    <button type="button" class="toast" class:error={t.kind === "error"} onclick={() => dismissToast(t.id)}>
      {t.msg}
    </button>
  {/each}
</div>

<style>
  .toasts {
    position: fixed;
    left: 50%;
    bottom: 46px;
    transform: translateX(-50%);
    z-index: 300;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    pointer-events: none;
  }
  .toast {
    pointer-events: auto;
    border: 1px solid var(--line);
    background: var(--s2);
    color: var(--text);
    font-family: var(--font);
    font-size: 12.5px;
    padding: 8px 14px;
    border-radius: var(--radius);
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.45);
    cursor: pointer;
    animation: rise 0.16s ease;
  }
  .toast.error {
    border-color: #a8443f;
    color: #ffb9b4;
  }
  @keyframes rise {
    from {
      opacity: 0;
      transform: translateY(6px);
    }
  }
</style>
