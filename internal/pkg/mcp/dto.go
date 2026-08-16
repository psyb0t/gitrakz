package mcp

import (
	"encoding/json"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
)

// formFieldDTO / stepDTO / blockDTO / templateDTO / eventDTO give gitrakz's
// domain types a flat, JSON-schema-friendly shape: they replace every
// json.RawMessage sub-field (which the MCP SDK's schema inference can't
// represent meaningfully) with a decoded map[string]any. This is the same
// translation internal/pkg/http/server's mapping.go performs for the
// generated HTTP API's types — the MCP tools need their own copy because
// the wire shape (flat JSON, not oapi-codegen's pointer-optional style)
// differs slightly.

type formFieldDTO struct {
	Key      string `json:"key"`
	Label    string `json:"label,omitempty"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Default  any    `json:"default,omitempty"`
}

type stepDTO struct {
	Name   string         `json:"name"`
	Params map[string]any `json:"params,omitempty"`
}

// blockDTO doubles as both a template.LayoutBlock (Source set, Data holding
// its decoded Params) and a rendered blocks.Block (Source always empty).
type blockDTO struct {
	Type   string         `json:"type"`
	Source string         `json:"source,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
}

type templateDTO struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Form        []formFieldDTO `json:"form,omitempty"`
	Transform   []stepDTO      `json:"transform,omitempty"`
	Layout      []blockDTO     `json:"layout,omitempty"`
	Exports     []string       `json:"exports,omitempty"`
	Model       string         `json:"model,omitempty"`
	Builtin     bool           `json:"builtin"`
}

type eventDTO struct {
	ID        string `json:"id"`
	TS        int64  `json:"ts"`
	Type      string `json:"type"`
	Owner     string `json:"owner"`
	Repo      string `json:"repo"`
	SHA       string `json:"sha,omitempty"`
	Number    int    `json:"number,omitempty"`
	Title     string `json:"title,omitempty"`
	URL       string `json:"url,omitempty"`
	Additions int    `json:"additions,omitempty"`
	Deletions int    `json:"deletions,omitempty"`
	Branch    string `json:"branch,omitempty"`
}

// rawToMap decodes a json.RawMessage object into a plain map, treating an
// empty raw value as "no data" rather than a decode error.
func rawToMap(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		//nolint:nilnil // absent data is a distinct, valid outcome
		return nil, nil
	}

	m := map[string]any{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, ctxerrors.Wrap(err, "unmarshal raw json")
	}

	return m, nil
}

func stepToDTO(step template.Step) (stepDTO, error) {
	params, err := rawToMap(step.Params)
	if err != nil {
		return stepDTO{}, err
	}

	return stepDTO{Name: step.Name, Params: params}, nil
}

func layoutBlockToDTO(lb template.LayoutBlock) (blockDTO, error) {
	data, err := rawToMap(lb.Params)
	if err != nil {
		return blockDTO{}, err
	}

	return blockDTO{Type: lb.Type, Source: lb.Source, Data: data}, nil
}

func formFieldToDTO(f template.FormField) formFieldDTO {
	return formFieldDTO{
		Key:      f.Key,
		Label:    f.Label,
		Type:     string(f.Type),
		Required: f.Required,
		Default:  f.Default,
	}
}

func templateToDTO(tmpl template.Template) (templateDTO, error) {
	form := make([]formFieldDTO, 0, len(tmpl.Form))
	for _, f := range tmpl.Form {
		form = append(form, formFieldToDTO(f))
	}

	transform := make([]stepDTO, 0, len(tmpl.Transform))

	for _, step := range tmpl.Transform {
		dto, err := stepToDTO(step)
		if err != nil {
			return templateDTO{}, err
		}

		transform = append(transform, dto)
	}

	layout := make([]blockDTO, 0, len(tmpl.Layout))

	for _, lb := range tmpl.Layout {
		dto, err := layoutBlockToDTO(lb)
		if err != nil {
			return templateDTO{}, err
		}

		layout = append(layout, dto)
	}

	return templateDTO{
		ID:          tmpl.ID,
		Name:        tmpl.Name,
		Description: tmpl.Description,
		Form:        form,
		Transform:   transform,
		Layout:      layout,
		Exports:     tmpl.Exports,
		Model:       tmpl.Model,
		Builtin:     tmpl.Builtin,
	}, nil
}

func templatesToDTO(tmpls []template.Template) ([]templateDTO, error) {
	out := make([]templateDTO, 0, len(tmpls))

	for _, tmpl := range tmpls {
		dto, err := templateToDTO(tmpl)
		if err != nil {
			return nil, err
		}

		out = append(out, dto)
	}

	return out, nil
}

func eventToDTO(ev types.Event) eventDTO {
	return eventDTO{
		ID:        ev.ID,
		TS:        ev.TS,
		Type:      string(ev.Type),
		Owner:     ev.Owner,
		Repo:      ev.Repo,
		SHA:       ev.SHA,
		Number:    ev.Number,
		Title:     ev.Title,
		URL:       ev.URL,
		Additions: ev.Additions,
		Deletions: ev.Deletions,
		Branch:    ev.Branch,
	}
}

func eventsToDTO(evs []types.Event) []eventDTO {
	out := make([]eventDTO, 0, len(evs))

	for _, ev := range evs {
		out = append(out, eventToDTO(ev))
	}

	return out
}

func blockToDTO(b blocks.Block) (blockDTO, error) {
	data, err := rawToMap(b.Data)
	if err != nil {
		return blockDTO{}, err
	}

	return blockDTO{Type: string(b.Type), Data: data}, nil
}

func documentToDTO(doc blocks.Document) ([]blockDTO, error) {
	out := make([]blockDTO, 0, len(doc))

	for _, b := range doc {
		dto, err := blockToDTO(b)
		if err != nil {
			return nil, err
		}

		out = append(out, dto)
	}

	return out, nil
}
