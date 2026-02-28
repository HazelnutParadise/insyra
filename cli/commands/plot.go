package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
	insyraplot "github.com/HazelnutParadise/insyra/plot"
)

func init() {
	_ = Register(&CommandHandler{Name: "plot", Usage: "plot <type> <var> [options...] [save <file>]", Description: "Create charts from variables", Run: runPlotCommand})
}

func runPlotCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: plot <type> <var> [options...] [save <file>]")
	}

	plotType := strings.ToLower(args[0])
	variableName := args[1]
	savePath, err := parsePlotSavePath(args[2:], plotType)
	if err != nil {
		return err
	}

	value, exists := ctx.Vars[variableName]
	if !exists {
		return fmt.Errorf("variable not found: %s", variableName)
	}

	var chart insyraplot.Renderable
	switch plotType {
	case "line":
		series, makeErr := extractPlotSeries(value)
		if makeErr != nil {
			return makeErr
		}
		chart = insyraplot.CreateLineChart(insyraplot.LineChartConfig{Title: "Line Chart"}, series...)
	case "bar":
		series, makeErr := extractPlotSeries(value)
		if makeErr != nil {
			return makeErr
		}
		chart = insyraplot.CreateBarChart(insyraplot.BarChartConfig{Title: "Bar Chart"}, series...)
	case "scatter":
		scatterData, makeErr := extractScatterSeries(value)
		if makeErr != nil {
			return makeErr
		}
		chart = insyraplot.CreateScatterChart(insyraplot.ScatterChartConfig{Title: "Scatter Chart"}, scatterData)
	default:
		return fmt.Errorf("unsupported plot type: %s", plotType)
	}

	if chart == nil {
		return fmt.Errorf("failed to create chart")
	}

	ext := strings.ToLower(filepath.Ext(savePath))
	if ext == ".png" {
		if err := insyraplot.SavePNG(chart, savePath); err != nil {
			return err
		}
	} else {
		if err := insyraplot.SaveHTML(chart, savePath); err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintf(ctx.Output, "plot saved: %s\n", savePath)
	return nil
}

func parsePlotSavePath(args []string, plotType string) (string, error) {
	for i := 0; i < len(args); i++ {
		if strings.EqualFold(args[i], "save") {
			if i+1 >= len(args) {
				return "", fmt.Errorf("missing file path after save")
			}
			return args[i+1], nil
		}
	}
	return fmt.Sprintf("%s.html", plotType), nil
}

func extractPlotSeries(value any) ([]insyra.IDataList, error) {
	switch typed := value.(type) {
	case *insyra.DataList:
		return []insyra.IDataList{typed}, nil
	case *insyra.DataTable:
		_, cols := typed.Size()
		series := make([]insyra.IDataList, 0, cols)
		for i := 0; i < cols; i++ {
			series = append(series, typed.GetColByNumber(i))
		}
		return series, nil
	default:
		return nil, fmt.Errorf("plot supports DataList or DataTable")
	}
}

func extractScatterSeries(value any) (map[string][]insyraplot.ScatterPoint, error) {
	dt, ok := value.(*insyra.DataTable)
	if !ok {
		return nil, fmt.Errorf("scatter plot requires DataTable with at least 2 columns")
	}
	_, cols := dt.Size()
	if cols < 2 {
		return nil, fmt.Errorf("scatter plot requires DataTable with at least 2 columns")
	}
	x := dt.GetColByNumber(0).ToF64Slice()
	y := dt.GetColByNumber(1).ToF64Slice()
	length := len(x)
	if len(y) < length {
		length = len(y)
	}
	points := make([]insyraplot.ScatterPoint, 0, length)
	for i := 0; i < length; i++ {
		points = append(points, insyraplot.ScatterPoint{X: x[i], Y: y[i]})
	}
	return map[string][]insyraplot.ScatterPoint{"series": points}, nil
}
