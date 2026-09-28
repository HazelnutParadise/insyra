// gplot/step_test.go

package gplot

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestCreateStepChart(t *testing.T) {
	config := StepChartConfig{
		Title:     "Test Step Chart",
		XAxisName: "X Axis",
		YAxisName: "Y Axis",
		StepStyle: "post",
	}

	plt, err := CreateStepChart(config,
		insyra.NewDataList(1, 2, 3, 4, 5).SetName("Series1"),
		insyra.NewDataList(2, 4, 3, 5, 4).SetName("Series2"),
	)
	if err != nil || plt == nil {
		t.Errorf("CreateStepChart = %v, %v; want a chart", plt, err)
	}
}

func TestCreateStepChartWithIDataLists(t *testing.T) {
	lists := []insyra.IDataList{
		insyra.NewDataList(1, 2, 3, 4, 5).SetName("Series1"),
		insyra.NewDataList(2, 4, 3, 5, 4).SetName("Series2"),
	}

	config := StepChartConfig{
		Title:     "Test Step Chart with DataList",
		XAxisName: "X Axis",
		YAxisName: "Y Axis",
		StepStyle: "mid",
	}

	plt, err := CreateStepChart(config, lists...)
	if err != nil || plt == nil {
		t.Errorf("CreateStepChart = %v, %v; want a chart", plt, err)
	}
}

func TestCreateStepChartWithCustomXAxis(t *testing.T) {
	config := StepChartConfig{
		Title:     "Test Step Chart with Custom X Axis",
		XAxis:     []float64{0, 1, 2, 3, 4},
		XAxisName: "X Axis",
		YAxisName: "Y Axis",
		StepStyle: "pre",
	}

	plt, err := CreateStepChart(config, insyra.NewDataList(1, 2, 3, 4, 5).SetName("Series1"))
	if err != nil || plt == nil {
		t.Errorf("CreateStepChart = %v, %v; want a chart with the custom X axis", plt, err)
	}
}

func TestCreateStepChartWithInvalidStepStyle(t *testing.T) {
	quietFatal(t)

	config := StepChartConfig{
		Title:     "Test Step Chart with Invalid Step Style",
		StepStyle: "invalid",
	}

	plt, err := CreateStepChart(config, insyra.NewDataList(1, 2, 3, 4, 5).SetName("Series1"))
	if err == nil || plt != nil {
		t.Errorf("CreateStepChart = %v, %v; an unknown step style should be an error, not a fallback to post", plt, err)
	}
}
