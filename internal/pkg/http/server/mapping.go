package server

import (
	"encoding/json"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
)

const (
	// secondsPerHour converts types.Session.DurationSeconds into the
	// api.Session.DurationHours the generated Session model carries.
	secondsPerHour = 3600.0

	// layoutBlockSourceKey is the api.Block.Data key a template.
	// LayoutBlock's Source string round-trips through, since the
	// generated Block schema (reused for both rendered Document blocks
	// and Template.Layout entries) has no dedicated "source" field.
	layoutBlockSourceKey = "source"
)

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

func intPtrOrNil(n int) *int {
	if n == 0 {
		return nil
	}

	return &n
}

func eventToAPI(ev types.Event) api.Event {
	return api.Event{
		Id:        ev.ID,
		Ts:        ev.TS,
		Type:      api.EventType(ev.Type),
		Owner:     ev.Owner,
		Repo:      ev.Repo,
		Sha:       stringPtrOrNil(ev.SHA),
		Number:    intPtrOrNil(ev.Number),
		Title:     stringPtrOrNil(ev.Title),
		Url:       stringPtrOrNil(ev.URL),
		Additions: intPtrOrNil(ev.Additions),
		Deletions: intPtrOrNil(ev.Deletions),
		Branch:    stringPtrOrNil(ev.Branch),
	}
}

func eventsToAPI(evs []types.Event) []api.Event {
	out := make([]api.Event, 0, len(evs))

	for _, ev := range evs {
		out = append(out, eventToAPI(ev))
	}

	return out
}

func sessionToAPI(s types.Session) api.Session {
	return api.Session{
		Owner:         s.Owner,
		StartTs:       s.Start,
		EndTs:         s.End,
		DurationHours: float64(s.DurationSeconds) / secondsPerHour,
		// types.Session carries no per-event breakdown (see
		// transform/sessionize), so the required "events" field is
		// always an empty, non-nil slice.
		Events: []api.Event{},
	}
}

func sessionsToAPI(sessions []types.Session) []api.Session {
	out := make([]api.Session, 0, len(sessions))

	for _, s := range sessions {
		out = append(out, sessionToAPI(s))
	}

	return out
}

// filterToDB converts an optional api.Filter (nil means "no filter") into
// a db.TimelineFilter. Page/PerPage are left at their zero values —
// callers that page (ListTimeline) or need the full range
// (queryFullTimeline) set those themselves.
func filterToDB(f *api.Filter) db.TimelineFilter {
	filter := db.TimelineFilter{}

	if f == nil {
		return filter
	}

	if f.Owner != nil {
		filter.Owner = *f.Owner
	}

	if f.Repo != nil {
		filter.Repo = *f.Repo
	}

	if f.Type != nil {
		filter.Type = string(*f.Type)
	}

	if f.From != nil {
		filter.From = *f.From
	}

	if f.To != nil {
		filter.To = *f.To
	}

	return filter
}

func blockToAPI(b blocks.Block) (api.Block, error) {
	data := map[string]any{}

	if len(b.Data) > 0 {
		if err := json.Unmarshal(b.Data, &data); err != nil {
			return api.Block{}, ctxerrors.Wrap(err, "unmarshal block data")
		}
	}

	return api.Block{Type: string(b.Type), Data: data}, nil
}

func documentToAPI(doc blocks.Document) (api.Document, error) {
	out := make(api.Document, 0, len(doc))

	for _, b := range doc {
		apiBlock, err := blockToAPI(b)
		if err != nil {
			return nil, err
		}

		out = append(out, apiBlock)
	}

	return out, nil
}

func blockFromAPI(b api.Block) (blocks.Block, error) {
	raw, err := json.Marshal(b.Data)
	if err != nil {
		return blocks.Block{}, ctxerrors.Wrap(err, "marshal block data")
	}

	return blocks.Block{Type: blocks.BlockType(b.Type), Data: raw}, nil
}

func documentFromAPI(doc api.Document) (blocks.Document, error) {
	out := make(blocks.Document, 0, len(doc))

	for _, b := range doc {
		block, err := blockFromAPI(b)
		if err != nil {
			return nil, err
		}

		out = append(out, block)
	}

	return out, nil
}

// formFieldToAPI maps a template.FormField onto api.FormField. The
// generated schema's "name" property carries what template.FormField
// calls Key — the form-field identifier the run's formValues map is
// keyed by, not a display label (that's Label).
func formFieldToAPI(f template.FormField) api.FormField {
	required := f.Required

	return api.FormField{
		Name:     f.Key,
		Label:    stringPtrOrNil(f.Label),
		Type:     string(f.Type),
		Required: &required,
		Default:  f.Default,
	}
}

func formFieldFromAPI(f api.FormField) template.FormField {
	required := false
	if f.Required != nil {
		required = *f.Required
	}

	label := ""
	if f.Label != nil {
		label = *f.Label
	}

	return template.FormField{
		Key:      f.Name,
		Label:    label,
		Type:     template.FieldType(f.Type),
		Required: required,
		Default:  f.Default,
	}
}

func stepToAPI(step template.Step) (api.TransformStep, error) {
	ts := api.TransformStep{Primitive: step.Name}

	if len(step.Params) > 0 {
		m := map[string]any{}
		if err := json.Unmarshal(step.Params, &m); err != nil {
			return api.TransformStep{}, ctxerrors.Wrap(
				err, "unmarshal step params",
			)
		}

		ts.Params = &m
	}

	return ts, nil
}

func stepFromAPI(ts api.TransformStep) (template.Step, error) {
	step := template.Step{Name: ts.Primitive}

	if ts.Params != nil {
		raw, err := json.Marshal(*ts.Params)
		if err != nil {
			return template.Step{}, ctxerrors.Wrap(
				err, "marshal step params",
			)
		}

		step.Params = raw
	}

	return step, nil
}

// layoutBlockToAPI folds a template.LayoutBlock's Source into its Data
// map under layoutBlockSourceKey, since the generated Block schema has
// no dedicated field for it.
func layoutBlockToAPI(lb template.LayoutBlock) (api.Block, error) {
	data := map[string]any{}

	if len(lb.Params) > 0 {
		if err := json.Unmarshal(lb.Params, &data); err != nil {
			return api.Block{}, ctxerrors.Wrap(
				err, "unmarshal layout block params",
			)
		}
	}

	if lb.Source != "" {
		data[layoutBlockSourceKey] = lb.Source
	}

	return api.Block{Type: lb.Type, Data: data}, nil
}

func layoutBlockFromAPI(b api.Block) (template.LayoutBlock, error) {
	lb := template.LayoutBlock{Type: b.Type}
	data := map[string]any{}

	for k, v := range b.Data {
		if k == layoutBlockSourceKey {
			if s, ok := v.(string); ok {
				lb.Source = s
			}

			continue
		}

		data[k] = v
	}

	if len(data) > 0 {
		raw, err := json.Marshal(data)
		if err != nil {
			return template.LayoutBlock{}, ctxerrors.Wrap(
				err, "marshal layout block params",
			)
		}

		lb.Params = raw
	}

	return lb, nil
}

func templateToAPI(tmpl template.Template) (api.Template, error) {
	form := make([]api.FormField, 0, len(tmpl.Form))
	for _, f := range tmpl.Form {
		form = append(form, formFieldToAPI(f))
	}

	transform := make([]api.TransformStep, 0, len(tmpl.Transform))

	for _, step := range tmpl.Transform {
		apiStep, err := stepToAPI(step)
		if err != nil {
			return api.Template{}, err
		}

		transform = append(transform, apiStep)
	}

	layout := make([]api.Block, 0, len(tmpl.Layout))

	for _, lb := range tmpl.Layout {
		apiBlock, err := layoutBlockToAPI(lb)
		if err != nil {
			return api.Template{}, err
		}

		layout = append(layout, apiBlock)
	}

	exports := make([]api.ExportFormat, 0, len(tmpl.Exports))
	for _, e := range tmpl.Exports {
		exports = append(exports, api.ExportFormat(e))
	}

	return api.Template{
		Id:          tmpl.ID,
		Name:        tmpl.Name,
		Description: stringPtrOrNil(tmpl.Description),
		Form:        form,
		Transform:   transform,
		Layout:      layout,
		Exports:     exports,
		Model:       stringPtrOrNil(tmpl.Model),
	}, nil
}

func templatesToAPI(tmpls []template.Template) ([]api.Template, error) {
	out := make([]api.Template, 0, len(tmpls))

	for _, tmpl := range tmpls {
		apiTmpl, err := templateToAPI(tmpl)
		if err != nil {
			return nil, err
		}

		out = append(out, apiTmpl)
	}

	return out, nil
}

// templateFromInput builds a non-builtin template.Template with id from
// a client-submitted api.TemplateInput (create or update).
func templateFromInput(
	id string,
	input api.TemplateInput,
) (template.Template, error) {
	form := make([]template.FormField, 0, len(input.Form))
	for _, f := range input.Form {
		form = append(form, formFieldFromAPI(f))
	}

	steps := make([]template.Step, 0, len(input.Transform))

	for _, ts := range input.Transform {
		step, err := stepFromAPI(ts)
		if err != nil {
			return template.Template{}, err
		}

		steps = append(steps, step)
	}

	layout := make([]template.LayoutBlock, 0, len(input.Layout))

	for _, b := range input.Layout {
		lb, err := layoutBlockFromAPI(b)
		if err != nil {
			return template.Template{}, err
		}

		layout = append(layout, lb)
	}

	exports := make([]string, 0, len(input.Exports))
	for _, e := range input.Exports {
		exports = append(exports, string(e))
	}

	description := ""
	if input.Description != nil {
		description = *input.Description
	}

	model := ""
	if input.Model != nil {
		model = *input.Model
	}

	return template.Template{
		ID:          id,
		Name:        input.Name,
		Description: description,
		Form:        form,
		Transform:   steps,
		Layout:      layout,
		Exports:     exports,
		Model:       model,
		Builtin:     false,
	}, nil
}
