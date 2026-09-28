package stats_test

import (
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestSkewnessRefusesBlank(t *testing.T) {
	_, err := stats.Skewness(insyra.NewDataList(1.0, nil, 3.0, 4.0))
	if err == nil || !strings.Contains(err.Error(), "sample") || !strings.Contains(err.Error(), "row 2") {
		t.Fatalf("expected error naming sample row 2, got %v", err)
	}
}

func TestKurtosisRefusesString(t *testing.T) {
	_, err := stats.Kurtosis([]any{1.0, "x", 3.0, 4.0})
	if err == nil || !strings.Contains(err.Error(), "sample") || !strings.Contains(err.Error(), "row 2") {
		t.Fatalf("expected error naming sample row 2, got %v", err)
	}
}

// A typed nil list and a value ProcessData cannot read are errors. The nil
// list used to crash inside ProcessData, and an unsupported type came back as
// "empty data".
func TestMomentsRefuseNilAndUnreadableInput(t *testing.T) {
	var nilList *insyra.DataList
	for _, input := range []any{nilList, 42} {
		if _, err := stats.Skewness(input); err == nil || err.Error() == "empty data" {
			t.Errorf("Skewness(%#v) error = %v, want ProcessData's error", input, err)
		}
		if _, err := stats.Kurtosis(input); err == nil || err.Error() == "empty data" {
			t.Errorf("Kurtosis(%#v) error = %v, want ProcessData's error", input, err)
		}
	}
	if _, err := stats.Skewness([]float64{}); err == nil || err.Error() != "empty data" {
		t.Errorf("Skewness(empty) error = %v, want empty data", err)
	}
}
