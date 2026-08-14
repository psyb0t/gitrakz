package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db/models"
	"github.com/psyb0t/gitrakz/internal/pkg/db/repositories"
	"gorm.io/gen"
	"gorm.io/gorm/clause"
)

// Store is the typed facade over gitrakz's SQLite storage. It maps between
// the domain types (types.Event, template.Template, ...) and the generated
// gorm-gen repositories, wraps every error with ctxerrors, and translates
// gorm's not-found result into either a zero value or commerr.ErrNotFound
// depending on what each method's caller needs.
type Store struct {
	query *repositories.Query
}

// TimelineFilter narrows QueryTimeline to a window of events. The zero
// value of every field means "no filter on that dimension" — From/To of 0
// disable the time bound, Owner/Repo/Type of "" disable that equality
// filter.
type TimelineFilter struct {
	Owner   string
	Repo    string
	Type    string
	From    int64
	To      int64
	Page    int
	PerPage int
}

// UpsertEvents inserts evs, updating any row whose id already exists. A nil
// or empty evs is a no-op.
func (s *Store) UpsertEvents(ctx context.Context, evs []types.Event) error {
	if len(evs) == 0 {
		return nil
	}

	rows := make([]*models.Event, 0, len(evs))
	for _, ev := range evs {
		rows = append(rows, eventToModel(ev))
	}

	err := s.query.Event.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).
		Create(rows...)
	if err != nil {
		return ctxerrors.Wrap(err, "upsert events")
	}

	return nil
}

// QueryTimeline returns the events matching f, newest first, one page at a
// time. hasMore reports whether a further page exists; when true, the last
// row of a full page has already been dropped from evs.
func (s *Store) QueryTimeline(
	ctx context.Context,
	f TimelineFilter,
) ([]types.Event, bool, error) {
	repo := s.query.Event

	//nolint:mnd // 5 optional filter dimensions
	wheres := make([]gen.Condition, 0, 5)
	if f.Owner != "" {
		wheres = append(wheres, repo.Owner.Eq(f.Owner))
	}

	if f.Repo != "" {
		wheres = append(wheres, repo.Repo.Eq(f.Repo))
	}

	if f.Type != "" {
		wheres = append(wheres, repo.Type.Eq(f.Type))
	}

	if f.From > 0 {
		wheres = append(wheres, repo.TS.Gte(f.From))
	}

	if f.To > 0 {
		wheres = append(wheres, repo.TS.Lte(f.To))
	}

	rows, err := repo.WithContext(ctx).
		Where(wheres...).
		Order(repo.TS.Desc()).
		Offset(f.Page * f.PerPage).
		Limit(f.PerPage + 1).
		Find()
	if err != nil {
		return nil, false, ctxerrors.Wrap(err, "query timeline")
	}

	hasMore := len(rows) == f.PerPage+1
	if hasMore {
		rows = rows[:f.PerPage]
	}

	evs := make([]types.Event, 0, len(rows))
	for _, row := range rows {
		evs = append(evs, eventFromModel(row))
	}

	return evs, hasMore, nil
}

// ListOwners returns every distinct owner with at least one stored event,
// sorted ascending.
func (s *Store) ListOwners(ctx context.Context) ([]string, error) {
	owners, err := s.query.Event.WithContext(ctx).ListOwners()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "list owners")
	}

	return owners, nil
}

// ListRepos returns every distinct repo owner has at least one stored event
// for, sorted ascending.
func (s *Store) ListRepos(ctx context.Context, owner string) ([]string, error) {
	repos, err := s.query.Event.WithContext(ctx).ListRepos(owner)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "list repos")
	}

	return repos, nil
}

// GetSyncState returns the last synced unix timestamp for (owner, repo), or
// 0 with a nil error if no sync has ever run for that pair.
func (s *Store) GetSyncState(
	ctx context.Context,
	owner, repo string,
) (int64, error) {
	r := s.query.SyncState

	row, err := r.WithContext(ctx).
		Where(r.Owner.Eq(owner), r.Repo.Eq(repo)).
		First()
	if err != nil {
		if isNotFound(err) {
			return 0, nil
		}

		return 0, ctxerrors.Wrap(err, "get sync state")
	}

	return row.LastSyncedTS, nil
}

// UpsertSyncState records ts as the last synced unix timestamp for (owner,
// repo), replacing any previous value.
func (s *Store) UpsertSyncState(
	ctx context.Context,
	owner, repo string,
	ts int64,
) error {
	err := s.query.SyncState.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "owner"}, {Name: "repo"}},
			UpdateAll: true,
		}).
		Create(&models.SyncState{
			Owner:        owner,
			Repo:         repo,
			LastSyncedTS: ts,
		})
	if err != nil {
		return ctxerrors.Wrap(err, "upsert sync state")
	}

	return nil
}

// ListTemplates returns every saved template.
func (s *Store) ListTemplates(
	ctx context.Context,
) ([]template.Template, error) {
	rows, err := s.query.Template.WithContext(ctx).Find()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "list templates")
	}

	templates := make([]template.Template, 0, len(rows))

	for _, row := range rows {
		tmpl, decodeErr := templateFromModel(row)
		if decodeErr != nil {
			return nil, ctxerrors.Wrap(decodeErr, "decode template")
		}

		templates = append(templates, tmpl)
	}

	return templates, nil
}

// GetTemplate returns the template with the given id, or wraps
// commerr.ErrNotFound if no template has that id.
func (s *Store) GetTemplate(
	ctx context.Context,
	id string,
) (template.Template, error) {
	r := s.query.Template

	row, err := r.WithContext(ctx).Where(r.ID.Eq(id)).First()
	if err != nil {
		if isNotFound(err) {
			return template.Template{}, ctxerrors.Wrap(
				commerr.ErrNotFound, "get template",
			)
		}

		return template.Template{}, ctxerrors.Wrap(err, "get template")
	}

	tmpl, err := templateFromModel(row)
	if err != nil {
		return template.Template{}, ctxerrors.Wrap(err, "decode template")
	}

	return tmpl, nil
}

// SaveTemplate creates tmpl, or replaces every column of the existing row
// with the same id.
func (s *Store) SaveTemplate(
	ctx context.Context,
	tmpl template.Template,
) error {
	row, err := templateToModel(tmpl)
	if err != nil {
		return ctxerrors.Wrap(err, "encode template")
	}

	err = s.query.Template.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).
		Create(row)
	if err != nil {
		return ctxerrors.Wrap(err, "save template")
	}

	return nil
}

// DeleteTemplate removes the template with the given id, if any.
func (s *Store) DeleteTemplate(ctx context.Context, id string) error {
	r := s.query.Template

	if _, err := r.WithContext(ctx).Where(r.ID.Eq(id)).Delete(); err != nil {
		return ctxerrors.Wrap(err, "delete template")
	}

	return nil
}

// SaveDocument persists one template run's output under id.
func (s *Store) SaveDocument(
	ctx context.Context,
	id, templateID, filter, formValues, doc string,
) error {
	err := s.query.Document.WithContext(ctx).Create(&models.Document{
		ID:         id,
		TemplateID: templateID,
		Filter:     filter,
		FormValues: formValues,
		Doc:        doc,
		CreatedTS:  time.Now().Unix(),
	})
	if err != nil {
		return ctxerrors.Wrap(err, "save document")
	}

	return nil
}

// LLMCacheGet returns the cached output for key, or ok=false with a nil
// error if key was never cached.
func (s *Store) LLMCacheGet(
	ctx context.Context,
	key string,
) (string, bool, error) {
	r := s.query.LLMCache

	row, err := r.WithContext(ctx).Where(r.Key.Eq(key)).First()
	if err != nil {
		if isNotFound(err) {
			return "", false, nil
		}

		return "", false, ctxerrors.Wrap(err, "get llm cache entry")
	}

	return row.Output, true, nil
}

// LLMCachePut stores output under key, replacing any previous entry for
// that key.
func (s *Store) LLMCachePut(
	ctx context.Context,
	key, step, processingVersion, inputHash, output string,
) error {
	err := s.query.LLMCache.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			UpdateAll: true,
		}).
		Create(&models.LLMCache{
			Key:               key,
			Step:              step,
			ProcessingVersion: processingVersion,
			InputHash:         inputHash,
			Output:            output,
			CreatedTS:         time.Now().Unix(),
		})
	if err != nil {
		return ctxerrors.Wrap(err, "put llm cache entry")
	}

	return nil
}

func eventToModel(ev types.Event) *models.Event {
	return &models.Event{
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
		Raw:       string(ev.Raw),
	}
}

func eventFromModel(row *models.Event) types.Event {
	var raw json.RawMessage
	if row.Raw != "" {
		raw = json.RawMessage(row.Raw)
	}

	return types.Event{
		ID:        row.ID,
		TS:        row.TS,
		Type:      types.EventType(row.Type),
		Owner:     row.Owner,
		Repo:      row.Repo,
		SHA:       row.SHA,
		Number:    row.Number,
		Title:     row.Title,
		URL:       row.URL,
		Additions: row.Additions,
		Deletions: row.Deletions,
		Branch:    row.Branch,
		Raw:       raw,
	}
}

// templateToModel JSON-encodes tmpl's sub-structures into the templates
// table's TEXT columns. CreatedTS is stamped at save time — template.Template
// carries no creation timestamp of its own, so every save (insert or
// OnConflict update) refreshes it.
func templateToModel(tmpl template.Template) (*models.Template, error) {
	formJSON, err := json.Marshal(tmpl.Form)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "marshal form")
	}

	transformJSON, err := json.Marshal(tmpl.Transform)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "marshal transform")
	}

	layoutJSON, err := json.Marshal(tmpl.Layout)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "marshal layout")
	}

	exportsJSON, err := json.Marshal(tmpl.Exports)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "marshal exports")
	}

	return &models.Template{
		ID:          tmpl.ID,
		Name:        tmpl.Name,
		Description: tmpl.Description,
		Form:        string(formJSON),
		Transform:   string(transformJSON),
		Layout:      string(layoutJSON),
		Exports:     string(exportsJSON),
		Model:       tmpl.Model,
		Builtin:     tmpl.Builtin,
		CreatedTS:   time.Now().Unix(),
	}, nil
}

func templateFromModel(row *models.Template) (template.Template, error) {
	tmpl := template.Template{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Model:       row.Model,
		Builtin:     row.Builtin,
	}

	if err := decodeJSONText(row.Form, &tmpl.Form); err != nil {
		return template.Template{}, ctxerrors.Wrap(err, "unmarshal form")
	}

	if err := decodeJSONText(row.Transform, &tmpl.Transform); err != nil {
		return template.Template{}, ctxerrors.Wrap(err, "unmarshal transform")
	}

	if err := decodeJSONText(row.Layout, &tmpl.Layout); err != nil {
		return template.Template{}, ctxerrors.Wrap(err, "unmarshal layout")
	}

	if err := decodeJSONText(row.Exports, &tmpl.Exports); err != nil {
		return template.Template{}, ctxerrors.Wrap(err, "unmarshal exports")
	}

	return tmpl, nil
}

// decodeJSONText unmarshals raw into out, treating an empty string as
// "leave out at its zero value" instead of a JSON syntax error.
func decodeJSONText(raw string, out any) error {
	if raw == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return ctxerrors.Wrap(err, "unmarshal json text")
	}

	return nil
}
