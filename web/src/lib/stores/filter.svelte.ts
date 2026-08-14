// Shared timeline filter state — the FilterSidebar writes to it, the
// Timeline/Sessions/Runner views all read it, so changing a filter
// live-refreshes every view without prop-drilling.

import type { EventType, Filter } from "$lib/common/types";

export interface FilterState {
  owner: string;
  repo: string;
  type: EventType | "";
  from?: number;
  to?: number;
}

const state = $state<FilterState>({ owner: "", repo: "", type: "" });

export function getFilterState(): FilterState {
  return state;
}

export function setOwner(owner: string): void {
  state.owner = owner;
  state.repo = ""; // repo list depends on owner — reset on owner change
}

export function setRepo(repo: string): void {
  state.repo = repo;
}

export function setType(type: EventType | ""): void {
  state.type = type;
}

export function setRange(from: number | undefined, to: number | undefined): void {
  state.from = from;
  state.to = to;
}

/** Projects the sidebar's FilterState into the API's Filter request shape,
 * dropping empty-string fields that mean "no filter". */
export function toAPIFilter(f: FilterState = state): Filter {
  const filter: Filter = {};

  if (f.owner) {
    filter.owner = f.owner;
  }
  if (f.repo) {
    filter.repo = f.repo;
  }
  if (f.type) {
    filter.type = f.type;
  }
  if (f.from !== undefined) {
    filter.from = f.from;
  }
  if (f.to !== undefined) {
    filter.to = f.to;
  }

  return filter;
}
