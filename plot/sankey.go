// plot/sankey.go

package plot

import (
	"encoding/json"
	"os"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// SankeyLink custom link structure.
type SankeyLink struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Value  float32 `json:"value"`
}

// SankeyChartConfig defines the configuration for a Sankey chart.
type SankeyChartConfig struct {
	Width           string   // Width of the chart (default "900px").
	Height          string   // Height of the chart (default "500px").
	BackgroundColor string   // Background color of the chart (default "white").
	Theme           Theme    // Theme of the chart.
	Title           string   // Title of the chart.
	Subtitle        string   // Subtitle of the chart.
	TitlePos        Position // Optional: Use const PositionXXX.

	Nodes      []string // Sankey chart node data (string slice)
	Curveness  float32  // Line curvature
	Color      string   // Line color, e.g., "source", "target", or a specific color code
	ShowLabels bool     // Whether to display labels
}

// CreateSankeyChart generates and returns a *charts.Sankey object based on SankeyChartConfig.
func CreateSankeyChart(config SankeyChartConfig, links ...SankeyLink) *charts.Sankey {
	if len(links) == 0 {
		insyra.LogWarning("plot", "CreateSankeyChart", "No link data available for sankey chart. Returning nil.")
		return nil
	}
	sankey := charts.NewSankey()

	internal.SetBaseChartGlobalOptions(sankey, internal.BaseChartConfig{
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

	// 將字串切片轉換為 go-echarts 使用的節點格式
	var nodes []opts.SankeyNode
	for _, node := range config.Nodes {
		nodes = append(nodes, opts.SankeyNode{Name: node})
	}

	// 將自定義的連結轉換為 go-echarts 使用的格式
	var sankeyLinks []opts.SankeyLink
	for _, link := range links {
		sankeyLinks = append(sankeyLinks, opts.SankeyLink{
			Source: link.Source,
			Target: link.Target,
			Value:  link.Value,
		})
	}

	// 添加系列數據
	sankey.AddSeries("sankey", nodes, sankeyLinks).
		SetSeriesOptions(
			charts.WithLineStyleOpts(opts.LineStyle{
				Color:     config.Color,
				Curveness: config.Curveness,
			}),
			charts.WithLabelOpts(opts.Label{
				Show: opts.Bool(config.ShowLabels),
			}),
		)

	return sankey
}

// LoadSankeyDataFromFile reads the Sankey chart node and link data from a JSON file.
func LoadSankeyDataFromFile(filePath string) ([]string, []SankeyLink) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		insyra.LogWarning("plot", "LoadSankeyDataFromFile", "Failed to read file: %v, return nil.", err)
		return nil, nil
	}

	type SankeyData struct {
		Nodes []string     `json:"nodes"`
		Links []SankeyLink `json:"links"`
	}

	var data SankeyData
	if err := json.Unmarshal(file, &data); err != nil {
		insyra.LogWarning("plot", "LoadSankeyDataFromFile", "Failed to unmarshal JSON: %v, return nil.", err)
		return nil, nil
	}

	return data.Nodes, data.Links
}
