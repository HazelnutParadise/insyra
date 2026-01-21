// gplot/step_test.go

package gplot

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestCreateStepChart(t *testing.T) {
	// Test with map[string][]float64
	data := map[string][]float64{
		"Series1": {1, 2, 3, 4, 5},
		"Series2": {2, 4, 3, 5, 4},
	}

	config := StepChartConfig{
		Title:     "Test Step Chart",
		XAxisName: "X Axis",
		YAxisName: "Y Axis",
		StepStyle: "post",
	}

	plt := CreateStepChart(config, data)
	if plt == nil {
		t.Error("Expected non-nil plot for map[string][]float64 data")
	}
}

func TestCreateStepChartWithDataList(t *testing.T) {
	// Test with []*insyra.DataList
	dl1 := insyra.NewDataList(1, 2, 3, 4, 5).SetName("Series1")
	dl2 := insyra.NewDataList(2, 4, 3, 5, 4).SetName("Series2")

	config := StepChartConfig{
		Title:     "Test Step Chart with DataList",
		XAxisName: "X Axis",
		YAxisName: "Y Axis",
		StepStyle: "mid",
	}

	plt := CreateStepChart(config, []*insyra.DataList{dl1, dl2})
	if plt == nil {
		t.Error("Expected non-nil plot for []*insyra.DataList data")
	}
}

func TestCreateStepChartWithCustomXAxis(t *testing.T) {
	data := map[string][]float64{
		"Series1": {1, 2, 3, 4, 5},
	}

	xAxis := []float64{0, 1, 2, 3, 4}

	config := StepChartConfig{
		Title:     "Test Step Chart with Custom X Axis",
		XAxis:     xAxis,
		XAxisName: "X Axis",
		YAxisName: "Y Axis",
		StepStyle: "pre",
	}

	plt := CreateStepChart(config, data)
	if plt == nil {
		t.Error("Expected non-nil plot with custom X axis")
	}
}

func TestCreateStepChartWithInvalidStepStyle(t *testing.T) {
	data := map[string][]float64{
		"Series1": {1, 2, 3, 4, 5},
	}

	config := StepChartConfig{
		Title:     "Test Step Chart with Invalid Step Style",
		StepStyle: "invalid",
	}

	plt := CreateStepChart(config, data)
	if plt == nil {
		t.Error("Expected non-nil plot even with invalid step style (should default to post)")
	}
}

func TestCreateStepChartWithUnsupportedDataType(t *testing.T) {
	config := StepChartConfig{
		Title: "Test Step Chart with Unsupported Data",
	}

	plt := CreateStepChart(config, "unsupported")
	if plt != nil {
		t.Error("Expected nil plot for unsupported data type")
	}
}
