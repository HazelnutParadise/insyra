package py

import (
	"math"
	"strings"
	"testing"
	"time"
)

// The review of py-results-keep-large-integers found an any already holding a
// pointer replaced by the decoded value, where encoding/json, and the JSON
// path before, decode into what the pointer points to.
func TestAnAnyHoldingAPointerIsDecodedIntoIt(t *testing.T) {
	type payload struct {
		Name string
		N    int64
	}
	type envelope struct {
		Status string
		Data   any
	}
	var p payload
	env := envelope{Data: &p}
	if err := bindPyResult(&env, map[string]any{"Status": "ok", "Data": map[string]any{"Name": "a", "N": int64(bigID)}}); err != nil {
		t.Fatal(err)
	}
	if env.Data != any(&p) || p.Name != "a" || p.N != bigID {
		t.Errorf("got Data %#v and payload %+v", env.Data, p)
	}
	x := int64(-1)
	var v any = &x
	if err := bindPyResult(&v, 5.0); err != nil || v != any(&x) || x != 5 {
		t.Errorf("any holding *int64: got %#v, x = %d, %v", v, x, err)
	}
	y := int64(-1)
	list := []any{&y}
	if err := bindPyResult(&list, []any{7.0}); err != nil || list[0] != any(&y) || y != 7 {
		t.Errorf("[]any holding *int64: got %#v, y = %d, %v", list, y, err)
	}
	// None still sets the any to nil, as in encoding/json.
	v = &x
	if err := bindPyResult(&v, nil); err != nil || v != nil {
		t.Errorf("None into an any holding a pointer gave %#v, %v", v, err)
	}
}

// go-json wraps a nineteen- or twenty-digit number around when it decodes it
// into an int64, an int or a uint64, and reports nothing.
func TestAnIntegerAFieldCannotHoldIsAnError(t *testing.T) {
	var n int64
	if err := bindPyResult(&n, uint64(9999999999999999999)); err == nil || !strings.Contains(err.Error(), "int64") {
		t.Errorf("int64 from 9999999999999999999: got %d, %v", n, err)
	}
	var u uint64
	if err := bindPyResult(&u, 18446744073709551616.0); err == nil {
		t.Errorf("uint64 from 2^64: got %d, nil", u)
	}
	type record struct {
		ID    int64
		D     time.Duration
		Small int32
	}
	var r record
	if err := bindPyResult(&r, map[string]any{"ID": uint64(1 << 63)}); err == nil {
		t.Errorf("an int64 field from 2^63: got %+v, nil", r)
	}
	var list []int
	if err := bindPyResult(&list, []any{1.0, 9.3e18}); err == nil {
		t.Errorf("[]int from 9.3e18: got %v, nil", list)
	}
	var frac int
	if err := bindPyResult(&frac, 1.5); err == nil {
		t.Errorf("int from 1.5: got %d, nil", frac)
	}
	// Values the field holds still decode.
	var ok record
	if err := bindPyResult(&ok, map[string]any{"ID": int64(math.MaxInt64), "D": int64(bigID), "Small": 7.0}); err != nil || ok.ID != math.MaxInt64 || ok.D != time.Duration(bigID) || ok.Small != 7 {
		t.Errorf("got %+v, %v", ok, err)
	}
	var maxU uint64
	if err := bindPyResult(&maxU, uint64(math.MaxUint64)); err != nil || maxU != math.MaxUint64 {
		t.Errorf("uint64 max: got %d, %v", maxU, err)
	}
}
