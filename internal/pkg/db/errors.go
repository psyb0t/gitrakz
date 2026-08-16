package db

import (
	"errors"

	"gorm.io/gorm"
)

// isNotFound reports whether err is gorm's "record not found" sentinel,
// bare or wrapped, for call sites that need not-found to mean something
// other than commerr.ErrNotFound (a zero value, a false ok, ...).
func isNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
