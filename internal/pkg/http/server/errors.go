package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/ctxscope"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
)

// Error codes for api.ErrorResponse.Code — UPPER_SNAKE_CASE per the
// generated ErrorResponse doc comment.
const (
	errCodeNotFound         = "NOT_FOUND"
	errCodeValidationFailed = "VALIDATION_FAILED"
	errCodePermissionDenied = "PERMISSION_DENIED"
	errCodeConflict         = "CONFLICT"
	errCodeInternal         = "INTERNAL_ERROR"
)

// validationError wraps commerr.ErrValidationFailed with msg, for
// request-shape checks a handler runs before touching any dependency.
func validationError(msg string) error {
	return ctxerrors.Wrap(commerr.ErrValidationFailed, msg)
}

// mapError translates err into the HTTP status + envelope body every
// operation's generated "default" response variant carries.
func mapError(err error) (int, api.ErrorResponse) {
	switch {
	case errors.Is(err, commerr.ErrNotFound):
		return http.StatusNotFound, api.ErrorResponse{
			Code:    errCodeNotFound,
			Message: err.Error(),
		}
	case errors.Is(err, commerr.ErrValidationFailed):
		return http.StatusBadRequest, api.ErrorResponse{
			Code:    errCodeValidationFailed,
			Message: err.Error(),
		}
	case errors.Is(err, commerr.ErrPermissionDenied):
		return http.StatusForbidden, api.ErrorResponse{
			Code:    errCodePermissionDenied,
			Message: err.Error(),
		}
	case errors.Is(err, commerr.ErrConflict),
		errors.Is(err, commerr.ErrAlreadyExists):
		return http.StatusConflict, api.ErrorResponse{
			Code:    errCodeConflict,
			Message: err.Error(),
		}
	default:
		return http.StatusInternalServerError, api.ErrorResponse{
			Code:    errCodeInternal,
			Message: "internal error",
		}
	}
}

// respondError maps err to its HTTP status + envelope body and logs the
// outcome at a severity matching the status: unexpected server-side
// failures at Error, client-caused rejections (validation, not-found,
// permission-denied, conflict) at Warn with the error code as the
// "reason" field.
func respondError(
	ctx context.Context,
	action string,
	err error,
) (int, api.ErrorResponse) {
	logger := ctxscope.GetLogger(ctx)
	status, body := mapError(err)

	if status >= http.StatusInternalServerError {
		logger.Error(action+" failed", "err", err)
	} else {
		logger.Warn(action+" rejected", "err", err, "reason", body.Code)
	}

	return status, body
}
