// Package db wires gitrakz's SQLite storage: opening the database, running
// the embedded migrations, and exposing a typed Store facade over the
// generated gorm-gen repositories.
package db

import (
	"context"

	gormsqlite "github.com/ncruces/go-sqlite3/gormlite"
	sqlitemigrate "github.com/psyb0t/common-go/db/sqlite"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/db/migrations"
	"github.com/psyb0t/gitrakz/internal/pkg/db/repositories"
	"gorm.io/gorm"
)

// migrationsRoot is "." rather than "" — the embedded migrations.FS has no
// subdirectory (the //go:embed directive embeds *.sql at its own root), and
// io/fs treats "" as an invalid path while "." means "this filesystem's
// root".
const migrationsRoot = "."

// Open opens (creating it if necessary) the SQLite database at dbPath, runs
// every pending migration, and returns a Store wired to its own generated
// query API instance (repositories.Use, not repositories.SetDefault — the
// latter mutates process-wide globals and races when more than one Store is
// opened concurrently, which every parallel test in this package does).
// ctx is accepted for API consistency with the rest of the codebase; the
// underlying driver and migration runner are synchronous and do not
// currently accept a context.
func Open(_ context.Context, dbPath string) (*Store, error) {
	gormDB, err := gorm.Open(gormsqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, ctxerrors.Wrap(err, "open sqlite database")
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "get underlying sql.DB")
	}

	if err := sqlitemigrate.MigrateUp(
		sqlDB, migrationsRoot, &migrations.FS,
	); err != nil {
		return nil, ctxerrors.Wrap(err, "migrate up")
	}

	return &Store{query: repositories.Use(gormDB)}, nil
}

// Close releases the underlying database connection.
func (s *Store) Close() error {
	sqlDB, err := s.query.UnderlyingDB().DB()
	if err != nil {
		return ctxerrors.Wrap(err, "get underlying sql.DB")
	}

	if err := sqlDB.Close(); err != nil {
		return ctxerrors.Wrap(err, "close sqlite database")
	}

	return nil
}
