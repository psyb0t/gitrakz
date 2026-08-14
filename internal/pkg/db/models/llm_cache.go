package models

// LLMCache is one cached LLM-step output, keyed by (step, processing_version,
// input_hash) so an unchanged input+version never re-hits the model.
type LLMCache struct {
	Key               string
	Step              string
	ProcessingVersion string
	InputHash         string
	Output            string
	CreatedTS         int64
}

// TableName is the llm_cache table.
func (LLMCache) TableName() string {
	return "llm_cache"
}
