// Package blocks defines the fixed library of display building blocks used
// to compose a template's rendered layout. A Block is a discriminated
// union: Type selects the payload shape and Data holds the matching
// json.RawMessage, produced by a New* constructor and read back with the
// matching As* accessor.
package blocks

import (
	"encoding/json"

	"github.com/psyb0t/ctxerrors"
)

// BlockType identifies which payload shape a Block's Data holds.
type BlockType string

const (
	BlockTypeHeading  BlockType = "heading"
	BlockTypeText     BlockType = "text"
	BlockTypeList     BlockType = "list"
	BlockTypeTable    BlockType = "table"
	BlockTypeKeyValue BlockType = "keyvalue"
	BlockTypeMetric   BlockType = "metric"
	BlockTypeCode     BlockType = "code"
	BlockTypeChart    BlockType = "chart"
)

type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

type Text struct {
	Markdown string `json:"markdown"`
}

type List struct {
	Ordered bool     `json:"ordered"`
	Items   []string `json:"items"`
}

type KVPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type KeyValue struct {
	Pairs []KVPair `json:"pairs"`
}

type Table struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
	Footer  []string   `json:"footer"`
}

type Metric struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Unit  string `json:"unit"`
}

type Code struct {
	Lang    string `json:"lang"`
	Content string `json:"content"`
}

type Chart struct {
	Kind   string    `json:"kind"`
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
}

// Block is a discriminated union over the display building blocks. Data
// holds the payload matching Type, as json.RawMessage.
type Block struct {
	Type BlockType       `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Document is an ordered sequence of Blocks composing a rendered layout.
type Document = []Block

// marshalPayload encodes payload into json.RawMessage for a Block's Data.
func marshalPayload(payload any) json.RawMessage {
	data, err := json.Marshal(payload)
	if err != nil {
		// Every payload type in this package is a plain struct of
		// string/int/bool/slice fields, so json.Marshal cannot fail
		// for them. A panic here means a payload type was extended
		// with something non-marshalable — a programming error, not
		// a runtime condition callers should handle.
		panic(ctxerrors.Wrap(err, "marshal block payload"))
	}

	return data
}

func newBlock(blockType BlockType, payload any) Block {
	return Block{
		Type: blockType,
		Data: marshalPayload(payload),
	}
}

func NewHeading(level int, text string) Block {
	return newBlock(BlockTypeHeading, Heading{Level: level, Text: text})
}

func NewText(markdown string) Block {
	return newBlock(BlockTypeText, Text{Markdown: markdown})
}

func NewList(ordered bool, items []string) Block {
	return newBlock(BlockTypeList, List{Ordered: ordered, Items: items})
}

func NewTable(
	columns []string,
	rows [][]string,
	footer []string,
) Block {
	return newBlock(BlockTypeTable, Table{
		Columns: columns,
		Rows:    rows,
		Footer:  footer,
	})
}

func NewKeyValue(pairs []KVPair) Block {
	return newBlock(BlockTypeKeyValue, KeyValue{Pairs: pairs})
}

func NewMetric(label, value, unit string) Block {
	return newBlock(BlockTypeMetric, Metric{
		Label: label,
		Value: value,
		Unit:  unit,
	})
}

func NewCode(lang, content string) Block {
	return newBlock(BlockTypeCode, Code{Lang: lang, Content: content})
}

func NewChart(
	kind string,
	labels []string,
	values []float64,
) Block {
	return newBlock(BlockTypeChart, Chart{
		Kind:   kind,
		Labels: labels,
		Values: values,
	})
}

// unmarshalPayload decodes block.Data into T after checking block.Type
// matches want, wrapping ErrBlockTypeMismatch on a type mismatch.
//
// struct at every call site (Heading, Table, ...), never a real interface.
//

func unmarshalPayload[T any](block Block, want BlockType) (T, error) {
	var payload T

	if block.Type != want {
		return payload, ctxerrors.Wrapf(
			ErrBlockTypeMismatch,
			"expected type %q, got %q",
			want,
			block.Type,
		)
	}

	if err := json.Unmarshal(block.Data, &payload); err != nil {
		return payload, ctxerrors.Wrap(err, "unmarshal block data")
	}

	return payload, nil
}

func (b Block) AsHeading() (Heading, error) {
	return unmarshalPayload[Heading](b, BlockTypeHeading)
}

func (b Block) AsText() (Text, error) {
	return unmarshalPayload[Text](b, BlockTypeText)
}

func (b Block) AsList() (List, error) {
	return unmarshalPayload[List](b, BlockTypeList)
}

func (b Block) AsTable() (Table, error) {
	return unmarshalPayload[Table](b, BlockTypeTable)
}

func (b Block) AsKeyValue() (KeyValue, error) {
	return unmarshalPayload[KeyValue](b, BlockTypeKeyValue)
}

func (b Block) AsMetric() (Metric, error) {
	return unmarshalPayload[Metric](b, BlockTypeMetric)
}

func (b Block) AsCode() (Code, error) {
	return unmarshalPayload[Code](b, BlockTypeCode)
}

func (b Block) AsChart() (Chart, error) {
	return unmarshalPayload[Chart](b, BlockTypeChart)
}
