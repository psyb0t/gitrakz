// Fetch wrapper — the "log shipper" of REQUEST-ID CORRELATION: every call
// mints a UUID v4, sends it as X-Request-Id, reads back whatever the server
// echoed (aichteeteapee's RequestID middleware always echoes one, generating
// its own if ours didn't validate), and console.logs a structured line so a
// full request reconstructs by matching request_id across this console and
// the server's `requestId`-tagged log line. There is no /api/logs endpoint
// to ship these to — the browser console IS the client-side half of the
// correlation, so console.log/console.error here are the deliverable, not
// stray debug output.

import { CONTENT_TYPE_JSON, REQUEST_ID_HEADER } from "$lib/common/constants";
import { ApiError, NetworkError } from "$lib/common/errors";
import type { ErrorResponse } from "$lib/common/types";

interface RequestLogFields {
  request_id: string;
  method: string;
  path: string;
  status: number;
  duration_ms: number;
}

export type QueryValue = string | number | boolean | undefined | null;
export type QueryParams = Record<string, QueryValue>;

export interface ApiRequestOptions {
  method?: string;
  body?: unknown;
  query?: QueryParams;
}

function generateRequestID(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  // Fallback for a non-secure-context browser without crypto.randomUUID —
  // Math.random is fine here since this is a correlation id, not a secret.
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === "x" ? r : (r & 0x3) | 0x8;

    return v.toString(16);
  });
}

function buildURL(path: string, query?: QueryParams): string {
  const url = new URL(path, window.location.origin);

  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value === undefined || value === null || value === "") {
        continue;
      }

      url.searchParams.set(key, String(value));
    }
  }

  return url.pathname + url.search;
}

/** Reads and parses the JSON error envelope off a failed response, if any. */
async function readErrorBody(
  response: Response,
): Promise<ErrorResponse | undefined> {
  try {
    return (await response.json()) as ErrorResponse;
  } catch (err) {
    // Non-JSON or empty error body (e.g. a raw 502 from an intermediary) —
    // the caller falls back to response.statusText, this is not fatal.
    console.debug("api error body was not JSON", {
      status: response.status,
      reason: "error_body_parse_failed",
      err,
    });

    return undefined;
  }
}

async function doFetch(
  path: string,
  options: ApiRequestOptions,
): Promise<{ response: Response; requestId: string; url: string }> {
  const method = options.method ?? "GET";
  const requestId = generateRequestID();
  const url = buildURL(path, options.query);
  const start = performance.now();

  let response: Response;
  try {
    response = await fetch(url, {
      method,
      headers: {
        "Content-Type": CONTENT_TYPE_JSON,
        [REQUEST_ID_HEADER]: requestId,
      },
      body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
    });
  } catch (err) {
    const durationMs = Math.round(performance.now() - start);
    console.error("api request network failure", {
      request_id: requestId,
      method,
      path: url,
      status: 0,
      duration_ms: durationMs,
      err,
    });

    throw new NetworkError(
      err instanceof Error ? err.message : "network error",
      0,
      "NETWORK_ERROR",
      requestId,
    );
  }

  const echoedRequestId = response.headers.get(REQUEST_ID_HEADER) ?? requestId;
  const durationMs = Math.round(performance.now() - start);
  const logFields: RequestLogFields = {
    request_id: echoedRequestId,
    method,
    path: url,
    status: response.status,
    duration_ms: durationMs,
  };

  if (!response.ok) {
    const body = await readErrorBody(response);
    console.error("api request failed", { ...logFields, body });

    throw new ApiError(
      body?.message ?? response.statusText,
      response.status,
      body?.code ?? "UNKNOWN_ERROR",
      echoedRequestId,
      body?.details ?? null,
    );
  }

  console.log("api request", logFields);

  return { response, requestId: echoedRequestId, url };
}

/** JSON request/response round trip. Throws ApiError/NetworkError on any failure. */
export async function apiFetch<T>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<T> {
  const { response } = await doFetch(path, options);

  if (response.status === 204) {
    return undefined as T;
  }

  const contentType = response.headers.get("content-type") ?? "";
  if (!contentType.includes(CONTENT_TYPE_JSON)) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export interface BlobResult {
  blob: Blob;
  contentType: string;
}

/** Binary round trip — used by /api/export, which returns application/octet-stream. */
export async function apiFetchBlob(
  path: string,
  options: ApiRequestOptions = {},
): Promise<BlobResult> {
  const { response } = await doFetch(path, options);
  const contentType = response.headers.get("content-type") ?? "application/octet-stream";
  const blob = await response.blob();

  return { blob, contentType };
}

/** Triggers a browser download for a blob export result without navigating away. */
export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}
