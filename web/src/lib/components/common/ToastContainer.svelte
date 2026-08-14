<script lang="ts">
  import { dismissToast, getToasts } from "$lib/stores/toast.svelte";

  const toasts = $derived(getToasts());
</script>

<div class="toast-container" role="status" aria-live="polite">
  {#each toasts as toast (toast.id)}
    <div class="toast toast-{toast.kind}">
      <span class="text">{toast.text}</span>
      {#if toast.requestId}
        <span class="request-id" title="request id — matches the server log line">{toast.requestId}</span>
      {/if}
      <button
        type="button"
        class="dismiss"
        aria-label="Dismiss"
        onclick={() => dismissToast(toast.id)}
      >
        &times;
      </button>
    </div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    top: 1rem;
    right: 1rem;
    z-index: 1000;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    max-width: 24rem;
  }

  .toast {
    display: flex;
    align-items: flex-start;
    gap: 0.5rem;
    padding: 0.6rem 0.75rem;
    border-radius: var(--radius);
    border: 1px solid var(--color-border);
    background: var(--color-surface);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
    font-size: 0.85rem;
  }

  .toast-error {
    border-color: var(--color-danger);
    background: var(--color-danger-bg);
    color: var(--color-danger);
  }

  .toast-success {
    border-color: var(--color-success);
    background: var(--color-success-bg);
    color: var(--color-success);
  }

  .text {
    flex: 1;
  }

  .request-id {
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono",
      monospace;
    font-size: 0.7rem;
    opacity: 0.75;
  }

  .dismiss {
    background: none;
    border: none;
    color: inherit;
    font-size: 1rem;
    line-height: 1;
    padding: 0;
  }
</style>
