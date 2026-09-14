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
