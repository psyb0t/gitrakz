// Date/time/number formatting helpers shared by the timeline, sessions, and
// block renderers — kept in one place so "how we show a unix-seconds
// timestamp" has a single definition.

const DAY_KEY_FORMATTER = new Intl.DateTimeFormat("en-CA", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
}); // en-CA gives YYYY-MM-DD directly, used as the day-group key

const DAY_LABEL_FORMATTER = new Intl.DateTimeFormat(undefined, {
  weekday: "long",
  year: "numeric",
  month: "long",
  day: "numeric",
});

const TIME_FORMATTER = new Intl.DateTimeFormat(undefined, {
  hour: "2-digit",
  minute: "2-digit",
});

/** Unix seconds -> a stable YYYY-MM-DD key, for grouping events by day. */
export function dayKey(unixSeconds: number): string {
  return DAY_KEY_FORMATTER.format(new Date(unixSeconds * 1000));
}

/** Unix seconds -> a human day label ("Monday, January 5, 2026"). */
export function dayLabel(unixSeconds: number): string {
  return DAY_LABEL_FORMATTER.format(new Date(unixSeconds * 1000));
}

/** Unix seconds -> a local time-of-day string ("14:32"). */
export function timeLabel(unixSeconds: number): string {
  return TIME_FORMATTER.format(new Date(unixSeconds * 1000));
}

/** Local <input type="datetime-local"> value -> unix seconds, or undefined if empty/invalid. */
export function datetimeLocalToUnixSeconds(value: string): number | undefined {
  if (!value) {
    return undefined;
  }

  const ms = new Date(value).getTime();

  return Number.isNaN(ms) ? undefined : Math.floor(ms / 1000);
}

/** Unix seconds -> the value a <input type="datetime-local"> expects. */
export function unixSecondsToDatetimeLocal(unixSeconds: number): string {
  const date = new Date(unixSeconds * 1000);
  const pad = (n: number): string => String(n).padStart(2, "0");

  return (
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}` +
    `T${pad(date.getHours())}:${pad(date.getMinutes())}`
  );
}

export function formatHours(hours: number): string {
  return `${hours.toFixed(2)}h`;
}

export function formatDelta(additions: number, deletions: number): string {
  return `+${additions}/-${deletions}`;
}
