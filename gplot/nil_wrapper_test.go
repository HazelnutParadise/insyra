package gplot

import (
	"testing"

	"github.com/HazelnutParadise/insyra/isr"
)

func TestBarAndHistogramReturnNilOnNilWrapper(t *testing.T) {
	if CreateBarChart(BarChartConfig{}, isr.DL) != nil {
		t.Fatal("bar chart from nil wrapper should be nil")
	}
	if CreateHistogram(HistogramConfig{}, isr.DL) != nil {
		t.Fatal("histogram from nil wrapper should be nil")
	}
}
