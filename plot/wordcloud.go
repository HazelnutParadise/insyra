package plot

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// WordCloudConfig defines the configuration for a word cloud chart.
type WordCloudConfig struct {
	Title     string             // Title of the word cloud chart.
	Subtitle  string             // Subtitle of the word cloud chart.
	Data      map[string]float32 // Accepts map[string]float32 for words and their frequencies.
	Shape     string             // Optional: Shape of the word cloud (e.g., "circle", "cardioid", "star").
	SizeRange []float32          // Optional: Size range for the words, e.g., [14, 80].
}

// CreateWordCloud generates and returns a *charts.WordCloud object based on WordCloudChartConfig.
func CreateWordCloud(config WordCloudConfig) *charts.WordCloud {
	wc := charts.NewWordCloud()

	// Set title and subtitle
	wc.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(false)}),
	)

	// Default size range if not provided
	if len(config.SizeRange) == 0 {
		config.SizeRange = []float32{0, 25}
	}

	// Add series data to word cloud
	wc.AddSeries("wordcloud", convertToWordCloudData(config.Data)).
		SetSeriesOptions(
			charts.WithWorldCloudChartOpts(
				opts.WordCloudChart{
					SizeRange: config.SizeRange,
					Shape:     config.Shape,
				}),
		)

	return wc
}

// convertToWordCloudData converts map[string]float32 to []opts.WordCloudData.
func convertToWordCloudData(data map[string]float32) []opts.WordCloudData {
	items := make([]opts.WordCloudData, len(data))
	i := 0
	for name, value := range data {
		items[i] = opts.WordCloudData{Name: name, Value: value}
		i++
	}
	return items
}
