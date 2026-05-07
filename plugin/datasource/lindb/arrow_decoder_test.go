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
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Register TimeSeries extension type via init().
	_ "github.com/lindb/arrow/pkg/arrow/array"
)

// timeSeriesExtType is the registered extension type for "time_series".
// We look it up after init() has run.
var timeSeriesExtType arrow.ExtensionType

func init() {
	// Retrieve the registered extension type so tests can build schemas.
	timeSeriesExtType = arrow.GetExtensionType("time_series")
}

// buildArrowIPC creates an Arrow IPC byte stream containing one RecordBatch.
// Schema: tagCols (String) + fieldCols (time_series extension).
// Each row has len(tags) tag values and len(fields) TimeSeries structs.
func buildArrowIPC(
	tagNames []string,
	fieldNames []string,
	rows []struct {
		tags   []string
		fields []struct {
			start, end, interval int64
			values               []float64
		}
	},
) ([]byte, error) {
	// Build schema fields.
	var arrowFields []arrow.Field
	for _, name := range tagNames {
		arrowFields = append(arrowFields, arrow.Field{Name: name, Type: arrow.BinaryTypes.String})
	}
	tsStorageType := arrow.StructOf(
		arrow.Field{Name: "start", Type: arrow.PrimitiveTypes.Int64},
		arrow.Field{Name: "end", Type: arrow.PrimitiveTypes.Int64},
		arrow.Field{Name: "interval", Type: arrow.PrimitiveTypes.Int64},
		arrow.Field{Name: "values", Type: arrow.ListOf(arrow.PrimitiveTypes.Float64)},
	)
	for _, name := range fieldNames {
		// Use the registered extension type so IPC reader recognises it.
		if timeSeriesExtType != nil {
			arrowFields = append(arrowFields, arrow.Field{Name: name, Type: timeSeriesExtType})
		} else {
			// Fallback: use raw struct storage (extension type not registered in test env).
			arrowFields = append(arrowFields, arrow.Field{Name: name, Type: tsStorageType})
		}
	}

	schema := arrow.NewSchema(arrowFields, nil)
	alloc := memory.NewGoAllocator()
	rb := array.NewRecordBuilder(alloc, schema)
	defer rb.Release()

	colIdx := 0
	// Append tag columns.
	for range tagNames {
		b := rb.Field(colIdx).(*array.StringBuilder)
		for _, row := range rows {
			b.Append(row.tags[colIdx])
		}
		colIdx++
	}
	// Append TimeSeries columns.
	for fi := range fieldNames {
		var structB *array.StructBuilder
		if timeSeriesExtType != nil {
			structB = rb.Field(colIdx).(*array.ExtensionBuilder).Builder.(*array.StructBuilder)
		} else {
			structB = rb.Field(colIdx).(*array.StructBuilder)
		}
		for _, row := range rows {
			fd := row.fields[fi]
			structB.Append(true)
			structB.FieldBuilder(0).(*array.Int64Builder).Append(fd.start)
			structB.FieldBuilder(1).(*array.Int64Builder).Append(fd.end)
			structB.FieldBuilder(2).(*array.Int64Builder).Append(fd.interval)
			lb := structB.FieldBuilder(3).(*array.ListBuilder)
			lb.Append(true)
			vb := lb.ValueBuilder().(*array.Float64Builder)
			for _, v := range fd.values {
				vb.Append(v)
			}
		}
		colIdx++
	}

	rec := rb.NewRecordBatch()
	defer rec.Release()

	var buf bytes.Buffer
	w := ipc.NewWriter(&buf, ipc.WithSchema(schema))
	if err := w.Write(rec); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestDecodeArrowToResultSet_Empty(t *testing.T) {
	rs, err := decodeArrowToResultSet(nil)
	assert.NoError(t, err)
	assert.Nil(t, rs)

	rs2, err2 := decodeArrowToResultSet([]byte{})
	assert.NoError(t, err2)
	assert.Nil(t, rs2)
}

func TestDecodeArrowToResultSet_InvalidData(t *testing.T) {
	_, err := decodeArrowToResultSet([]byte("not arrow data"))
	assert.Error(t, err)
}

func TestDecodeArrowToResultSet_SingleRowNoTags(t *testing.T) {
	data, err := buildArrowIPC(
		nil, // no tag columns
		[]string{"value"},
		[]struct {
			tags   []string
			fields []struct {
				start, end, interval int64
				values               []float64
			}
		}{
			{
				fields: []struct {
					start, end, interval int64
					values               []float64
				}{
					{start: 1000, end: 3000, interval: 1000, values: []float64{1.0, 2.0}},
				},
			},
		},
	)
	require.NoError(t, err)

	rs, err := decodeArrowToResultSet(data)
	require.NoError(t, err)
	require.NotNil(t, rs)

	// Schema should have one TimeSeries column.
	require.Len(t, rs.Schema.Columns, 1)
	assert.Equal(t, "value", rs.Schema.Columns[0].Name)
	assert.Equal(t, DTTimeSeries, rs.Schema.Columns[0].DataType)

	// Should have one row.
	require.Len(t, rs.Rows, 1)
	ts, ok := rs.Rows[0][0].(*TimeSeries)
	require.True(t, ok)
	assert.Equal(t, []float64{1.0, 2.0}, ts.Values)
	assert.Equal(t, int64(1000), ts.TimeRange.Start)
	assert.Equal(t, int64(3000), ts.TimeRange.End)
	assert.Equal(t, int64(1000), ts.Interval)
	assert.Equal(t, 2, ts.NumOfPoints)
}

func TestDecodeArrowToResultSet_MultiTagMultiField(t *testing.T) {
	data, err := buildArrowIPC(
		[]string{"host", "region"},
		[]string{"cpu", "mem"},
		[]struct {
			tags   []string
			fields []struct {
				start, end, interval int64
				values               []float64
			}
		}{
			{
				tags: []string{"server1", "us-east"},
				fields: []struct {
					start, end, interval int64
					values               []float64
				}{
					{start: 0, end: 2000, interval: 1000, values: []float64{10.0, 20.0}},
					{start: 0, end: 2000, interval: 1000, values: []float64{50.0, 60.0}},
				},
			},
			{
				tags: []string{"server2", "us-west"},
				fields: []struct {
					start, end, interval int64
					values               []float64
				}{
					{start: 0, end: 2000, interval: 1000, values: []float64{30.0, 40.0}},
					{start: 0, end: 2000, interval: 1000, values: []float64{70.0, 80.0}},
				},
			},
		},
	)
	require.NoError(t, err)

	rs, err := decodeArrowToResultSet(data)
	require.NoError(t, err)
	require.NotNil(t, rs)

	// Schema: 2 tag columns + 2 field columns.
	require.Len(t, rs.Schema.Columns, 4)
	assert.Equal(t, ColumnMetadata{Name: "host", DataType: DTString}, rs.Schema.Columns[0])
	assert.Equal(t, ColumnMetadata{Name: "region", DataType: DTString}, rs.Schema.Columns[1])
	assert.Equal(t, ColumnMetadata{Name: "cpu", DataType: DTTimeSeries}, rs.Schema.Columns[2])
	assert.Equal(t, ColumnMetadata{Name: "mem", DataType: DTTimeSeries}, rs.Schema.Columns[3])

	// Two rows.
	require.Len(t, rs.Rows, 2)

	// First row.
	row0 := rs.Rows[0]
	assert.Equal(t, "server1", row0[0])
	assert.Equal(t, "us-east", row0[1])
	ts0cpu, ok := row0[2].(*TimeSeries)
	require.True(t, ok)
	assert.Equal(t, []float64{10.0, 20.0}, ts0cpu.Values)
	ts0mem, ok := row0[3].(*TimeSeries)
	require.True(t, ok)
	assert.Equal(t, []float64{50.0, 60.0}, ts0mem.Values)

	// Second row.
	row1 := rs.Rows[1]
	assert.Equal(t, "server2", row1[0])
	assert.Equal(t, "us-west", row1[1])
	ts1cpu, ok := row1[2].(*TimeSeries)
	require.True(t, ok)
	assert.Equal(t, []float64{30.0, 40.0}, ts1cpu.Values)
}

func TestDecodeArrowToResultSet_HiddenColumnSkipped(t *testing.T) {
	// Build a schema that contains a hidden=true column between two visible columns.
	// The hidden column should be absent from rs.Schema.Columns.
	tsStorageType := arrow.StructOf(
		arrow.Field{Name: "start", Type: arrow.PrimitiveTypes.Int64},
		arrow.Field{Name: "end", Type: arrow.PrimitiveTypes.Int64},
		arrow.Field{Name: "interval", Type: arrow.PrimitiveTypes.Int64},
		arrow.Field{Name: "values", Type: arrow.ListOf(arrow.PrimitiveTypes.Float64)},
	)
	hiddenMeta := arrow.NewMetadata([]string{"hidden"}, []string{"true"})
	var tsType arrow.DataType
	if timeSeriesExtType != nil {
		tsType = timeSeriesExtType
	} else {
		tsType = tsStorageType
	}
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "host", Type: arrow.BinaryTypes.String},
		{Name: "_internal", Type: arrow.BinaryTypes.String, Metadata: hiddenMeta},
		{Name: "cpu", Type: tsType},
	}, nil)

	alloc := memory.NewGoAllocator()
	rb := array.NewRecordBuilder(alloc, schema)
	defer rb.Release()

	// host column
	rb.Field(0).(*array.StringBuilder).Append("server1")
	// _internal column (hidden)
	rb.Field(1).(*array.StringBuilder).Append("skip-me")
	// cpu column
	var structB *array.StructBuilder
	if timeSeriesExtType != nil {
		structB = rb.Field(2).(*array.ExtensionBuilder).Builder.(*array.StructBuilder)
	} else {
		structB = rb.Field(2).(*array.StructBuilder)
	}
	structB.Append(true)
	structB.FieldBuilder(0).(*array.Int64Builder).Append(0)
	structB.FieldBuilder(1).(*array.Int64Builder).Append(1000)
	structB.FieldBuilder(2).(*array.Int64Builder).Append(1000)
	lb := structB.FieldBuilder(3).(*array.ListBuilder)
	lb.Append(true)
	lb.ValueBuilder().(*array.Float64Builder).Append(42.0)

	rec := rb.NewRecordBatch()
	defer rec.Release()

	var buf bytes.Buffer
	w := ipc.NewWriter(&buf, ipc.WithSchema(schema))
	require.NoError(t, w.Write(rec))
	require.NoError(t, w.Close())

	rs, err := decodeArrowToResultSet(buf.Bytes())
	require.NoError(t, err)
	require.NotNil(t, rs)

	// Hidden column must not appear in the schema.
	require.Len(t, rs.Schema.Columns, 2)
	assert.Equal(t, "host", rs.Schema.Columns[0].Name)
	assert.Equal(t, "cpu", rs.Schema.Columns[1].Name)

	// Row should have 2 cells, no hidden value.
	require.Len(t, rs.Rows, 1)
	assert.Equal(t, "server1", rs.Rows[0][0])
	ts, ok := rs.Rows[0][1].(*TimeSeries)
	require.True(t, ok)
	assert.Equal(t, []float64{42.0}, ts.Values)
}

func TestDecodeArrowToResultSet_MultiBatch(t *testing.T) {
	// Build an IPC stream with two separate RecordBatches to verify that
	// rows from all batches are accumulated into a single ResultSet.
	batch1, err := buildArrowIPC(
		[]string{"host"},
		[]string{"cpu"},
		[]struct {
			tags   []string
			fields []struct {
				start, end, interval int64
				values               []float64
			}
		}{
			{tags: []string{"a"}, fields: []struct {
				start, end, interval int64
				values               []float64
			}{{start: 0, end: 1000, interval: 1000, values: []float64{1.0}}}},
		},
	)
	require.NoError(t, err)

	batch2, err := buildArrowIPC(
		[]string{"host"},
		[]string{"cpu"},
		[]struct {
			tags   []string
			fields []struct {
				start, end, interval int64
				values               []float64
			}
		}{
			{tags: []string{"b"}, fields: []struct {
				start, end, interval int64
				values               []float64
			}{{start: 0, end: 1000, interval: 1000, values: []float64{2.0}}}},
		},
	)
	require.NoError(t, err)

	// Concatenate the two IPC payloads by re-encoding them into a single stream.
	// Each buildArrowIPC call wraps the batch with its own IPC header; we need to
	// decode both and re-write into one stream.
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "host", Type: arrow.BinaryTypes.String},
	}, nil)
	var tsType arrow.DataType
	if timeSeriesExtType != nil {
		tsType = timeSeriesExtType
	} else {
		tsType = arrow.StructOf(
			arrow.Field{Name: "start", Type: arrow.PrimitiveTypes.Int64},
			arrow.Field{Name: "end", Type: arrow.PrimitiveTypes.Int64},
			arrow.Field{Name: "interval", Type: arrow.PrimitiveTypes.Int64},
			arrow.Field{Name: "values", Type: arrow.ListOf(arrow.PrimitiveTypes.Float64)},
		)
	}
	fullSchema := arrow.NewSchema([]arrow.Field{
		{Name: "host", Type: arrow.BinaryTypes.String},
		{Name: "cpu", Type: tsType},
	}, nil)
	_ = schema

	var combined bytes.Buffer
	cw := ipc.NewWriter(&combined, ipc.WithSchema(fullSchema))

	for _, payload := range [][]byte{batch1, batch2} {
		r, rErr := ipc.NewReader(bytes.NewReader(payload))
		require.NoError(t, rErr)
		for r.Next() {
			require.NoError(t, cw.Write(r.RecordBatch()))
		}
		r.Release()
	}
	require.NoError(t, cw.Close())

	rs, err := decodeArrowToResultSet(combined.Bytes())
	require.NoError(t, err)
	require.NotNil(t, rs)

	// Both batches should produce rows in the same ResultSet.
	require.Len(t, rs.Rows, 2)
	assert.Equal(t, "a", rs.Rows[0][0])
	assert.Equal(t, "b", rs.Rows[1][0])
}
