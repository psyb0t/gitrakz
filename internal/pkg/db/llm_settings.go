package db

import (
	"context"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/db/models"
	"gorm.io/gorm/clause"
)

// llmSettingsID is the fixed primary key of the single settings row.
const llmSettingsID = "default"

// LLMSettings is the stored runtime LLM configuration read by the
// summarization and composer blocks.
type LLMSettings struct {
	Model           string
	ReasoningEffort string
	Temperature     float64
}

// GetLLMSettings returns the stored LLM settings, or a zero value when none
// have been saved yet.
func (s *Store) GetLLMSettings(ctx context.Context) (LLMSettings, error) {
	r := s.query.LLMSettings

	row, err := r.WithContext(ctx).Where(r.ID.Eq(llmSettingsID)).First()
	if err != nil {
		if isNotFound(err) {
			return LLMSettings{}, nil
		}

		return LLMSettings{}, ctxerrors.Wrap(err, "get llm settings")
	}

	return LLMSettings{
		Model:           row.Model,
		ReasoningEffort: row.ReasoningEffort,
		Temperature:     row.Temperature,
	}, nil
}

// SaveLLMSettings replaces the stored LLM settings.
func (s *Store) SaveLLMSettings(
	ctx context.Context,
	settings LLMSettings,
) error {
	err := s.query.LLMSettings.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).
		Create(&models.LLMSettings{
			ID:              llmSettingsID,
			Model:           settings.Model,
			ReasoningEffort: settings.ReasoningEffort,
			Temperature:     settings.Temperature,
			UpdatedTS:       time.Now().Unix(),
		})
	if err != nil {
		return ctxerrors.Wrap(err, "save llm settings")
	}

	return nil
}
