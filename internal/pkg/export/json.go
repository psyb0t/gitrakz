// Package export serializes a blocks.Document into the output formats
// gitrakz exposes to callers: raw JSON, CSV, and PDF.
package export

import (
	"encoding/json"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
)

// ToJSON serializes doc as its canonical JSON representation, for piping
// the block document to another consumer.
func ToJSON(doc blocks.Document) ([]byte, error) {
	data, err := json.Marshal(doc)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "marshal document to json")
	}

	return data, nil
}
