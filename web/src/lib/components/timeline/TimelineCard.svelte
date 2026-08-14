<script lang="ts">
  import { timeLabel } from "$lib/format";
  import { isSelected, toggleSelected } from "$lib/stores/selection.svelte";
  import type { Event } from "$lib/common/types";
  import EventTypeIcon from "./EventTypeIcon.svelte";

  interface Props {
    event: Event;
  }

  const { event }: Props = $props();

  const hasDelta = $derived(
    event.additions !== null &&
      event.additions !== undefined &&
      event.deletions !== null &&
      event.deletions !== undefined,
  );
</script>

<div class="timeline-card">
  <input
    type="checkbox"
    checked={isSelected(event.id)}
    onchange={() => toggleSelected(event)}
    aria-label={`Select event ${event.id}`}
  />
  <EventTypeIcon type={event.type} />
  <div class="body">
    <div class="top-row">
      <span class="owner-repo">{event.owner}/{event.repo}</span>
      <span class="time">{timeLabel(event.ts)}</span>
    </div>
    <div class="title">
      {#if event.url}
        <a href={event.url} target="_blank" rel="noopener noreferrer">
          {event.title || event.id}
        </a>
      {:else}
        {event.title || event.id}
      {/if}
    </div>
    {#if hasDelta}
      <div class="delta">
        <span class="additions">+{event.additions}</span>
        <span class="deletions">-{event.deletions}</span>
      </div>
    {/if}
  </div>
</div>

<style>
  .timeline-card {
    display: flex;
    align-items: flex-start;
    gap: 0.6rem;
    padding: 0.6rem 0.75rem;
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    background: var(--color-surface);
  }

  .body {
    flex: 1;
    min-width: 0;
  }

  .top-row {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
    font-size: 0.75rem;
    color: var(--color-text-muted);
  }

  .owner-repo {
    font-weight: 600;
  }

  .title {
    margin-top: 0.15rem;
    font-size: 0.9rem;
    overflow-wrap: anywhere;
  }

  .delta {
    margin-top: 0.25rem;
    font-size: 0.75rem;
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono",
      monospace;
  }

  .additions {
    color: var(--color-success);
  }

  .deletions {
    color: var(--color-danger);
    margin-left: 0.4rem;
  }
</style>
