package export

import (
	"fmt"
	"strconv"

	"github.com/johnfercher/maroto/v2"
	rowcomponent "github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
)

const (
	pdfGridSize         = 12
	pdfKeyValueColSize  = pdfGridSize / 2
	pdfHeaderFooterRows = 2 // header row + optional footer row

	pdfTableRowHeight    = 7.0
	pdfKeyValueRowHeight = 7.0

	pdfHeadingFontSizeLevel1 = 20.0
	pdfHeadingFontSizeLevel2 = 16.0
	pdfHeadingFontSizeLevel3 = 13.0
	pdfMetricFontSize        = 20.0

	pdfListBullet = "- "

	headingLevelTop     = 1
	headingLevelSection = 2
)

// ToPDF renders every block in doc, in document order, into a single
// paginated PDF via maroto.
func ToPDF(doc blocks.Document) ([]byte, error) {
	m := maroto.New()

	for _, block := range doc {
		rows, err := blockToPDFRows(block)
		if err != nil {
			return nil, ctxerrors.Wrapf(
				err, "render block %q to pdf", block.Type,
			)
		}

		m.AddRows(rows...)
	}

	document, err := m.Generate()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "generate pdf")
	}

	return document.GetBytes(), nil
}

func blockToPDFRows(block blocks.Block) ([]core.Row, error) {
	switch block.Type {
	case blocks.BlockTypeHeading:
		return headingToPDFRows(block)
	case blocks.BlockTypeText:
		return textToPDFRows(block)
	case blocks.BlockTypeList:
		return listToPDFRows(block)
	case blocks.BlockTypeTable:
		return tableToPDFRows(block)
	case blocks.BlockTypeKeyValue:
		return keyValueToPDFRows(block)
	case blocks.BlockTypeMetric:
		return metricToPDFRows(block)
	case blocks.BlockTypeCode:
		return codeToPDFRows(block)
	case blocks.BlockTypeChart:
		return chartToPDFRows(block)
	}

	return nil, ctxerrors.Wrapf(ErrUnknownBlockType, "type %q", block.Type)
}

func headingToPDFRows(block blocks.Block) ([]core.Row, error) {
	heading, err := block.AsHeading()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "decode heading block")
	}

	return []core.Row{
		text.NewAutoRow(heading.Text, props.Text{
			Style: fontstyle.Bold,
			Size:  headingFontSize(heading.Level),
		}),
	}, nil
}

func headingFontSize(level int) float64 {
	switch {
	case level <= headingLevelTop:
		return pdfHeadingFontSizeLevel1
	case level == headingLevelSection:
		return pdfHeadingFontSizeLevel2
	default:
		return pdfHeadingFontSizeLevel3
	}
}

func textToPDFRows(block blocks.Block) ([]core.Row, error) {
	t, err := block.AsText()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "decode text block")
	}

	return []core.Row{text.NewAutoRow(t.Markdown)}, nil
}

func listToPDFRows(block blocks.Block) ([]core.Row, error) {
	list, err := block.AsList()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "decode list block")
	}

	rows := make([]core.Row, 0, len(list.Items))

	for i, item := range list.Items {
		prefix := pdfListBullet
		if list.Ordered {
			prefix = strconv.Itoa(i+1) + ". "
		}

		rows = append(rows, text.NewAutoRow(prefix+item))
	}

	return rows, nil
}

func tableToPDFRows(block blocks.Block) ([]core.Row, error) {
	table, err := block.AsTable()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "decode table block")
	}

	colSizes := gridColumnSizes(len(table.Columns))

	rows := make([]core.Row, 0, len(table.Rows)+pdfHeaderFooterRows)
	rows = append(rows, tableGridRow(table.Columns, colSizes, true))

	for _, dataRow := range table.Rows {
		rows = append(rows, tableGridRow(dataRow, colSizes, false))
	}

	if len(table.Footer) > 0 {
		rows = append(rows, tableGridRow(table.Footer, colSizes, true))
	}

	return rows, nil
}

// gridColumnSizes divides maroto's pdfGridSize-wide grid evenly across n
// columns, handing any remainder to the last column so the sizes always sum
// to pdfGridSize.
func gridColumnSizes(n int) []int {
	if n == 0 {
		return nil
	}

	base := max(pdfGridSize/n, 1)

	sizes := make([]int, n)

	used := 0

	for i := range sizes {
		sizes[i] = base
		used += base
	}

	sizes[n-1] += pdfGridSize - used
	if sizes[n-1] < 1 {
		sizes[n-1] = 1
	}

	return sizes
}

// tableGridRow returns a maroto core.Row — the type maroto's own AddRows /
// row.Row.Add API requires; there is no concrete struct to return instead.
//

func tableGridRow(cells []string, colSizes []int, bold bool) core.Row {
	textProps := props.Text{}
	if bold {
		textProps.Style = fontstyle.Bold
	}

	cellStyle := &props.Cell{BorderType: border.Full}

	cols := make([]core.Col, 0, len(cells))

	for i, cellValue := range cells {
		colSize := 1
		if i < len(colSizes) {
			colSize = colSizes[i]
		}

		col := text.NewCol(colSize, cellValue, textProps)
		cols = append(cols, col.WithStyle(cellStyle))
	}

	return rowcomponent.New(pdfTableRowHeight).Add(cols...)
}

func keyValueToPDFRows(block blocks.Block) ([]core.Row, error) {
	kv, err := block.AsKeyValue()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "decode keyvalue block")
	}

	rows := make([]core.Row, 0, len(kv.Pairs))

	for _, pair := range kv.Pairs {
		keyCol := text.NewCol(
			pdfKeyValueColSize, pair.Key, props.Text{Style: fontstyle.Bold},
		)
		valueCol := text.NewCol(pdfKeyValueColSize, pair.Value, props.Text{})

		rows = append(
			rows, rowcomponent.New(pdfKeyValueRowHeight).Add(keyCol, valueCol),
		)
	}

	return rows, nil
}

func metricToPDFRows(block blocks.Block) ([]core.Row, error) {
	metric, err := block.AsMetric()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "decode metric block")
	}

	value := metric.Value
	if metric.Unit != "" {
		value = metric.Value + " " + metric.Unit
	}

	line := metric.Label + ": " + value

	return []core.Row{
		text.NewAutoRow(line, props.Text{
			Style: fontstyle.Bold,
			Size:  pdfMetricFontSize,
		}),
	}, nil
}

func codeToPDFRows(block blocks.Block) ([]core.Row, error) {
	code, err := block.AsCode()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "decode code block")
	}

	return []core.Row{
		text.NewAutoRow(code.Content, props.Text{Family: fontfamily.Courier}),
	}, nil
}

func chartToPDFRows(block blocks.Block) ([]core.Row, error) {
	chart, err := block.AsChart()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "decode chart block")
	}

	summary := fmt.Sprintf(
		"[chart: %s, %d data points]", chart.Kind, len(chart.Values),
	)

	return []core.Row{
		text.NewAutoRow(summary, props.Text{Style: fontstyle.Italic}),
	}, nil
}
