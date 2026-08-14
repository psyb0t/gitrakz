package blocks

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentMarshalRoundTrip(t *testing.T) {
	heading := NewHeading(1, "Report")
	table := NewTable(
		[]string{"Day", "Hours"},
		[][]string{{"Mon", "8"}, {"Tue", "6"}},
		[]string{"", "14"},
	)
	metric := NewMetric("Total Hours", "14", "h")
	code := NewCode("go", `fmt.Println("hi")`)

	original := Document{heading, table, metric, code}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Document

	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Len(t, decoded, len(original))

	assert.Equal(t, BlockTypeHeading, decoded[0].Type)
	assert.Equal(t, BlockTypeTable, decoded[1].Type)
	assert.Equal(t, BlockTypeMetric, decoded[2].Type)
	assert.Equal(t, BlockTypeCode, decoded[3].Type)

	gotHeading, err := decoded[0].AsHeading()
	require.NoError(t, err)
	assert.Equal(t, Heading{Level: 1, Text: "Report"}, gotHeading)

	gotTable, err := decoded[1].AsTable()
	require.NoError(t, err)
	assert.Equal(t, Table{
		Columns: []string{"Day", "Hours"},
		Rows:    [][]string{{"Mon", "8"}, {"Tue", "6"}},
		Footer:  []string{"", "14"},
	}, gotTable)

	gotMetric, err := decoded[2].AsMetric()
	require.NoError(t, err)
	assert.Equal(t, Metric{
		Label: "Total Hours",
		Value: "14",
		Unit:  "h",
	}, gotMetric)

	gotCode, err := decoded[3].AsCode()
	require.NoError(t, err)
	assert.Equal(t, Code{Lang: "go", Content: `fmt.Println("hi")`}, gotCode)

	_, err = decoded[0].AsTable()
	assert.ErrorIs(t, err, ErrBlockTypeMismatch)

	_, err = decoded[1].AsCode()
	assert.ErrorIs(t, err, ErrBlockTypeMismatch)

	_, err = decoded[2].AsHeading()
	assert.ErrorIs(t, err, ErrBlockTypeMismatch)

	_, err = decoded[3].AsMetric()
	assert.ErrorIs(t, err, ErrBlockTypeMismatch)
}

func TestBlockConstructorsAndAccessors(t *testing.T) {
	testCases := []struct {
		name     string
		block    Block
		wantType BlockType
		assertAs func(t *testing.T, b Block)
	}{
		{
			name:     "heading",
			block:    NewHeading(1, "Report"),
			wantType: BlockTypeHeading,
			assertAs: func(t *testing.T, b Block) {
				t.Helper()

				got, err := b.AsHeading()
				require.NoError(t, err)
				assert.Equal(t, Heading{Level: 1, Text: "Report"}, got)
			},
		},
		{
			name:     "text",
			block:    NewText("**bold**"),
			wantType: BlockTypeText,
			assertAs: func(t *testing.T, b Block) {
				t.Helper()

				got, err := b.AsText()
				require.NoError(t, err)
				assert.Equal(t, Text{Markdown: "**bold**"}, got)
			},
		},
		{
			name:     "list",
			block:    NewList(true, []string{"one", "two"}),
			wantType: BlockTypeList,
			assertAs: func(t *testing.T, b Block) {
				t.Helper()

				got, err := b.AsList()
				require.NoError(t, err)
				assert.Equal(t, List{
					Ordered: true,
					Items:   []string{"one", "two"},
				}, got)
			},
		},
		{
			name: "keyvalue",
			block: NewKeyValue([]KVPair{
				{Key: "owner", Value: "psyb0t"},
			}),
			wantType: BlockTypeKeyValue,
			assertAs: func(t *testing.T, b Block) {
				t.Helper()

				got, err := b.AsKeyValue()
				require.NoError(t, err)
				assert.Equal(t, KeyValue{
					Pairs: []KVPair{{Key: "owner", Value: "psyb0t"}},
				}, got)
			},
		},
		{
			name:     "chart",
			block:    NewChart("bar", []string{"Mon", "Tue"}, []float64{1, 2}),
			wantType: BlockTypeChart,
			assertAs: func(t *testing.T, b Block) {
				t.Helper()

				got, err := b.AsChart()
				require.NoError(t, err)
				assert.Equal(t, Chart{
					Kind:   "bar",
					Labels: []string{"Mon", "Tue"},
					Values: []float64{1, 2},
				}, got)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantType, tc.block.Type)
			tc.assertAs(t, tc.block)
		})
	}
}

func TestBlockAccessorTypeMismatch(t *testing.T) {
	heading := NewHeading(1, "Report")

	_, err := heading.AsTable()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBlockTypeMismatch)
}
