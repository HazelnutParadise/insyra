// plot/save_chart.go

package plot

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/HazelnutParadise/insyra"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/snapshot-chromedp/render"
	"github.com/google/uuid"
)

// Renderer
// Any kinds of charts have their render implementation, and
// you can define your own render logic easily.
type Renderable interface {
	Render(w io.Writer) error
	RenderContent() []byte
}

// SaveHTML 將圖表渲染並保存為 HTML 文件
func SaveHTML(chart Renderable, path string, animation ...bool) {
	if len(animation) > 0 && !animation[0] {
		disableAnimation(chart)
	} else {
		enableAnimation(chart)
	}

	// 創建輸出文件
	f, err := os.Create(path)
	if err != nil {
		insyra.LogFatal("plot", "SaveHTML", "failed to create file %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()

	// 渲染圖表到指定文件
	if err := chart.Render(f); err != nil {
		insyra.LogFatal("plot", "SaveHTML", "failed to render chart: %v", err)
	}
	insyra.LogInfo("plot", "SaveHTML", "successfully saved HTML file in %s.", path)
}

// SavePNG 將圖表渲染為 PNG 文件，使用 snapshot-chromedp
func SavePNG(chart Renderable, pngPath string) {
	disableAnimation(chart)

	chartContentBytes := chart.RenderContent()

	useOnlineService := false
	uuid := uuid.New().String()
	tempDir := os.TempDir()
	snapshotConfig := render.NewSnapshotConfig(chartContentBytes, pngPath)
	snapshotConfig.Quality = 2
	// Use a temp directory (not a filename) for HTML assets to avoid incorrect path joins on Windows.
	snapshotConfig.HtmlPath = filepath.Join(tempDir, uuid+"_temp")
	// Ensure the temp directory exists so snapshotter can write files into it.
	if mkerr := os.MkdirAll(snapshotConfig.HtmlPath, 0700); mkerr != nil {
		insyra.LogWarning("plot", "SavePNG", "failed to create temp html dir %s: %v", snapshotConfig.HtmlPath, mkerr)
	}
	// Ensure temp directory is removed when done (safe even if MakeSnapshot cleans up already).
	defer func() { _ = os.RemoveAll(snapshotConfig.HtmlPath) }()

	err := render.MakeSnapshot(snapshotConfig)
	if err != nil {
		insyra.LogWarning("plot", "SavePNG", "failed to render chart to PNG: %v, trying to use HazelnutParadise online service...", err)
		useOnlineService = true
	}

	if useOnlineService {
		// 使用 http.NewRequest 並設定 Accept 標頭為 image/png
		req, err := http.NewRequest(
			"POST",
			"https://server3.hazelnut-paradise.com/api/v1/go-echarts-render-image",
			bytes.NewReader(chartContentBytes),
		)
		if err != nil {
			insyra.LogFatal("plot", "SavePNG", "failed to create HTTP request: %v", err)
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Accept", "image/png") // 指定接收 PNG 格式

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			insyra.LogFatal("plot", "SavePNG", "failed to send HTTP request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			insyra.LogFatal("plot", "SavePNG", "online service returned non-OK status: %s", resp.Status)
		}
		insyra.LogInfo("plot", "SavePNG", "successfully received PNG response from HazelnutParadise online service.")
		// 將響應的 PNG 數據寫入文件
		outFile, err := os.Create(pngPath)
		if err != nil {
			insyra.LogFatal("plot", "SavePNG", "failed to create PNG file: %v", err)
		}
		defer func() { _ = outFile.Close() }()

		_, err = io.Copy(outFile, resp.Body)
		if err != nil {
			insyra.LogFatal("plot", "SavePNG", "failed to save PNG file: %v", err)
		}
	}

	insyra.LogInfo("plot", "SavePNG", "successfully saved PNG file in %s.", pngPath)
}

func disableAnimation(chart Renderable) {
	if barChart, ok := chart.(*charts.Bar); ok {
		barChart.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if lineChart, ok := chart.(*charts.Line); ok {
		lineChart.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if pieChart, ok := chart.(*charts.Pie); ok {
		pieChart.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if scatterChart, ok := chart.(*charts.Scatter); ok {
		scatterChart.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if heatMap, ok := chart.(*charts.HeatMap); ok {
		heatMap.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if mapChart, ok := chart.(*charts.Map); ok {
		mapChart.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if radarChart, ok := chart.(*charts.Radar); ok {
		radarChart.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if funnelChart, ok := chart.(*charts.Funnel); ok {
		funnelChart.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if liquidChart, ok := chart.(*charts.Liquid); ok {
		liquidChart.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if wordCloudChart, ok := chart.(*charts.WordCloud); ok {
		wordCloudChart.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if boxPlot, ok := chart.(*charts.BoxPlot); ok {
		boxPlot.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if kline, ok := chart.(*charts.Kline); ok {
		kline.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if gauge, ok := chart.(*charts.Gauge); ok {
		gauge.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if themeRiverChart, ok := chart.(*charts.ThemeRiver); ok {
		themeRiverChart.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else if sankeyChart, ok := chart.(*charts.Sankey); ok {
		sankeyChart.SetGlobalOptions(charts.WithAnimation(false)) // 關閉動畫
	} else {
		insyra.LogFatal("plot", "SavePNG", "unsupported chart type. Using default animation settings.")
	}
}

func enableAnimation(chart Renderable) {
	if barChart, ok := chart.(*charts.Bar); ok {
		barChart.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if lineChart, ok := chart.(*charts.Line); ok {
		lineChart.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if pieChart, ok := chart.(*charts.Pie); ok {
		pieChart.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if scatterChart, ok := chart.(*charts.Scatter); ok {
		scatterChart.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if heatMap, ok := chart.(*charts.HeatMap); ok {
		heatMap.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if mapChart, ok := chart.(*charts.Map); ok {
		mapChart.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if radarChart, ok := chart.(*charts.Radar); ok {
		radarChart.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if funnelChart, ok := chart.(*charts.Funnel); ok {
		funnelChart.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if liquidChart, ok := chart.(*charts.Liquid); ok {
		liquidChart.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if wordCloudChart, ok := chart.(*charts.WordCloud); ok {
		wordCloudChart.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if boxPlot, ok := chart.(*charts.BoxPlot); ok {
		boxPlot.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if kline, ok := chart.(*charts.Kline); ok {
		kline.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if gauge, ok := chart.(*charts.Gauge); ok {
		gauge.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if themeRiverChart, ok := chart.(*charts.ThemeRiver); ok {
		themeRiverChart.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else if sankeyChart, ok := chart.(*charts.Sankey); ok {
		sankeyChart.SetGlobalOptions(charts.WithAnimation(true)) // 開啟動畫
	} else {
		insyra.LogFatal("plot", "SavePNG", "unsupported chart type. Using default animation settings.")
	}
}
