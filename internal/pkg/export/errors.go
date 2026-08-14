package export

import "errors"

// ErrUnknownBlockType is returned when a blocks.Block carries a BlockType
// this package has no serializer for.
var ErrUnknownBlockType = errors.New("unknown block type")
