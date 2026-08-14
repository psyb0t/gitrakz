package db

import (
	"errors"

	"gorm.io/gorm"
)

// isNotFound reports whether err is gorm's "record not found" sentinel,
// bare or wrapped. Call sites that need not-found to mean something other
// than commerr.ErrNotFound (a zero value, a false ok, ...) branch on this
// instead of a generic error-map translation.
func isNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
