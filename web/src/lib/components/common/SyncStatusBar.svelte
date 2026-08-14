<script lang="ts">
  import type { SyncStatus } from "$lib/common/types";
  import { getSyncStatus, triggerSync } from "$lib/resources";
  import { pushToast, reportError } from "$lib/stores/toast.svelte";

  const POLL_INTERVAL_MS = 5000;

  let status = $state<SyncStatus | undefined>(undefined);
  let syncing = $state(false);

  async function refresh(): Promise<void> {
    try {
      status = await getSyncStatus();
    } catch (err) {
      // Non-fatal — the status bar just stays stale, but the failure is
      // still logged + reported once, not swallowed.
      reportError("Failed to load sync status", err);
    }
  }

  async function startSync(): Promise<void> {
    syncing = true;
    try {
      await triggerSync();
      pushToast("info", "Sync started");
      await refresh();
    } catch (err) {
      reportError("Failed to start sync", err);
    } finally {
      syncing = false;
    }
  }

  $effect(() => {
    void refresh();
    const interval = setInterval(() => void refresh(), POLL_INTERVAL_MS);

    return () => clearInterval(interval);
  });

  const lastSyncedLabel = $derived(
    status && status.lastSyncedTs > 0
      ? new Date(status.lastSyncedTs * 1000).toLocaleString()
      : "never",
  );
</script>

<div class="sync-status-bar">
  <span class="muted">
    Last sync: {lastSyncedLabel}
    {#if status?.inProgress}
      <span class="in-progress">(in progress…)</span>
    {/if}
  </span>
  <button
    type="button"
    class="btn"
    onclick={startSync}
    disabled={syncing || status?.inProgress}
  >
    {syncing ? "Starting…" : "Sync now"}
  </button>
</div>

<style>
  .sync-status-bar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 0.8rem;
  }

  .in-progress {
    color: var(--color-primary);
  }
</style>
