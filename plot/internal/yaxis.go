package internal

import (
	"math"
	"strconv"

	"github.com/HazelnutParadise/insyra"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

// ApplyYAxis builds Y-axis options, applies them to the provided chart, and
// returns isCategory and mapping (if any). It will also set caller's labels
// if a pointer is provided so callers get derived labels automatically.
func ApplyYAxis[T any](chart Chart[T], name string, labels *[]string, min *float64, max *float64, splitNumber *int, formatter string, data ...insyra.IDataList) (bool, map[string]int, func(insyra.IDataList) []opts.LineData, func(insyra.IDataList) []float64) {
	isCategory := false
	var outLabels []string
	// If labels provided, use them
	if labels != nil && len(*labels) > 0 {
		isCategory = true
		outLabels = *labels
	} else {
		// try to parse values as floats; if any cannot be parsed -> category
		for _, dl := range data {
			dl.AtomicDo(func(d *insyra.DataList) {
				for _, s := range d.ToStringSlice() {
					if s == "" {
						continue
					}
					if _, err := strconv.ParseFloat(s, 64); err != nil {
						isCategory = true
						return
					}
				}
			})
			if isCategory {
				break
			}
		}
	}

	yAxis := opts.YAxis{
		Name: name,
	}

	if isCategory {
		// derive labels if not provided
		if len(outLabels) == 0 {
			labelsMap := map[string]struct{}{}
			outLabels = []string{}
			for _, dl := range data {
				dl.AtomicDo(func(d *insyra.DataList) {
					for _, s := range d.ToStringSlice() {
						if _, ok := labelsMap[s]; !ok {
							labelsMap[s] = struct{}{}
							outLabels = append(outLabels, s)
						}
					}
				})
			}
		}
		yAxis.Type = "category"
		yAxis.Data = outLabels
		// default min/max for categories
		if min == nil && max == nil {
			yAxis.Min = 0
			yAxis.Max = float64(len(outLabels) - 1)
		}
	} else {
		yAxis.Type = "value"
		if min != nil {
			yAxis.Min = *min
		}
		if max != nil {
			yAxis.Max = *max
		}
	}

	if splitNumber != nil {
		yAxis.SplitNumber = *splitNumber
	}

	if formatter != "" {
		yAxis.AxisLabel = &opts.AxisLabel{Formatter: types.FuncStr(formatter)}
	}

	// Build mapping and converters
	mapping := map[string]int{}
	var toLineData func(insyra.IDataList) []opts.LineData
	var toF64 func(insyra.IDataList) []float64
	if isCategory {
		for i, lbl := range outLabels {
			mapping[lbl] = i
		}
		toLineData = func(dl insyra.IDataList) []opts.LineData {
			s := dl.ToStringSlice()
			pts := make([]opts.LineData, len(s))
			for i, sv := range s {
				if idx, ok := mapping[sv]; ok {
					pts[i] = opts.LineData{Value: float64(idx)}
				} else {
					pts[i] = opts.LineData{Value: nil}
				}
			}
			return pts
		}
		toF64 = func(dl insyra.IDataList) []float64 {
			s := dl.ToStringSlice()
			arr := make([]float64, len(s))
			for i, sv := range s {
				if idx, ok := mapping[sv]; ok {
					arr[i] = float64(idx)
				} else {
					arr[i] = math.NaN()
				}
			}
			return arr
		}
	} else {
		// numeric converters
		toLineData = func(dl insyra.IDataList) []opts.LineData {
			vals := dl.ToF64Slice()
			pts := make([]opts.LineData, len(vals))
			for i, v := range vals {
				pts[i] = opts.LineData{Value: v}
			}
			return pts
		}
		toF64 = func(dl insyra.IDataList) []float64 {
			return dl.ToF64Slice()
		}
	}

	// Apply to chart
	chart.SetGlobalOptions(charts.WithYAxisOpts(yAxis))

	// Update caller-provided labels slice (if pointer supplied)
	if labels != nil {
		*labels = outLabels
	}

	return isCategory, mapping, toLineData, toF64
}
