package py

import (
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// An integer neither an int64 nor a uint64 holds, such as the Int128 polars
// 2.0 gives for Int64 + UInt64, came back as the nearest float64 with no
// error wherever the result type held no float (#415).
const (
	above64 = "18446744073709551620" // 2**64 + 4
	below64 = "-9223372036854775809" // -(2**63) - 1
)

// bindMessage decodes the message insyra.Return would send for data and binds
// its value into out.
func bindMessage(t *testing.T, data string, out any) error {
	t.Helper()
	m, err := decodeResultMessage([]byte(`{"execution_id": "a1", "data": [` + data + `, null]}`))
	if err != nil {
		return err
	}
	return bindPyResult(out, m.Data[0])
}

func isWideIntegerError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "fits neither an int64 nor a uint64") && strings.Contains(err.Error(), "str(x)")
}

func TestAnIntegerBeyond64BitsIsAnErrorWhereNoFloatWasAsked(t *testing.T) {
	for _, n := range []string{above64, below64} {
		var v any
		if err := bindMessage(t, n, &v); !isWideIntegerError(err) {
			t.Errorf("%s into any: got %#v, %v", n, v, err)
		}
		var l []any
		if err := bindMessage(t, `[1, `+n+`]`, &l); !isWideIntegerError(err) {
			t.Errorf("%s in []any: got %#v, %v", n, l, err)
		}
		var m map[string]any
		if err := bindMessage(t, `{"id": `+n+`}`, &m); !isWideIntegerError(err) {
			t.Errorf("%s in map[string]any: got %#v, %v", n, m, err)
		}
		var r struct{ ID any }
		if err := bindMessage(t, `{"ID": `+n+`}`, &r); !isWideIntegerError(err) {
			t.Errorf("%s in an any field: got %#v, %v", n, r, err)
		}
		var i int64
		if err := bindMessage(t, n, &i); !isWideIntegerError(err) {
			t.Errorf("%s into int64: got %d, %v", n, i, err)
		}
		var u uint64
		if err := bindMessage(t, n, &u); !isWideIntegerError(err) {
			t.Errorf("%s into uint64: got %d, %v", n, u, err)
		}
		var dt *insyra.DataTable
		if err := bindMessage(t, `{"_insyra_type": "datatable", "data": [[`+n+`]], "columns": ["id"]}`, &dt); !isWideIntegerError(err) {
			t.Errorf("%s in a table cell: got %v", n, err)
		}
		var dl *insyra.DataList
		if err := bindMessage(t, `{"_insyra_type": "datalist", "data": [1, `+n+`]}`, &dl); !isWideIntegerError(err) {
			t.Errorf("%s in a list cell: got %v", n, err)
		}
		var embedded struct{ *insyra.DataTable }
		if err := bindMessage(t, `[[`+n+`]]`, &embedded); !isWideIntegerError(err) {
			t.Errorf("%s in an embedded table: got %v", n, err)
		}
	}
}

func TestAnIntegerBeyond64BitsIsApproximatedWhereAFloatWasAsked(t *testing.T) {
	var f float64
	if err := bindMessage(t, above64, &f); err != nil || f != 18446744073709551620.0 {
		t.Errorf("float64: got %v, %v", f, err)
	}
	var p *float64
	if err := bindMessage(t, below64, &p); err != nil || p == nil || *p != -9223372036854775809.0 {
		t.Errorf("*float64: got %v, %v", p, err)
	}
	var r struct {
		Total float64
		Rest  []float32
		ByKey map[string]float64
	}
	if err := bindMessage(t, `{"Total": `+above64+`, "Rest": [1.5, `+above64+`], "ByKey": {"a": `+below64+`}}`, &r); err != nil || r.Total != 18446744073709551620.0 || len(r.Rest) != 2 || r.Rest[1] != float32(18446744073709551620.0) || r.ByKey["a"] != -9223372036854775809.0 {
		t.Errorf("struct: got %+v, %v", r, err)
	}
	// A float32 field cannot hold every float64 the integer becomes.
	var small float32
	if err := bindMessage(t, strings.Repeat("9", 40), &small); err == nil {
		t.Errorf("float32 from a 40-digit integer: got %v, nil", small)
	}
}

func TestAnIntegerTooLargeForAFloatSaysSo(t *testing.T) {
	big := strings.Repeat("9", 400)
	var f float64
	err := bindMessage(t, big, &f)
	if err == nil || !strings.Contains(err.Error(), "float64") || !strings.Contains(err.Error(), "str(x)") || strings.Contains(err.Error(), big) {
		t.Errorf("got %v, want an error that names float64, suggests str(x) and shortens the number", err)
	}
}

// The 64-bit integers and Python floats keep decoding as before.
func TestIntegersWithin64BitsAndFloatsAreUnchanged(t *testing.T) {
	var v []any
	if err := bindMessage(t, `[18446744073709551615, -9223372036854775808, 1e+20, 3]`, &v); err != nil {
		t.Fatal(err)
	}
	if len(v) != 4 || v[0] != any(uint64(18446744073709551615)) || v[1] != any(int64(-9223372036854775808)) || v[2] != any(1e20) || v[3] != any(3.0) {
		t.Errorf("got %#v", v)
	}
	var s struct {
		ID string `json:"id"`
	}
	if err := bindMessage(t, `{"id": "`+above64+`"}`, &s); err != nil || s.ID != above64 {
		t.Errorf("an integer sent as a string: got %+v, %v", s, err)
	}
}
