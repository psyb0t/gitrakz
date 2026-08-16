// Command repogen runs gorm-gen over the db models to produce the typed
// repositories. It is invoked by the //go:generate directive in the output
// package's gen.go, so `go generate ./...` regenerates the repositories.
package main

import (
	"flag"

	"github.com/psyb0t/gitrakz/internal/pkg/db/models"
	"gorm.io/gen"
)

const defaultOutPath = "internal/pkg/db/repositories"

func main() {
	outPath := flag.String(
		"out",
		defaultOutPath,
		"directory the generated repositories are written to",
	)

	flag.Parse()

	g := gen.NewGenerator(gen.Config{
		OutPath: *outPath,
		OutFile: "repositories.gen.go",
		Mode: gen.WithoutContext |
			gen.WithDefaultQuery |
			gen.WithQueryInterface,
	})

	g.ApplyBasic(
		models.Document{},
		models.Event{},
		models.LLMCache{},
		models.LLMSettings{},
		models.SyncState{},
		models.Template{},
	)

	g.ApplyInterface(func(EventQuerier) {}, models.Event{})

	g.Execute()
}
