<script lang="ts">
  // Small inline SVGs per event type — no icon library dependency.
  import type { EventType } from "$lib/common/types";

  interface Props {
    type: EventType;
  }

  const { type }: Props = $props();

  const LABELS: Record<EventType, string> = {
    commit: "Commit",
    pr: "Pull request",
    review: "Review",
    issue: "Issue",
    release: "Release",
  };
</script>

<span class="event-icon event-icon-{type}" title={LABELS[type]} aria-label={LABELS[type]}>
  {#if type === "commit"}
    <svg viewBox="0 0 16 16" width="14" height="14" fill="currentColor">
      <circle cx="8" cy="8" r="3" />
      <path d="M1 8h4M11 8h4" stroke="currentColor" stroke-width="1.5" fill="none" />
    </svg>
  {:else if type === "pr"}
    <svg viewBox="0 0 16 16" width="14" height="14" fill="currentColor">
      <circle cx="4" cy="3" r="2" />
      <circle cx="4" cy="13" r="2" />
      <circle cx="12" cy="8" r="2" />
      <path
        d="M4 5v6M4 5c0 4 4 3 8 3"
        stroke="currentColor"
        stroke-width="1.5"
        fill="none"
      />
    </svg>
  {:else if type === "review"}
    <svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.5">
      <path d="M1 8s2.5-4.5 7-4.5S15 8 15 8s-2.5 4.5-7 4.5S1 8 1 8z" />
      <circle cx="8" cy="8" r="2" />
    </svg>
  {:else if type === "issue"}
    <svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.5">
      <circle cx="8" cy="8" r="6.5" />
      <path d="M8 5v4" stroke-linecap="round" />
      <circle cx="8" cy="11" r="0.75" fill="currentColor" stroke="none" />
    </svg>
  {:else}
    <svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.5">
      <path d="M2 8.5 8 2l6 6.5-6 5.5z" />
      <circle cx="6" cy="6" r="1" fill="currentColor" stroke="none" />
    </svg>
  {/if}
</span>

<style>
  .event-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 50%;
    background: var(--color-surface-alt);
    flex-shrink: 0;
  }

  .event-icon-commit {
    color: #8b949e;
  }

  .event-icon-pr {
    color: #3fb950;
  }

  .event-icon-review {
    color: #58a6ff;
  }

  .event-icon-issue {
    color: #f0883e;
  }

  .event-icon-release {
    color: #a371f7;
  }
</style>
