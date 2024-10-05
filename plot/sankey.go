// plot/sankey.go

package plot

import (
	"encoding/json"
	"os"

	"github.com/HazelnutParadise/insyra"
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
	Title      string       // Chart title
	Subtitle   string       // Chart subtitle
	Nodes      []string     // Sankey chart node data (string slice)
	Links      []SankeyLink // Sankey chart link data
	Curveness  float32      // Line curvature
	Color      string       // Line color
	ShowLabels bool         // Whether to display labels
}

// CreateSankeyChart generates and returns a *charts.Sankey object based on SankeyChartConfig.
func CreateSankeyChart(config SankeyChartConfig) *charts.Sankey {
	sankey := charts.NewSankey()

	// 設置標題和副標題
	sankey.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(false),
		}),
	)

	// 將字串切片轉換為 go-echarts 使用的節點格式
	var nodes []opts.SankeyNode
	for _, node := range config.Nodes {
		nodes = append(nodes, opts.SankeyNode{Name: node})
	}

	// 將自定義的連結轉換為 go-echarts 使用的格式
	var links []opts.SankeyLink
	for _, link := range config.Links {
		links = append(links, opts.SankeyLink{
			Source: link.Source,
			Target: link.Target,
			Value:  link.Value,
		})
	}

	// 添加系列數據
	sankey.AddSeries("sankey", nodes, links).
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
		insyra.LogWarning("Failed to read file: %v, return nil.", err)
		return nil, nil
	}

	type SankeyData struct {
		Nodes []string     `json:"nodes"`
		Links []SankeyLink `json:"links"`
	}

	var data SankeyData
	if err := json.Unmarshal(file, &data); err != nil {
		insyra.LogWarning("Failed to unmarshal JSON: %v, return nil.", err)
		return nil, nil
	}

	return data.Nodes, data.Links
}
