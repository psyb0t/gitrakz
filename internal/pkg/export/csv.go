package export

import (
	"bytes"
	"encoding/csv"
	"strconv"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
)

// ToCSV walks doc and serializes every tabular block (table, list, keyvalue,
// metric, chart) into a single CSV document. Prose blocks (heading, text,
// code) carry no tabular data and are skipped. Records vary in width across
// block types, so a caller reading the result back with encoding/csv must
// set csv.Reader.FieldsPerRecord = -1.
func ToCSV(doc blocks.Document) ([]byte, error) {
	var buf bytes.Buffer

	w := csv.NewWriter(&buf)

	for _, block := range doc {
		if err := writeBlockCSV(w, block); err != nil {
			return nil, ctxerrors.Wrapf(
				err, "write block %q to csv", block.Type,
			)
		}
	}

	w.Flush()

	if err := w.Error(); err != nil {
		return nil, ctxerrors.Wrap(err, "flush csv writer")
	}

	return buf.Bytes(), nil
}

func writeBlockCSV(w *csv.Writer, block blocks.Block) error {
	switch block.Type {
	case blocks.BlockTypeHeading, blocks.BlockTypeText, blocks.BlockTypeCode:
		return nil
	case blocks.BlockTypeTable:
		return writeTableCSV(w, block)
	case blocks.BlockTypeList:
		return writeListCSV(w, block)
	case blocks.BlockTypeKeyValue:
		return writeKeyValueCSV(w, block)
	case blocks.BlockTypeMetric:
		return writeMetricCSV(w, block)
	case blocks.BlockTypeChart:
		return writeChartCSV(w, block)
	}

	return ctxerrors.Wrapf(ErrUnknownBlockType, "type %q", block.Type)
}

func writeTableCSV(w *csv.Writer, block blocks.Block) error {
	table, err := block.AsTable()
	if err != nil {
		return ctxerrors.Wrap(err, "decode table block")
	}

	if err := w.Write(table.Columns); err != nil {
		return ctxerrors.Wrap(err, "write table header")
	}

	for _, row := range table.Rows {
		if err := w.Write(row); err != nil {
			return ctxerrors.Wrap(err, "write table row")
		}
	}

	if len(table.Footer) > 0 {
		if err := w.Write(table.Footer); err != nil {
			return ctxerrors.Wrap(err, "write table footer")
		}
	}

	return nil
}

func writeListCSV(w *csv.Writer, block blocks.Block) error {
	list, err := block.AsList()
	if err != nil {
		return ctxerrors.Wrap(err, "decode list block")
	}

	for _, item := range list.Items {
		if err := w.Write([]string{item}); err != nil {
			return ctxerrors.Wrap(err, "write list item")
		}
	}

	return nil
}

func writeKeyValueCSV(w *csv.Writer, block blocks.Block) error {
	kv, err := block.AsKeyValue()
	if err != nil {
		return ctxerrors.Wrap(err, "decode keyvalue block")
	}

	for _, pair := range kv.Pairs {
		if err := w.Write([]string{pair.Key, pair.Value}); err != nil {
			return ctxerrors.Wrap(err, "write keyvalue pair")
		}
	}

	return nil
}

func writeMetricCSV(w *csv.Writer, block blocks.Block) error {
	metric, err := block.AsMetric()
	if err != nil {
		return ctxerrors.Wrap(err, "decode metric block")
	}

	if err := w.Write([]string{metric.Label, metric.Value}); err != nil {
		return ctxerrors.Wrap(err, "write metric row")
	}

	return nil
}

func writeChartCSV(w *csv.Writer, block blocks.Block) error {
	chart, err := block.AsChart()
	if err != nil {
		return ctxerrors.Wrap(err, "decode chart block")
	}

	for i, label := range chart.Labels {
		value := ""
		if i < len(chart.Values) {
			value = strconv.FormatFloat(chart.Values[i], 'f', -1, 64)
		}

		if err := w.Write([]string{label, value}); err != nil {
			return ctxerrors.Wrap(err, "write chart data point")
		}
	}

	return nil
}
