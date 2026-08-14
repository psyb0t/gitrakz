// Typed error hierarchy for the SPA — mirrors the Go project's
// ctxerrors/commerr sentinel-vocabulary convention: callers `instanceof`
// switch on error type instead of string-matching messages.

export class AppError extends Error {
  constructor(message: string, public readonly cause?: unknown) {
    super(message);
    this.name = this.constructor.name;
  }
}

/**
 * ApiError is thrown by every failed call through lib/api.ts's apiFetch /
 * apiFetchBlob. It carries the same envelope the Go server maps errors to
 * (see internal/pkg/http/server/errors.go: NOT_FOUND, VALIDATION_FAILED,
 * PERMISSION_DENIED, CONFLICT, INTERNAL_ERROR) plus the request id so a UI
 * error banner can point back at the matching server log line.
 */
export class ApiError extends AppError {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code: string,
    public readonly requestId: string,
    public readonly details?: Record<string, unknown> | null,
  ) {
    super(message);
  }
}

/** Thrown when a network-level fetch failure means no HTTP response ever came back. */
export class NetworkError extends ApiError {}

/** A response body didn't match the shape a caller expected. */
export class DecodeError extends AppError {}
