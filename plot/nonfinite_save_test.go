package plot

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestSaveHTMLRefusesNonFiniteData(t *testing.T) {
	for name, v := range map[string]float64{"NaN": math.NaN(), "+Inf": math.Inf(1), "-Inf": math.Inf(-1)} {
		chart := CreateBarChart(BarChartConfig{}, insyra.NewDataList(1.0, v))
		path := filepath.Join(t.TempDir(), "bar.html")
		err := SaveHTML(chart, path)
		if err == nil || !strings.Contains(err.Error(), "cannot be encoded") {
			t.Fatalf("%s: want an encoding error, got %v", name, err)
		}
		if _, statErr := os.Stat(path); statErr == nil {
			t.Fatalf("%s: a file was written", name)
		}
	}
}

func TestSaveHTMLRefusesNonFiniteGaugeAndPie(t *testing.T) {
	if err := SaveHTML(CreateGaugeChart(GaugeChartConfig{}, math.NaN()), filepath.Join(t.TempDir(), "g.html")); err == nil {
		t.Fatal("gauge with NaN saved without error")
	}
	pie := CreatePieChart(PieChartConfig{}, PieItem{Name: "a", Value: 1}, PieItem{Name: "b", Value: math.NaN()})
	if err := SaveHTML(pie, filepath.Join(t.TempDir(), "p.html")); err == nil {
		t.Fatal("pie with NaN saved without error")
	}
}

func TestSaveHTMLStillSavesFiniteData(t *testing.T) {
	chart := CreateBarChart(BarChartConfig{}, insyra.NewDataList(1.0, 2.0))
	if err := SaveHTML(chart, filepath.Join(t.TempDir(), "ok.html")); err != nil {
		t.Fatal(err)
	}
}

func TestSavePNGRefusesNonFiniteData(t *testing.T) {
	chart := CreateBarChart(BarChartConfig{}, insyra.NewDataList(1.0, math.NaN()))
	if err := SavePNG(chart, filepath.Join(t.TempDir(), "x.png"), false); err == nil || !strings.Contains(err.Error(), "cannot be encoded") {
		t.Fatalf("got %v", err)
	}
}
