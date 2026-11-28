package preprocess

import (
	"encoding/json"
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
	if string(data) != legacyStandardJSON {
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
