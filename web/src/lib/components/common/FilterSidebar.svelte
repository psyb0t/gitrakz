<script lang="ts">
  import { EVENT_TYPES } from "$lib/common/types";
  import { datetimeLocalToUnixSeconds, unixSecondsToDatetimeLocal } from "$lib/format";
  import { listOwners, listRepos } from "$lib/resources";
  import {
    getFilterState,
    setOwner,
    setRange,
    setRepo,
    setType,
  } from "$lib/stores/filter.svelte";
  import { reportError } from "$lib/stores/toast.svelte";

  const filter = getFilterState();

  let owners = $state<string[]>([]);
  let repos = $state<string[]>([]);
  let ownersLoading = $state(false);
  let reposLoading = $state(false);

  async function loadOwners(): Promise<void> {
    ownersLoading = true;
    try {
      owners = await listOwners();
    } catch (err) {
      reportError("Failed to load owners", err);
    } finally {
      ownersLoading = false;
    }
  }

  $effect(() => {
    void loadOwners();
  });

  $effect(() => {
    const owner = filter.owner;
    if (!owner) {
      repos = [];

      return;
    }

    reposLoading = true;
    listRepos(owner)
      .then((result) => {
        repos = result;
      })
      .catch((err: unknown) => {
        reportError("Failed to load repos", err);
      })
      .finally(() => {
        reposLoading = false;
      });
  });

  function onOwnerChange(e: Event & { currentTarget: HTMLSelectElement }): void {
    setOwner(e.currentTarget.value);
  }

  function onRepoChange(e: Event & { currentTarget: HTMLSelectElement }): void {
    setRepo(e.currentTarget.value);
  }

  function onTypeChange(e: Event & { currentTarget: HTMLSelectElement }): void {
    const value = e.currentTarget.value;
    setType(value === "" ? "" : (value as (typeof EVENT_TYPES)[number]));
  }

  function onFromChange(e: Event & { currentTarget: HTMLInputElement }): void {
    setRange(datetimeLocalToUnixSeconds(e.currentTarget.value), filter.to);
  }

  function onToChange(e: Event & { currentTarget: HTMLInputElement }): void {
    setRange(filter.from, datetimeLocalToUnixSeconds(e.currentTarget.value));
  }

  function clearRange(): void {
    setRange(undefined, undefined);
  }
</script>

<aside class="filter-sidebar">
  <h2>Filters</h2>

  <div class="field">
    <label for="filter-owner">Owner</label>
    <select id="filter-owner" value={filter.owner} onchange={onOwnerChange}>
      <option value="">All owners</option>
      {#each owners as owner (owner)}
        <option value={owner}>{owner}</option>
      {/each}
    </select>
    {#if ownersLoading}<span class="hint muted">Loading…</span>{/if}
  </div>

  <div class="field">
    <label for="filter-repo">Repo</label>
    <select
      id="filter-repo"
      value={filter.repo}
      onchange={onRepoChange}
      disabled={!filter.owner}
    >
      <option value="">All repos</option>
      {#each repos as repo (repo)}
        <option value={repo}>{repo}</option>
      {/each}
    </select>
    {#if reposLoading}<span class="hint muted">Loading…</span>{/if}
    {#if !filter.owner}<span class="hint muted">Select an owner first…</span>{/if}
  </div>

  <div class="field">
    <label for="filter-type">Type</label>
    <select id="filter-type" value={filter.type} onchange={onTypeChange}>
      <option value="">All types</option>
      {#each EVENT_TYPES as type (type)}
        <option value={type}>{type}</option>
      {/each}
    </select>
  </div>

  <div class="field">
    <label for="filter-from">From</label>
    <input
      id="filter-from"
      type="datetime-local"
      value={filter.from !== undefined ? unixSecondsToDatetimeLocal(filter.from) : ""}
      onchange={onFromChange}
    />
  </div>

  <div class="field">
    <label for="filter-to">To</label>
    <input
      id="filter-to"
      type="datetime-local"
      value={filter.to !== undefined ? unixSecondsToDatetimeLocal(filter.to) : ""}
      onchange={onToChange}
    />
  </div>

  {#if filter.from !== undefined || filter.to !== undefined}
    <button type="button" class="btn" onclick={clearRange}>Clear date range</button>
  {/if}
</aside>

<style>
  .filter-sidebar {
    display: flex;
    flex-direction: column;
    padding: var(--spacing-3);
    border-right: 1px solid var(--color-border);
    min-width: 15rem;
  }

  .filter-sidebar h2 {
    margin: 0 0 0.75rem;
    font-size: 1rem;
  }

  .hint {
    font-size: 0.7rem;
  }
</style>
