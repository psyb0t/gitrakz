package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleDoc() blocks.Document {
	return blocks.Document{
		blocks.NewHeading(1, "Weekly Report"),
		blocks.NewTable(
			[]string{"Day", "Hours"},
			[][]string{
				{"Mon", "8"},
				{"Tue", "6"},
			},
			nil,
		),
		blocks.NewMetric("Total", "14", "h"),
		blocks.NewList(false, []string{"item one", "item two"}),
	}
}

// fullDoc returns a document covering every block type export knows about
// (heading, text, table w/ footer, keyvalue, metric, code, chart, list),
// including an ordered list and a chart with fewer Values than Labels.
func fullDoc() blocks.Document {
	return blocks.Document{
		blocks.NewHeading(1, "Top Heading"),
		blocks.NewHeading(2, "Section Heading"),
		blocks.NewHeading(3, "Sub Heading"),
		blocks.NewText("**bold** markdown text"),
		blocks.NewTable(
			[]string{"Day", "Hours"},
			[][]string{
				{"Mon", "8"},
				{"Tue", "6"},
			},
			[]string{"Total", "14"},
		),
		blocks.NewKeyValue([]blocks.KVPair{
			{Key: "owner", Value: "psyb0t"},
			{Key: "repo", Value: "gitrakz"},
		}),
		blocks.NewMetric("Total", "14", "h"),
		blocks.NewCode("go", `fmt.Println("hi")`),
		blocks.NewChart(
			"bar",
			[]string{"Mon", "Tue", "Wed"},
			[]float64{1, 2},
		),
		blocks.NewList(false, []string{"item one", "item two"}),
		blocks.NewList(true, []string{"first", "second"}),
	}
}

func TestToJSON_RoundTrip(t *testing.T) {
	t.Parallel()

	doc := sampleDoc()

	data, err := ToJSON(doc)
	require.NoError(t, err)

	var got blocks.Document
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, doc, got)
}

func TestToCSV_ContainsTableAndMetricCells(t *testing.T) {
	t.Parallel()

	doc := sampleDoc()

	data, err := ToCSV(doc)
	require.NoError(t, err)

	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1

	records, err := r.ReadAll()
	require.NoError(t, err)

	// Heading/text/code carry no tabular data and are skipped, so the CSV
	// starts with the table header, followed by both data rows, the metric
	// label/value row, then one row per list item.
	require.Len(t, records, 6)
	assert.Equal(t, []string{"Day", "Hours"}, records[0])
	assert.Equal(t, []string{"Mon", "8"}, records[1])
	assert.Equal(t, []string{"Tue", "6"}, records[2])
	assert.Equal(t, []string{"Total", "14"}, records[3])
	assert.Equal(t, []string{"item one"}, records[4])
	assert.Equal(t, []string{"item two"}, records[5])
}

func TestToCSV_UnknownBlockType(t *testing.T) {
	t.Parallel()

	doc := blocks.Document{
		{Type: "bogus", Data: json.RawMessage(`{}`)},
	}

	_, err := ToCSV(doc)
	require.ErrorIs(t, err, ErrUnknownBlockType)
}

func TestToPDF_ReturnsValidPDFBytes(t *testing.T) {
	t.Parallel()

	doc := sampleDoc()

	data, err := ToPDF(doc)
	require.NoError(t, err)
	require.NotEmpty(t, data)
	assert.True(t, bytes.HasPrefix(data, []byte("%PDF")))
}

func TestToPDF_UnknownBlockType(t *testing.T) {
	t.Parallel()

	doc := blocks.Document{
		{Type: "bogus", Data: json.RawMessage(`{}`)},
	}

	_, err := ToPDF(doc)
	require.ErrorIs(t, err, ErrUnknownBlockType)
}

func TestToJSON_EmptyDocument(t *testing.T) {
	t.Parallel()

	data, err := ToJSON(blocks.Document{})
	require.NoError(t, err)
	assert.JSONEq(t, "[]", string(data))
}

func TestToCSV_AllBlockTypes(t *testing.T) {
	t.Parallel()

	data, err := ToCSV(fullDoc())
	require.NoError(t, err)

	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1

	records, err := r.ReadAll()
	require.NoError(t, err)

	// Heading/text/code carry no tabular data and are skipped. The chart's
	// third label ("Wed") has no matching value, so writeChartCSV falls
	// back to an empty string for it.
	want := [][]string{
		{"Day", "Hours"},
		{"Mon", "8"},
		{"Tue", "6"},
		{"Total", "14"},
		{"owner", "psyb0t"},
		{"repo", "gitrakz"},
		{"Total", "14"},
		{"Mon", "1"},
		{"Tue", "2"},
		{"Wed", ""},
		{"item one"},
		{"item two"},
		{"first"},
		{"second"},
	}
	assert.Equal(t, want, records)
}

func TestToPDF_AllBlockTypes(t *testing.T) {
	t.Parallel()

	// fullDoc covers heading levels 1/2/3 (headingFontSize's three
	// branches), text, table w/ footer, keyvalue, metric, code, chart, and
	// both an unordered and an ordered list — every blockToPDFRows case.
	data, err := ToPDF(fullDoc())
	require.NoError(t, err)
	require.NotEmpty(t, data)
	assert.True(t, bytes.HasPrefix(data, []byte("%PDF")))
}

// TestToCSV_BlockDecodeErrors covers every tabular block type's "decode
// block" error branch: Type matches (so writeBlockCSV dispatches to the
// matching writer) but Data is malformed JSON, so the As* accessor's
// json.Unmarshal fails. Heading/text/code are skipped by writeBlockCSV
// before decoding, so they have no decode-error branch to cover here.
func TestToCSV_BlockDecodeErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		blockType blocks.BlockType
	}{
		{"table", blocks.BlockTypeTable},
		{"list", blocks.BlockTypeList},
		{"keyvalue", blocks.BlockTypeKeyValue},
		{"metric", blocks.BlockTypeMetric},
		{"chart", blocks.BlockTypeChart},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc := blocks.Document{
				{Type: tc.blockType, Data: json.RawMessage(`{`)},
			}

			_, err := ToCSV(doc)
			require.Error(t, err)
		})
	}
}

// TestToPDF_BlockDecodeErrors covers every block type's "decode block"
// error branch in blockToPDFRows: Type matches but Data is malformed JSON.
func TestToPDF_BlockDecodeErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		blockType blocks.BlockType
	}{
		{"heading", blocks.BlockTypeHeading},
		{"text", blocks.BlockTypeText},
		{"list", blocks.BlockTypeList},
		{"table", blocks.BlockTypeTable},
		{"keyvalue", blocks.BlockTypeKeyValue},
		{"metric", blocks.BlockTypeMetric},
		{"code", blocks.BlockTypeCode},
		{"chart", blocks.BlockTypeChart},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc := blocks.Document{
				{Type: tc.blockType, Data: json.RawMessage(`{`)},
			}

			_, err := ToPDF(doc)
			require.Error(t, err)
		})
	}
}

func TestGridColumnSizes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		n    int
		want []int
	}{
		{"zero columns", 0, nil},
		{"one column takes the whole grid", 1, []int{12}},
		{"even split", 3, []int{4, 4, 4}},
		{
			"more columns than grid size clamps the last to 1",
			15,
			[]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, gridColumnSizes(tc.n))
		})
	}
}
