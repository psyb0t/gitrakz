// Package migrations embeds gitrakz's SQLite schema migrations into the
// binary so it stays self-contained — no migrations directory to mount, no
// version skew between the binary and the SQL files.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
