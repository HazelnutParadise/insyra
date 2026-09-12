package insyra

import (
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// printsItself is a struct cell that knows its own text but not how to marshal
// itself, which is the shape of a decimal read from a Parquet column: unexported
// fields, a String() method, and nothing else.
type printsItself struct {
	digits string
}

func (p printsItself) String() string { return p.digits }

// silent has neither.
type silent struct{ n int }

func TestACellThatKnowsItsOwnTextIsShownAsThatText(t *testing.T) {
	if got := utils.FormatValue(printsItself{"10.50"}); got != "10.50" {
		t.Errorf("FormatValue = %q, want %q", got, "10.50")
	}
	if got := utils.FormatValue(silent{1}); got != "<insyra.silent>" {
		t.Errorf("a struct with no String() should still show its type, got %q", got)
	}
	// The explicit cases above the fallback must be untouched.
	if got := utils.FormatValue(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)); got != "2024-01-01 00:00:00" {
		t.Errorf("time.Time = %q", got)
	}
	if got := utils.FormatValue(1.5); got != "1.5" {
		t.Errorf("float64 = %q", got)
	}
}

func TestExportingToJSONDoesNotDropAValueItCouldWrite(t *testing.T) {
	dt := NewDataTable(NewDataList(printsItself{"10.50"}).SetName("price"))
	got := dt.ToJSON_String(true)
	if strings.Contains(got, "{}") {
		t.Errorf("the value was dropped: %s", got)
	}
	if !strings.Contains(got, `"10.50"`) {
		t.Errorf("JSON does not carry the value: %s", got)
	}
}

func TestAValueThatMarshalsItselfKeepsItsOwnForm(t *testing.T) {
	dt := NewDataTable(NewDataList(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).SetName("t"))
	got := dt.ToJSON_String(true)
	if !strings.Contains(got, "2024-01-01T00:00:00Z") {
		t.Errorf("time.Time lost its RFC 3339 form: %s", got)
	}
}
