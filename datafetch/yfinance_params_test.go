package datafetch

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// #252: YFHistoryParams is insyra's own struct, converted to go-yfinance's
// parameters inside History. These tests fail when a go-yfinance upgrade adds,
// drops or renames a history parameter, so the upgrade has to decide whether
// to expose it instead of dropping it without notice.

// jsonTags maps each field of a struct type to its JSON tag.
func jsonTags(t reflect.Type) map[string]string {
	tags := make(map[string]string, t.NumField())
	for i := range t.NumField() {
		f := t.Field(i)
		tags[f.Name] = f.Tag.Get("json")
	}
	return tags
}

func TestYFHistoryParamsMatchGoYFinance(t *testing.T) {
	pairs := []struct{ ours, theirs reflect.Type }{
		{reflect.TypeFor[YFHistoryParams](), reflect.TypeFor[models.HistoryParams]()},
		{reflect.TypeFor[YFRepairOptions](), reflect.TypeFor[models.RepairOptions]()},
	}
	for _, p := range pairs {
		ours, theirs := jsonTags(p.ours), jsonTags(p.theirs)
		if !reflect.DeepEqual(ours, theirs) {
			t.Errorf("%s has fields and JSON tags %v, go-yfinance's %s has %v", p.ours.Name(), ours, p.theirs.Name(), theirs)
		}
	}
}

// requireEveryFieldSet fails when a field of v holds its zero value, so a
// field added later must be given a value here before the conversion test
// can pass.
func requireEveryFieldSet(t *testing.T, v reflect.Value) {
	t.Helper()
	for i := range v.NumField() {
		if v.Field(i).IsZero() {
			t.Fatalf("%s.%s is not set; give it a value so the conversion test covers it", v.Type().Name(), v.Type().Field(i).Name)
		}
	}
}

// HistoryContext may abandon a request that is still running, so the
// parameters it hands to go-yfinance must not share memory with the caller's:
// a caller changing its start date afterwards would race the request.
func TestYFHistoryParamsConversionCopiesTheDates(t *testing.T) {
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	params := YFHistoryParams{Start: &start, End: &end, RepairOptions: &YFRepairOptions{FixZeroes: true}}

	model := params.toModel()
	start = start.AddDate(1, 0, 0)
	end = end.AddDate(1, 0, 0)
	params.RepairOptions.FixZeroes = false

	if !model.Start.Equal(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)) || !model.End.Equal(time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("the converted dates followed the caller's change: start %v, end %v", model.Start, model.End)
	}
	if !model.RepairOptions.FixZeroes {
		t.Error("the converted repair options followed the caller's change")
	}
}

func TestYFHistoryParamsConvertEveryField(t *testing.T) {
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	params := YFHistoryParams{
		Period:     "5d",
		Interval:   "1h",
		Start:      &start,
		End:        &end,
		PrePost:    true,
		AutoAdjust: true,
		Actions:    true,
		Repair:     true,
		RepairOptions: &YFRepairOptions{
			FixUnitMixups:   true,
			FixZeroes:       true,
			FixSplits:       true,
			FixDividends:    true,
			FixCapitalGains: true,
		},
		KeepNA: true,
	}
	requireEveryFieldSet(t, reflect.ValueOf(params))
	requireEveryFieldSet(t, reflect.ValueOf(*params.RepairOptions))

	ours, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	theirs, err := json.Marshal(params.toModel())
	if err != nil {
		t.Fatal(err)
	}
	if string(ours) != string(theirs) {
		t.Errorf("History would send %s, the caller asked for %s", theirs, ours)
	}

	if got := (YFHistoryParams{Period: "1mo"}).toModel().RepairOptions; got != nil {
		t.Errorf("a nil RepairOptions became %+v", got)
	}
}
