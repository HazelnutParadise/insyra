package accel_test

import (
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

func TestExecuteNearestExactRefusesUnreadableBuffers(t *testing.T) {
	cases := []struct {
		name string
		buf  []accel.Buffer
		want string
	}{
		{"float32", []accel.Buffer{{Name: "x", Type: accel.DataTypeFloat64, Values: []float32{1, 2}, Len: 2}}, `"x"`},
		{"int", []accel.Buffer{{Name: "n", Type: accel.DataTypeInt64, Values: []int{1, 2}, Len: 2}}, `"n"`},
		{"nil values", []accel.Buffer{{Name: "z", Type: accel.DataTypeFloat64, Len: 2}}, `"z"`},
		{"type disagrees", []accel.Buffer{{Name: "t", Type: accel.DataTypeInt64, Values: []float64{1, 2}, Len: 2}}, "Type"},
		{"ragged", []accel.Buffer{
			{Name: "a", Type: accel.DataTypeFloat64, Values: []float64{1, 2}, Len: 2},
			{Name: "b", Type: accel.DataTypeFloat64, Values: []float64{1}, Len: 1}}, `"b"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := &accel.Dataset{Rows: 2, Buffers: c.buf}
			queries := [][]float64{make([]float64, len(c.buf))}
			r, err := accel.NewSession().ExecuteNearestExact(ds, queries, 1, accel.WorkloadEstimate{})
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want an error naming %s, got err=%v result=%v", c.want, err, r.Index)
			}
		})
	}
}
