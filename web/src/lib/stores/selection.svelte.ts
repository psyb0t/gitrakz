// Timeline row-selection store — backs the "Range-select" flow: checking
// rows on the Timeline view feeds the Runner's "N events selected" bar and
// lets it derive a from/to range spanning the selection.

import { SvelteMap } from "svelte/reactivity";
import type { Event } from "$lib/common/types";

// Plain `$state(new Map())` does not deep-proxy Map/Set mutations — SvelteMap
// is the reactivity package's dedicated reactive Map, so .set/.delete/.clear
// each trigger dependents to re-run.
const selected = new SvelteMap<string, Event>();

export function getSelected(): Map<string, Event> {
  return selected;
}

export function isSelected(id: string): boolean {
  return selected.has(id);
}

export function toggleSelected(event: Event): void {
  if (selected.has(event.id)) {
    selected.delete(event.id);
  } else {
    selected.set(event.id, event);
  }
}

export function clearSelection(): void {
  selected.clear();
}

/** The [min, max] ts across the current selection, or undefined if empty. */
export function selectionRange(): { from: number; to: number } | undefined {
  if (selected.size === 0) {
    return undefined;
  }

  let from = Number.POSITIVE_INFINITY;
  let to = Number.NEGATIVE_INFINITY;

  for (const event of selected.values()) {
    from = Math.min(from, event.ts);
    to = Math.max(to, event.ts);
  }

  return { from, to };
}
