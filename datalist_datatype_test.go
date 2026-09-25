package insyra

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/TimLai666/go-decimal/decimal"
)

func TestDataListDataType(t *testing.T) {
	cases := []struct {
		name   string
		values []any
		want   DataType
	}{
		{"integers and floats with gaps", []any{10, 2.5, nil, math.NaN(), int64(3)}, DataTypeNumber},
		{"infinity is a number", []any{math.Inf(1), 1.0}, DataTypeNumber},
		{"a decimal is a number", []any{decimal.MustParse(decimal.Context{Scale: 10}, "1.50")}, DataTypeNumber},
		{"a named numeric type is a number", []any{celsius(21.5)}, DataTypeNumber},
		{"text", []any{"台北", "台中", nil}, DataTypeString},
		{"text that looks numeric stays text", []any{"0050", "2330"}, DataTypeString},
		{"booleans", []any{true, false}, DataTypeBool},
		{"times", []any{time.Unix(0, 0), time.Unix(60, 0)}, DataTypeTime},
		{"bytes", []any{[]byte("a"), []byte("b")}, DataTypeOther},
		{"numbers and text", []any{1, "a"}, DataTypeMixed},
		{"numbers and booleans", []any{1, true}, DataTypeMixed},
		{"only missing values", []any{nil, math.NaN()}, DataTypeEmpty},
		{"no values", nil, DataTypeEmpty},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dl := NewDataList()
			for _, v := range tc.values {
				dl.Append(Cell(v))
			}
			if got := dl.DataType(); got != tc.want {
				t.Fatalf("DataType() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDataTableColDataTypes(t *testing.T) {
	dt := NewDataTable(
		NewDataList(10, nil, 30).SetName("price"),
		NewDataList("台北", nil, "台中").SetName("city"),
		NewDataList(1, "a", nil).SetName("notes"),
		NewDataList(nil, nil, nil).SetName("blank"),
	)
	want := []DataType{DataTypeNumber, DataTypeString, DataTypeMixed, DataTypeEmpty}
	if got := dt.ColDataTypes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ColDataTypes() = %v, want %v", got, want)
	}
}

func TestDataTypeString(t *testing.T) {
	want := map[DataType]string{
		DataTypeEmpty:  "empty",
		DataTypeNumber: "number",
		DataTypeString: "string",
		DataTypeBool:   "bool",
		DataTypeTime:   "time",
		DataTypeOther:  "other",
		DataTypeMixed:  "mixed",
	}
	for dt, s := range want {
		if dt.String() != s {
			t.Errorf("%d.String() = %q, want %q", int(dt), dt.String(), s)
		}
	}
}
