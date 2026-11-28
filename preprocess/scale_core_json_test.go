package preprocess

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// On 0.3.x the deprecated core scaler still exists, so the JSON literal and the
// name-first rule above are checked against it: a scaler the core writes loads
// here and transforms every table exactly as the core one does.
func TestCoreScalerJSONLoadsAndTransformsAlike(t *testing.T) {
	core := insyra.NewStandardScaler() //nolint:staticcheck // the deprecated core scaler is the reference the moved one must match
	if err := core.Fit(legacyFitTable(), "a", "B"); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(core)
	if err != nil {
		t.Fatal(err)
	}
	// The literal was recorded on one machine, and a standard deviation can
	// differ in its last bit between amd64 and arm64, so numbers are compared
	// within a relative 1e-12 while every key and string must match exactly.
	var gotDoc, wantDoc any
	if err := json.Unmarshal(data, &gotDoc); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(legacyStandardJSON), &wantDoc); err != nil {
		t.Fatal(err)
	}
	if !jsonClose(gotDoc, wantDoc) {
		t.Fatalf("core JSON changed:\n got %s\nwant %s", data, legacyStandardJSON)
	}
	var moved StandardScaler
	if err := json.Unmarshal(data, &moved); err != nil {
		t.Fatal(err)
	}
	for name, dt := range map[string]*insyra.DataTable{
		"fit table": legacyFitTable(),
		"named B elsewhere": insyra.NewDataTable(
			insyra.NewDataList(1.5, 3.0, 10.0).SetName("a"),
			insyra.NewDataList(100, 200, 300).SetName("x"),
			insyra.NewDataList(4, 8, 16).SetName("B"),
		),
	} {
		want, err := core.Transform(dt)
		if err != nil {
			t.Fatal(err)
		}
		got, err := moved.Transform(dt)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(name, func(t *testing.T) { assertScaleJSONSameTable(t, want, got) })
	}
}

// jsonClose reports whether two decoded JSON documents have the same shape,
// keys and strings, with numbers equal within a relative 1e-12.
func jsonClose(a, b any) bool {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			w, ok := bv[k]
			if !ok || !jsonClose(v, w) {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !jsonClose(av[i], bv[i]) {
				return false
			}
		}
		return true
	case float64:
		bv, ok := b.(float64)
		return ok && math.Abs(av-bv) <= 1e-12*math.Max(1, math.Max(math.Abs(av), math.Abs(bv)))
	default:
		return a == b
	}
}
