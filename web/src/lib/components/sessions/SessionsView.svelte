<script lang="ts">
  import { DEFAULT_SESSION_GAP_SECONDS } from "$lib/common/constants";
  import type { Session } from "$lib/common/types";
  import { formatHours, timeLabel } from "$lib/format";
  import { listSessions } from "$lib/resources";
  import { getFilterState } from "$lib/stores/filter.svelte";
  import { reportError } from "$lib/stores/toast.svelte";
  import LoadingInline from "$lib/components/common/LoadingInline.svelte";

  const filter = getFilterState();

  let sessions = $state<Session[]>([]);
  let loading = $state(false);
  let loadedOnce = $state(false);
  let gapSeconds = $state(DEFAULT_SESSION_GAP_SECONDS);
  let gapOverridden = $state(false);

  async function load(): Promise<void> {
    loading = true;
    try {
      const result = await listSessions({
        owner: filter.owner || undefined,
        from: filter.from,
        to: filter.to,
        gap: gapOverridden ? gapSeconds : undefined,
      });
      sessions = result.sessions;
    } catch (err) {
      reportError("Failed to load sessions", err);
    } finally {
      loading = false;
      loadedOnce = true;
    }
  }

  $effect(() => {
    void filter.owner;
    void filter.from;
    void filter.to;
    void gapOverridden;
    void gapSeconds;

    void load();
  });

  function onGapChange(e: Event & { currentTarget: HTMLInputElement }): void {
    const minutes = Number(e.currentTarget.value);
    if (Number.isFinite(minutes) && minutes > 0) {
      gapSeconds = minutes * 60;
      gapOverridden = true;
    } else {
      gapOverridden = false;
    }
  }

  const totalHours = $derived(
    sessions.reduce((sum, s) => sum + s.durationHours, 0),
  );

  const groupedByDay = $derived.by(() => {
    const map = new Map<string, Session[]>();
    for (const session of sessions) {
      const key = session.day ?? "unknown";
      const bucket = map.get(key);
      if (bucket) {
        bucket.push(session);
      } else {
        map.set(key, [session]);
      }
    }

    return [...map.entries()].sort(([a], [b]) => (a < b ? 1 : a > b ? -1 : 0));
  });
</script>

<div class="sessions-view">
  <div class="controls">
    <label class="field-inline" for="session-gap">
      Session gap (minutes)
      <input
        id="session-gap"
        type="number"
        min="1"
        placeholder={String(DEFAULT_SESSION_GAP_SECONDS / 60)}
        onchange={onGapChange}
      />
    </label>
    <span class="muted">Total: {formatHours(totalHours)} across {sessions.length} session(s)</span>
  </div>

  {#if loading && !loadedOnce}
    <LoadingInline label="Loading sessions…" />
  {:else if groupedByDay.length === 0}
    <p class="muted">No sessions in range.</p>
  {:else}
    {#each groupedByDay as [day, daySessions] (day)}
      <section class="day-group">
        <h3>{day}</h3>
        <div class="cards">
          {#each daySessions as session, i (session.owner + i)}
            <details class="card session-card">
              <summary>
                <span class="owner">{session.owner}</span>
                <span class="time-range">
                  {timeLabel(session.startTs)}&ndash;{timeLabel(session.endTs)}
                </span>
                <span class="hours">{formatHours(session.durationHours)}</span>
                <span class="muted">({session.events.length} events)</span>
              </summary>
              <ul class="event-list">
                {#each session.events as event (event.id)}
                  <li>
                    <span class="muted">{timeLabel(event.ts)}</span>
                    {event.repo} &mdash; {event.title || event.id}
                  </li>
                {/each}
              </ul>
            </details>
          {/each}
        </div>
      </section>
    {/each}
  {/if}
</div>

<style>
  .sessions-view {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .controls {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .field-inline {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.8rem;
  }

  .field-inline input {
    width: 5rem;
    padding: 0.3rem 0.5rem;
    border-radius: var(--radius);
    border: 1px solid var(--color-border);
    background: var(--color-surface);
  }

  .cards {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .session-card summary {
    display: flex;
    gap: 0.75rem;
    align-items: baseline;
    cursor: pointer;
    list-style: none;
  }

  .session-card summary::-webkit-details-marker {
    display: none;
  }

  .owner {
    font-weight: 600;
  }

  .hours {
    color: var(--color-primary);
    font-weight: 600;
  }

  .event-list {
    margin: 0.5rem 0 0;
    padding-left: 1.1rem;
    font-size: 0.85rem;
  }
</style>
