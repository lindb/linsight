// Licensed to LinDB under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. LinDB licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package lindb

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	_ "github.com/lindb/arrow/pkg/arrow/array"
)

// schemaLayout pre-processes an Arrow schema once per ResultSet.
type schemaLayout struct {
	// columns mirrors Schema.Columns (hidden columns excluded).
	columns []ColumnMetadata
	// arrowIndices maps each non-hidden column to its Arrow record column index.
	arrowIndices []int
}

// analyzeSchema classifies every column in the schema into tag (String) or
// field (time_series extension), skipping columns explicitly marked hidden=true.
func analyzeSchema(schema *arrow.Schema) *schemaLayout {
	layout := &schemaLayout{}
	for i, f := range schema.Fields() {
		// Skip internal columns that LinDB marks as hidden (hidden=true).
		// We check the value explicitly to avoid skipping columns with hidden=false.
		if idx := f.Metadata.FindKey("hidden"); idx >= 0 && f.Metadata.Values()[idx] == "true" {
			continue
		}
		switch {
		case arrow.TypeEqual(f.Type, arrow.BinaryTypes.String):
			layout.columns = append(layout.columns, ColumnMetadata{Name: f.Name, DataType: DTString})
			layout.arrowIndices = append(layout.arrowIndices, i)
		case isTimeSeriesExtension(f.Type):
			layout.columns = append(layout.columns, ColumnMetadata{Name: f.Name, DataType: DTTimeSeries})
			layout.arrowIndices = append(layout.arrowIndices, i)
		// Other column types (e.g. timestamp, duration) carry internal metadata and
		// are not part of the metric ResultSet visible to the frontend.
		}
	}
	return layout
}

// isTimeSeriesExtension reports whether dt is the LinDB "time_series" extension type.
func isTimeSeriesExtension(dt arrow.DataType) bool {
	ext, ok := dt.(arrow.ExtensionType)
	return ok && ext.ExtensionName() == "time_series"
}

// decodeArrowToResultSet reads an Arrow IPC stream from data and converts it into
// a ResultSet whose structure mirrors LinDB sql/execution/model.ResultSet
// (schema with typed columns + row-oriented data).
func decodeArrowToResultSet(data []byte) (*ResultSet, error) {
	if len(data) == 0 {
		return nil, nil
	}

	reader, err := ipc.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("arrow ipc: create reader: %w", err)
	}
	defer reader.Release()

	rs := &ResultSet{Schema: &Schema{}}

	var layout *schemaLayout

	for reader.Next() {
		rec := reader.RecordBatch()

		// Analyze schema only once — all batches share the same schema.
		if layout == nil {
			layout = analyzeSchema(rec.Schema())
			rs.Schema.Columns = layout.columns
		}

		if err := appendRecordToResultSet(rs, rec, layout); err != nil {
			rec.Release()
			return nil, err
		}
		rec.Release()
	}

	if err := reader.Err(); err != nil {
		return nil, fmt.Errorf("arrow ipc: read record: %w", err)
	}

	return rs, nil
}

// appendRecordToResultSet converts one RecordBatch into rows and appends them to rs.
func appendRecordToResultSet(rs *ResultSet, rec arrow.RecordBatch, layout *schemaLayout) error {
	numRows := int(rec.NumRows())
	if numRows == 0 {
		return nil
	}

	for row := range numRows {
		rowData := make([]any, len(layout.columns))
		for ci, col := range layout.columns {
			arrowCol := rec.Column(layout.arrowIndices[ci])
			switch col.DataType {
			case DTString:
				strCol, ok := arrowCol.(*array.String)
				if !ok {
					return fmt.Errorf("arrow decoder: column %q expected String, got %T", col.Name, arrowCol)
				}
				// strings.Clone copies the value out of the Arrow buffer so it remains
				// valid after rec.Release() decrements the Arrow reference count.
				rowData[ci] = strings.Clone(strCol.Value(row))

			case DTTimeSeries:
				ts, err := extractTimeSeries(arrowCol, col.Name, row)
				if err != nil {
					return err
				}
				rowData[ci] = ts
			}
		}
		rs.Rows = append(rs.Rows, rowData)
	}
	return nil
}

// extractTimeSeries converts one TimeSeries Arrow cell (row) into a TimeSeries value.
// The Arrow TimeSeries struct layout is: {start:int64, end:int64, interval:int64, values:List<float64>}.
func extractTimeSeries(col arrow.Array, fieldName string, row int) (*TimeSeries, error) {
	// The column may be an ExtensionArray wrapping a Struct, or directly a Struct.
	var structArr *array.Struct
	switch c := col.(type) {
	case array.ExtensionArray:
		s, ok := c.Storage().(*array.Struct)
		if !ok {
			return nil, fmt.Errorf("arrow decoder: TimeSeries storage is not Struct for field %q, got %T",
				fieldName, c.Storage())
		}
		structArr = s
	case *array.Struct:
		structArr = c
	default:
		return nil, fmt.Errorf("arrow decoder: unsupported column type %T for field %q", col, fieldName)
	}

	if structArr.IsNull(row) {
		return nil, nil
	}
	if structArr.NumField() < 4 {
		return nil, fmt.Errorf("arrow decoder: expected 4 struct fields (start,end,interval,values), got %d", structArr.NumField())
	}

	start64, ok := structArr.Field(0).(*array.Int64)
	if !ok {
		return nil, fmt.Errorf("arrow decoder: field[0] (start) expected Int64, got %T", structArr.Field(0))
	}
	end64, ok := structArr.Field(1).(*array.Int64)
	if !ok {
		return nil, fmt.Errorf("arrow decoder: field[1] (end) expected Int64, got %T", structArr.Field(1))
	}
	interval64, ok := structArr.Field(2).(*array.Int64)
	if !ok {
		return nil, fmt.Errorf("arrow decoder: field[2] (interval) expected Int64, got %T", structArr.Field(2))
	}
	listArr, ok := structArr.Field(3).(*array.List)
	if !ok {
		return nil, fmt.Errorf("arrow decoder: field[3] (values) expected List, got %T", structArr.Field(3))
	}
	floats, ok := listArr.ListValues().(*array.Float64)
	if !ok {
		return nil, fmt.Errorf("arrow decoder: values list element expected Float64, got %T", listArr.ListValues())
	}

	from, to := listArr.ValueOffsets(row)
	values := make([]float64, int(to-from))
	for i := int(from); i < int(to); i++ {
		// LinDB does not produce null float values in Arrow output; treating null as 0.0
		// is intentional and consistent with the server-side behavior.
		values[i-int(from)] = floats.Value(i)
	}

	return &TimeSeries{
		Values:      values,
		TimeRange:   TimeRange{Start: start64.Value(row), End: end64.Value(row)},
		Interval:    interval64.Value(row),
		NumOfPoints: len(values),
	}, nil
}
