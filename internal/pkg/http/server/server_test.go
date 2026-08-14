package server

import (
	"context"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/psyb0t/gitrakz/internal/pkg/ghsync"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStore is a hand-rolled Store test double — each test wires only
// the func fields the scenario needs.
type fakeStore struct {
	listOwnersFn    func(ctx context.Context) ([]string, error)
	listReposFn     func(ctx context.Context, owner string) ([]string, error)
	queryTimelineFn func(
		ctx context.Context,
		filter db.TimelineFilter,
	) ([]types.Event, bool, error)
	listTemplatesFn func(
		ctx context.Context,
	) ([]template.Template, error)
	getTemplateFn func(
		ctx context.Context,
		id string,
	) (template.Template, error)
	saveTemplateFn   func(ctx context.Context, tmpl template.Template) error
	deleteTemplateFn func(ctx context.Context, id string) error
}

func (f *fakeStore) ListOwners(ctx context.Context) ([]string, error) {
	return f.listOwnersFn(ctx)
}

func (f *fakeStore) ListRepos(
	ctx context.Context,
	owner string,
) ([]string, error) {
	return f.listReposFn(ctx, owner)
}

func (f *fakeStore) QueryTimeline(
	ctx context.Context,
	filter db.TimelineFilter,
) ([]types.Event, bool, error) {
	return f.queryTimelineFn(ctx, filter)
}

func (f *fakeStore) ListTemplates(
	ctx context.Context,
) ([]template.Template, error) {
	return f.listTemplatesFn(ctx)
}

func (f *fakeStore) GetTemplate(
	ctx context.Context,
	id string,
) (template.Template, error) {
	return f.getTemplateFn(ctx, id)
}

func (f *fakeStore) SaveTemplate(
	ctx context.Context,
	tmpl template.Template,
) error {
	return f.saveTemplateFn(ctx, tmpl)
}

func (f *fakeStore) DeleteTemplate(ctx context.Context, id string) error {
	return f.deleteTemplateFn(ctx, id)
}

// fakeEngine is a hand-rolled Engine test double.
type fakeEngine struct {
	runFn func(
		ctx context.Context,
		tmpl template.Template,
		timeline types.Timeline,
		form types.FormValues,
	) (blocks.Document, error)
}

func (f *fakeEngine) Run(
	ctx context.Context,
	tmpl template.Template,
	timeline types.Timeline,
	form types.FormValues,
) (blocks.Document, error) {
	return f.runFn(ctx, tmpl, timeline, form)
}

// fakeSessionizer is a hand-rolled Sessionizer test double.
type fakeSessionizer struct {
	sessionsFn func(
		ctx context.Context,
		timeline types.Timeline,
	) ([]types.Session, error)
}

func (f *fakeSessionizer) Sessions(
	ctx context.Context,
	timeline types.Timeline,
) ([]types.Session, error) {
	return f.sessionsFn(ctx, timeline)
}

// fakeSyncController is a hand-rolled SyncController test double.
type fakeSyncController struct {
	triggerFn func(ctx context.Context) (ghsync.SyncResult, error)
	statusFn  func(ctx context.Context) (api.SyncStatus, error)
}

func (f *fakeSyncController) Trigger(
	ctx context.Context,
) (ghsync.SyncResult, error) {
	return f.triggerFn(ctx)
}

func (f *fakeSyncController) Status(
	ctx context.Context,
) (api.SyncStatus, error) {
	return f.statusFn(ctx)
}

// fakeLLMComposer is a hand-rolled LLMComposer test double.
type fakeLLMComposer struct {
	generateTemplateFn func(
		ctx context.Context,
		description string,
	) (template.Template, error)
}

func (f *fakeLLMComposer) GenerateTemplate(
	ctx context.Context,
	description string,
) (template.Template, error) {
	return f.generateTemplateFn(ctx, description)
}

func TestNew(t *testing.T) {
	t.Parallel()

	deps := Deps{
		Store:          &fakeStore{},
		Engine:         &fakeEngine{},
		Sessionizer:    &fakeSessionizer{},
		SyncController: &fakeSyncController{},
		LLMComposer:    &fakeLLMComposer{},
	}

	srv := New(deps)
	require.NotNil(t, srv)
	assert.Equal(t, deps, srv.deps)

	var _ api.StrictServerInterface = srv
}
