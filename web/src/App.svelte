<script lang="ts">
  import FilterSidebar from "$lib/components/common/FilterSidebar.svelte";
  import SyncStatusBar from "$lib/components/common/SyncStatusBar.svelte";
  import ToastContainer from "$lib/components/common/ToastContainer.svelte";
  import RunnerView from "$lib/components/runner/RunnerView.svelte";
  import SessionsView from "$lib/components/sessions/SessionsView.svelte";
  import SettingsView from "$lib/components/settings/SettingsView.svelte";
  import TemplatesManager from "$lib/components/templates/TemplatesManager.svelte";
  import Timeline from "$lib/components/timeline/Timeline.svelte";

  type ViewID = "timeline" | "sessions" | "runner" | "templates" | "settings";

  interface ViewDef {
    id: ViewID;
    label: string;
  }

  const VIEWS: ViewDef[] = [
    { id: "timeline", label: "Timeline" },
    { id: "sessions", label: "Sessions" },
    { id: "runner", label: "Run a template" },
    { id: "templates", label: "Templates" },
    { id: "settings", label: "Settings" },
  ];

  // A /?tpl=<id> deep-link opens straight on the runner so RunnerView mounts
  // and applies the link (see RunnerView.applyDeepLink) — a shareable run URL.
  function initialView(): ViewID {
    return new URLSearchParams(window.location.search).has("tpl")
      ? "runner"
      : "timeline";
  }

  let view = $state<ViewID>(initialView());
</script>

<div class="app-shell">
  <header class="app-header">
    <h1>gitrakz</h1>
    <SyncStatusBar />
  </header>

  <nav class="tabs">
    {#each VIEWS as v (v.id)}
      <button
        type="button"
        class="tab"
        class:active={view === v.id}
        onclick={() => (view = v.id)}
      >
        {v.label}
      </button>
    {/each}
  </nav>

  <div class="app-body">
    {#if view !== "templates" && view !== "settings"}
      <FilterSidebar />
    {/if}
    <main class="app-main">
      {#if view === "timeline"}
        <Timeline />
      {:else if view === "sessions"}
        <SessionsView />
      {:else if view === "runner"}
        <RunnerView />
      {:else if view === "templates"}
        <TemplatesManager />
      {:else if view === "settings"}
        <SettingsView />
      {/if}
    </main>
  </div>
</div>

<ToastContainer />

<style>
  .app-shell {
    display: flex;
    flex-direction: column;
    min-height: 100vh;
  }

  .app-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem 1.25rem;
    border-bottom: 1px solid var(--color-border);
  }

  .app-header h1 {
    margin: 0;
    font-size: 1.1rem;
  }

  .tabs {
    display: flex;
    gap: 0.25rem;
    padding: 0.5rem 1.25rem 0;
    border-bottom: 1px solid var(--color-border);
  }

  .tab {
    padding: 0.5rem 0.9rem;
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    color: var(--color-text-muted);
    font-size: 0.9rem;
  }

  .tab.active {
    color: var(--color-text);
    border-bottom-color: var(--color-primary);
  }

  .app-body {
    display: flex;
    flex: 1;
    min-height: 0;
  }

  .app-main {
    flex: 1;
    padding: 1.25rem;
    overflow-y: auto;
  }
</style>
