<script lang="ts">
  import { DEFAULT_PAGE, DEFAULT_PER_PAGE } from "$lib/common/constants";
  import type { Event } from "$lib/common/types";
  import { dayLabel, dayKey } from "$lib/format";
  import { listTimeline } from "$lib/resources";
  import { getFilterState, toAPIFilter } from "$lib/stores/filter.svelte";
  import { reportError } from "$lib/stores/toast.svelte";
  import LoadingInline from "$lib/components/common/LoadingInline.svelte";
  import TimelineCard from "./TimelineCard.svelte";

  const filter = getFilterState();

  let items = $state<Event[]>([]);
  let page = $state(DEFAULT_PAGE);
  let hasMore = $state(false);
  let loading = $state(false);
  let loadedOnce = $state(false);

  async function load(targetPage: number, replace: boolean): Promise<void> {
    loading = true;
    try {
      const result = await listTimeline({
        ...toAPIFilter(filter),
        page: targetPage,
        perPage: DEFAULT_PER_PAGE,
      });
      items = replace ? result.items : [...items, ...result.items];
      hasMore = result.hasMore;
      page = targetPage;
    } catch (err) {
      reportError("Failed to load timeline", err);
    } finally {
      loading = false;
      loadedOnce = true;
    }
  }

  // Re-fetch from page 1 whenever any filter field changes.
  $effect(() => {
    // Reading each field here is what establishes the effect's
    // dependencies — the actual filtering happens server-side.
    void filter.owner;
    void filter.repo;
    void filter.type;
    void filter.from;
    void filter.to;

    void load(DEFAULT_PAGE, true);
  });

  function loadMore(): void {
    void load(page + 1, false);
  }

  const groups = $derived.by(() => {
    const map = new Map<string, Event[]>();
    for (const item of items) {
      const key = dayKey(item.ts);
      const bucket = map.get(key);
      if (bucket) {
        bucket.push(item);
      } else {
        map.set(key, [item]);
      }
    }

    return [...map.entries()].sort(([a], [b]) => (a < b ? 1 : a > b ? -1 : 0));
  });
</script>

<div class="timeline">
  {#if loading && !loadedOnce}
    <LoadingInline label="Loading timeline…" />
  {:else if groups.length === 0}
    <p class="muted">No events match the current filters.</p>
  {:else}
    {#each groups as [key, dayItems] (key)}
      <section class="day-group">
        <h3 class="day-heading">{dayLabel(dayItems[0]?.ts ?? 0)}</h3>
        <div class="cards">
          {#each dayItems as event (event.id)}
            <TimelineCard {event} />
          {/each}
        </div>
      </section>
    {/each}

    <div class="pagination">
      {#if hasMore}
        <button type="button" class="btn" onclick={loadMore} disabled={loading}>
          {loading ? "Loading…" : "Load more"}
        </button>
      {:else}
        <span class="muted">End of results.</span>
      {/if}
    </div>
  {/if}
</div>

<style>
  .timeline {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .day-heading {
    margin: 0 0 0.5rem;
    font-size: 0.95rem;
    color: var(--color-text-muted);
  }

  .cards {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .pagination {
    display: flex;
    justify-content: center;
    padding: 0.5rem 0 1.5rem;
  }
</style>
