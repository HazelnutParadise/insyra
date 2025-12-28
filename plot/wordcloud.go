package plot

import (
	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

type WordCloudShape string

const (
	WordCloudShapeCircle    WordCloudShape = "circle"
	WordCloudShapeRect      WordCloudShape = "rect"
	WordCloudShapeRoundRect WordCloudShape = "roundRect"
	WordCloudShapeTriangle  WordCloudShape = "triangle"
	WordCloudShapeDiamond   WordCloudShape = "diamond"
	WordCloudShapePin       WordCloudShape = "pin"
	WordCloudShapeArrow     WordCloudShape = "arrow"
)

// WordCloudConfig defines the configuration for a word cloud chart.
type WordCloudConfig struct {
	Width           string   // Width of the chart (default "900px").
	Height          string   // Height of the chart (default "500px").
	BackgroundColor string   // Background color of the chart (default "white").
	Theme           Theme    // Theme of the chart.
	Title           string   // Title of the chart.
	Subtitle        string   // Subtitle of the chart.
	TitlePos        Position // Optional: Use const PositionXXX.

	Shape     WordCloudShape // Optional: Shape of the word cloud. Use const WordCloudShapeXXX.
	SizeRange []float32      // Optional: Size range for the words, e.g., [14, 80].
}

// CreateWordCloud generates and returns a *charts.WordCloud object based on WordCloudChartConfig.
func CreateWordCloud(config WordCloudConfig, data insyra.IDataList) *charts.WordCloud {
	wc := charts.NewWordCloud()

	internal.SetBaseChartGlobalOptions(wc, internal.BaseChartConfig{
		Width:           config.Width,
		Height:          config.Height,
		BackgroundColor: config.BackgroundColor,
		Theme:           string(config.Theme),
		Title:           config.Title,
		Subtitle:        config.Subtitle,
		TitlePos:        string(config.TitlePos),
		HideLegend:      true,
		LegendPos:       "",
	})

	// Default size range if not provided
	if len(config.SizeRange) == 0 {
		config.SizeRange = []float32{0, 25}
	}

	dataMap := make(map[any]float32)
	isEmpty := true
	data.AtomicDo(func(dl *insyra.DataList) {
		if data.Len() > 0 {
			isEmpty = false
		}
		for _, item := range data.Data() {
			_, ok := dataMap[item]
			if !ok {
				dataMap[item] = 1
			} else {
				dataMap[item]++
			}
		}
	})
	if isEmpty {
		insyra.LogWarning("plot", "CreateWordCloud", "No data available for word cloud chart. Returning nil.")
		return nil
	}

	// Add series data to word cloud
	wc.AddSeries("", convertToWordCloudData(dataMap)).
		SetSeriesOptions(
			charts.WithWorldCloudChartOpts(
				opts.WordCloudChart{
					SizeRange: config.SizeRange,
					Shape:     string(config.Shape),
				}),
		)

	return wc
}

// convertToWordCloudData converts map[string]float32 to []opts.WordCloudData.
func convertToWordCloudData(data map[any]float32) []opts.WordCloudData {
	items := make([]opts.WordCloudData, len(data))
	i := 0
	for name, value := range data {
		items[i] = opts.WordCloudData{Name: conv.ToString(name), Value: value}
		i++
	}
	return items
}
