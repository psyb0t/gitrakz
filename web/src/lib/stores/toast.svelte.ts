// App-wide toast/error-banner store — every surfaced API failure (per the
// error-handling mandate: no empty catches, no silent nulls) pushes one of
// these instead of only logging to the console.

import { TOAST_DEFAULT_DURATION_MS } from "$lib/common/constants";

export type ToastKind = "error" | "info" | "success";

export interface ToastMessage {
  id: number;
  kind: ToastKind;
  text: string;
  requestId?: string;
}

let nextID = 0;
const toasts = $state<ToastMessage[]>([]);

export function getToasts(): ToastMessage[] {
  return toasts;
}

export function pushToast(
  kind: ToastKind,
  text: string,
  requestId?: string,
  durationMs: number = TOAST_DEFAULT_DURATION_MS,
): void {
  const id = ++nextID;
  toasts.push({ id, kind, text, requestId });

  setTimeout(() => dismissToast(id), durationMs);
}

export function dismissToast(id: number): void {
  const index = toasts.findIndex((t) => t.id === id);
  if (index !== -1) {
    toasts.splice(index, 1);
  }
}

/** Standard failure path for a caught ApiError/NetworkError — logs (via the
 * thrower, api.ts already console.error'd it) and always surfaces a toast,
 * never swallows. */
export function reportError(action: string, err: unknown): void {
  const message = err instanceof Error ? err.message : String(err);
  const requestId =
    err !== null && typeof err === "object" && "requestId" in err
      ? String((err as { requestId?: unknown }).requestId ?? "")
      : undefined;

  pushToast("error", `${action}: ${message}`, requestId);
}
