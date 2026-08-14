// Package registry wires every built-in transform primitive into a
// transform.Registry so the template engine can build a pipeline from a
// template's named steps. This is the one place that imports all primitives;
// each primitive package stays independent and unaware of the others.
package registry

import (
	ctransform "github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/aggregate"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/excludeofftime"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/groupby"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/passthrough"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/rate"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/sessionize"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/splitbyactivedays"
)

// Default returns a registry with every built-in primitive registered under its
// canonical name.
func Default() *ctransform.Registry {
	r := ctransform.NewRegistry()
	r.Register(sessionize.Name, sessionize.New)
	r.Register(excludeofftime.Name, excludeofftime.New)
	r.Register(splitbyactivedays.Name, splitbyactivedays.New)
	r.Register(groupby.Name, groupby.New)
	r.Register(aggregate.Name, aggregate.New)
	r.Register(rate.Name, rate.New)
	r.Register(passthrough.Name, passthrough.New)

	return r
}
