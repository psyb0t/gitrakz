package models

// LLMSettings is the single-row runtime LLM configuration: the model the
// summarization and composer blocks use, plus the reasoning effort and
// temperature applied when the model supports them.
type LLMSettings struct {
	ID              string
	Model           string
	ReasoningEffort string
	Temperature     float64
	UpdatedTS       int64
}

// TableName is the llm_settings table.
func (LLMSettings) TableName() string {
	return "llm_settings"
}
