package engine

import (
	"encoding/json"
	"sort"
	"strconv"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
)

const (
	layoutTypeTable   = "table"
	layoutTypeMetric  = "metric"
	layoutTypeHeading = "heading"
	layoutTypeText    = "text"

	tableKeyColumn = "key"

	floatFormat    = 'f'
	floatPrecision = -1
	floatBitSize   = 64
)

// headingParams is the JSON shape of a "heading" layout block's Params.
type headingParams struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

// textParams is the JSON shape of a "text" layout block's Params.
type textParams struct {
	Markdown string `json:"markdown"`
}

// metricParams is the JSON shape of a "metric" layout block's Params.
type metricParams struct {
	Label string `json:"label"`
	Unit  string `json:"unit"`
}

// applyLayout renders every LayoutBlock in layout against the final
// pipeline state s, producing the display Document in layout order.
func applyLayout(
	layout []template.LayoutBlock,
	s *transform.State,
) (blocks.Document, error) {
	doc := make(blocks.Document, 0, len(layout))

	for _, block := range layout {
		rendered, err := renderLayoutBlock(block, s)
		if err != nil {
			return nil, ctxerrors.Wrapf(err, "layout block type %q", block.Type)
		}

		doc = append(doc, rendered)
	}

	return doc, nil
}

// renderLayoutBlock dispatches block to the renderer matching its Type,
// wrapping ErrUnknownLayoutType for any other Type.
func renderLayoutBlock(
	block template.LayoutBlock,
	s *transform.State,
) (blocks.Block, error) {
	switch block.Type {
	case layoutTypeTable:
		return renderTable(s), nil
	case layoutTypeMetric:
		return renderMetric(block, s)
	case layoutTypeHeading:
		return renderHeading(block)
	case layoutTypeText:
		return renderText(block)
	default:
		return blocks.Block{}, ctxerrors.Wrapf(
			ErrUnknownLayoutType, "type %q", block.Type,
		)
	}
}

// renderTable builds a blocks.Table from s.Rows: columns are ["key"] +
// the sorted union of every row's Values keys + the sorted union of every
// row's Labels keys, one output row per s.Rows entry.
func renderTable(s *transform.State) blocks.Block {
	valueKeys, labelKeys := rowKeyUnions(s.Rows)

	columns := make([]string, 0, 1+len(valueKeys)+len(labelKeys))
	columns = append(columns, tableKeyColumn)
	columns = append(columns, valueKeys...)
	columns = append(columns, labelKeys...)

	rows := make([][]string, 0, len(s.Rows))

	for _, row := range s.Rows {
		cells := make([]string, 0, len(columns))
		cells = append(cells, row.Key)

		for _, key := range valueKeys {
			cells = append(cells, formatFloat(row.Values[key]))
		}

		for _, key := range labelKeys {
			cells = append(cells, row.Labels[key])
		}

		rows = append(rows, cells)
	}

	return blocks.NewTable(columns, rows, nil)
}

// rowKeyUnions returns the sorted union of Values keys and the sorted
// union of Labels keys across every row in rows.
func rowKeyUnions(rows []transform.Row) ([]string, []string) {
	valueSet := map[string]struct{}{}
	labelSet := map[string]struct{}{}

	for _, row := range rows {
		for key := range row.Values {
			valueSet[key] = struct{}{}
		}

		for key := range row.Labels {
			labelSet[key] = struct{}{}
		}
	}

	return sortedSetKeys(valueSet), sortedSetKeys(labelSet)
}

// sortedSetKeys returns set's keys sorted ascending.
func sortedSetKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

// renderMetric builds a blocks.Metric summing block.Source across every
// row's Values in s.Rows. The label comes from params.Label, falling back
// to block.Source.
func renderMetric(
	block template.LayoutBlock,
	s *transform.State,
) (blocks.Block, error) {
	var params metricParams

	if len(block.Params) > 0 {
		if err := json.Unmarshal(block.Params, &params); err != nil {
			return blocks.Block{}, ctxerrors.Wrap(
				err, "unmarshal metric params",
			)
		}
	}

	label := params.Label
	if label == "" {
		label = block.Source
	}

	var sum float64

	for _, row := range s.Rows {
		sum += row.Values[block.Source]
	}

	return blocks.NewMetric(label, formatFloat(sum), params.Unit), nil
}

// renderHeading builds a blocks.Heading from block.Params {level,text}.
func renderHeading(block template.LayoutBlock) (blocks.Block, error) {
	var params headingParams

	if len(block.Params) > 0 {
		if err := json.Unmarshal(block.Params, &params); err != nil {
			return blocks.Block{}, ctxerrors.Wrap(
				err, "unmarshal heading params",
			)
		}
	}

	return blocks.NewHeading(params.Level, params.Text), nil
}

// renderText builds a blocks.Text from block.Params {markdown}.
func renderText(block template.LayoutBlock) (blocks.Block, error) {
	var params textParams

	if len(block.Params) > 0 {
		if err := json.Unmarshal(block.Params, &params); err != nil {
			return blocks.Block{}, ctxerrors.Wrap(err, "unmarshal text params")
		}
	}

	return blocks.NewText(params.Markdown), nil
}

// formatFloat renders v as its shortest decimal string representation.
func formatFloat(v float64) string {
	return strconv.FormatFloat(v, floatFormat, floatPrecision, floatBitSize)
}
