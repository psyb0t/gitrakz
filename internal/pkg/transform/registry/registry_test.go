package registry

import (
	"testing"

	ctransform "github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault_RegistersEveryPrimitive(t *testing.T) {
	r := Default()

	// A superset params blob that satisfies every primitive's New (aggregate
	// needs op+field, group-by needs a valid by); others ignore the extra keys.
	params := []byte(`{"op":"sum","field":"hours","by":"owner"}`)

	names := []string{
		"sessionize",
		"exclude-off-time",
		"split-by-active-days",
		"group-by",
		"aggregate",
		"rate",
		"passthrough",
	}

	for _, name := range names {
		p, err := r.Build(name, params)
		require.NoError(t, err, name)
		assert.Equal(t, name, p.Name(), name)
	}
}

func TestDefault_UnknownPrimitive(t *testing.T) {
	r := Default()

	_, err := r.Build("does-not-exist", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ctransform.ErrUnknownPrimitive)
}
