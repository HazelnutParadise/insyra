// plot/save_chart.go

package plot

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/HazelnutParadise/insyra"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/snapshot-chromedp/render"
	"github.com/google/uuid"
)

// sanitizeChartText neutralizes characters that could break out of the
// <script>-embedded option JSON that go-echarts emits with HTML escaping
// disabled, preventing stored XSS through user-provided chart title/subtitle/
// label text. It HTML-escapes '<', '>' and '&' so a value like
// "</script><script>…" can no longer form a real tag; ECharts still renders the
// (escaped) text on the canvas.
func sanitizeChartText(s string) string {
	return html.EscapeString(s)
}

// Renderer
// Any kinds of charts have their render implementation, and
// you can define your own render logic easily.
type Renderable interface {
	Render(w io.Writer) error
	RenderContent() []byte
}

// SaveHTML 將圖表渲染並保存為 HTML 文件
func SaveHTML(chart Renderable, path string, animation ...bool) error {
	if len(animation) > 0 && !animation[0] {
		disableAnimation(chart)
	} else {
		enableAnimation(chart)
	}

	// 創建輸出文件
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	// 渲染圖表到指定文件
	if err := chart.Render(f); err != nil {
		return fmt.Errorf("failed to render chart: %w", err)
	}
	insyra.LogInfo("plot", "SaveHTML", "successfully saved HTML file in %s.", path)
	return nil
}

// SavePNG renders the chart to a PNG file with a local Chrome/Chromium.
//
// It never sends data off the host unless asked: only when useOnlineServiceOnFail
// is explicitly true does a failed local render fall back to POSTing the chart
// (including all of its data) to HazelnutParadise's online renderer at
// server3.hazelnut-paradise.com. With no argument, or false, a failed local
// render is returned as an error.
func SavePNG(chart Renderable, pngPath string, useOnlineServiceOnFail ...bool) error {
	if len(useOnlineServiceOnFail) > 1 {
		return fmt.Errorf("invalid number of arguments for useOnlineServiceOnFail; expected at most 1")
	}

	doesUseOnlineServiceOnFail := len(useOnlineServiceOnFail) > 0 && useOnlineServiceOnFail[0]

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
		return fmt.Errorf("failed to create temp html dir %s: %w", snapshotConfig.HtmlPath, mkerr)
	}
	// Ensure temp directory is removed when done (safe even if MakeSnapshot cleans up already).
	defer func() { _ = os.RemoveAll(snapshotConfig.HtmlPath) }()

	err := render.MakeSnapshot(snapshotConfig)
	if err != nil {
		if !doesUseOnlineServiceOnFail {
			return fmt.Errorf("failed to render chart to PNG (a local Chrome/Chromium is required; pass true as the third argument to allow the online fallback, which uploads the chart data): %w", err)
		}
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
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Accept", "image/png") // 指定接收 PNG 格式

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send HTTP request: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("online service returned non-OK status: %s", resp.Status)
		}
		insyra.LogInfo("plot", "SavePNG", "successfully received PNG response from HazelnutParadise online service.")
		// 讀入回應並驗證 PNG 簽章，避免把錯誤頁（如 HTML）當成圖片寫入 .png。
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read PNG response: %w", err)
		}
		pngSig := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		if len(data) < len(pngSig) || !bytes.Equal(data[:len(pngSig)], pngSig) {
			return fmt.Errorf("online service response is not a valid PNG image")
		}
		outFile, err := os.Create(pngPath)
		if err != nil {
			return fmt.Errorf("failed to create PNG file: %w", err)
		}
		defer func() { _ = outFile.Close() }()

		if _, err = outFile.Write(data); err != nil {
			return fmt.Errorf("failed to save PNG file: %w", err)
		}
	}

	insyra.LogInfo("plot", "SavePNG", "successfully saved PNG file in %s.", pngPath)
	return nil
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
		insyra.LogWarning("plot", "SavePNG", "unsupported chart type. Using default animation settings.")
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
		insyra.LogWarning("plot", "SavePNG", "unsupported chart type. Using default animation settings.")
	}
}
