// plot/save_chart.go

package plot

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/snapshot-chromedp/render"
)

// Renderable 定義了可以被渲染的圖表接口
type Renderable interface {
	Render(w io.Writer) error
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
		insyra.LogFatal("plot.SaveHTML: failed to create file %s: %v", path, err)
	}
	defer f.Close()

	// 渲染圖表到指定文件
	if err := chart.Render(f); err != nil {
		insyra.LogFatal("plot.SaveHTML: failed to render chart: %v", err)
	}
	insyra.LogInfo("plot.SaveHTML: successfully saved HTML file.")
}

// SavePNG 將圖表渲染為 PNG 文件，使用 snapshot-chromedp
func SavePNG(chart Renderable, pngPath string) {

	dir := filepath.Dir(pngPath)
	baseName := strings.TrimSuffix(pngPath, filepath.Ext(pngPath)) // 分離主檔名和副檔名

	disableAnimation(chart)
	setBackgroundToWhite(chart)

	// 先將 Renderable 渲染為 HTML
	var buf bytes.Buffer
	if err := chart.Render(&buf); err != nil {
		insyra.LogFatal("plot.SavePNG: failed to render chart to HTML: %v", err)
		return
	}

	// 使用 snapshot-chromedp 將 HTML 渲染為 PNG，並設置高品質
	config := &render.SnapshotConfig{
		RenderContent: buf.Bytes(),
		Path:          dir,
		Suffix:        "png",
		FileName:      baseName,
		HtmlPath:      dir,
		KeepHtml:      false,
		Quality:       3, // 將圖片質量設置為 3，這裡可以根據需求進行調整
	}

	// 使用自定義配置進行渲染
	err := render.MakeSnapshot(config)
	if err != nil {
		insyra.LogWarning("plot.SavePNG: failed to save PNG: %v", err)
		goto useOnlineService
	}
	insyra.LogInfo("plot.SavePNG: successfully saved PNG file.")
	return

useOnlineService:
	func() {
		if r := recover(); r != nil {
			insyra.LogWarning("plot.SavePNG: failed to render chart locally. Trying to use HazelnutParadise online service.\nWaiting for the result...")

			// 將 Renderable 渲染成 HTML
			var buf bytes.Buffer
			if err := chart.Render(&buf); err != nil {
				insyra.LogFatal("plot.SavePNG: failed to render chart to HTML", err)
				return
			}

			// 將渲染的 HTML 放入表單數據
			formData := "html=" + url.QueryEscape(buf.String())

			// 使用備援服務發送請求
			resp, err := http.Post("https://server3.hazelnut-paradise.com/htmltoimage", "application/x-www-form-urlencoded", strings.NewReader(formData))
			if err != nil {
				insyra.LogFatal("plot.SavePNG: failed to use online service", err)
				return
			}
			defer resp.Body.Close()

			// 讀取備援服務返回的圖片數據
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				insyra.LogFatal("plot.SavePNG: failed to read response from online service", err)
				return
			}

			// 將接收到的圖片數據寫入本地 PNG 文件
			if err := os.WriteFile(pngPath, body, 0644); err != nil {
				insyra.LogFatal("plot.SavePNG: failed to save PNG file from online service", err)
				return
			}

			insyra.LogInfo("plot.SavePNG: successfully saved PNG file from hazelnut-paradise.com.")
		}
	}()
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
	} else {
		insyra.LogFatal("plot.SavePNG: unsupported chart type. Using default animation settings.")
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
	} else {
		insyra.LogFatal("plot.SavePNG: unsupported chart type. Using default animation settings.")
	}
}

func setBackgroundToWhite(chart Renderable) {
	if barChart, ok := chart.(*charts.Bar); ok {
		barChart.SetGlobalOptions(charts.WithInitializationOpts(opts.Initialization{BackgroundColor: "#FFFFFF"}))
	} else if lineChart, ok := chart.(*charts.Line); ok {
		lineChart.SetGlobalOptions(charts.WithInitializationOpts(opts.Initialization{BackgroundColor: "#FFFFFF"}))
	} else if pieChart, ok := chart.(*charts.Pie); ok {
		pieChart.SetGlobalOptions(charts.WithInitializationOpts(opts.Initialization{BackgroundColor: "#FFFFFF"}))
	} else if scatterChart, ok := chart.(*charts.Scatter); ok {
		scatterChart.SetGlobalOptions(charts.WithInitializationOpts(opts.Initialization{BackgroundColor: "#FFFFFF"}))
	} else if heatMap, ok := chart.(*charts.HeatMap); ok {
		heatMap.SetGlobalOptions(charts.WithInitializationOpts(opts.Initialization{BackgroundColor: "#FFFFFF"}))
	} else if mapChart, ok := chart.(*charts.Map); ok {
		mapChart.SetGlobalOptions(charts.WithInitializationOpts(opts.Initialization{BackgroundColor: "#FFFFFF"}))
	} else if radarChart, ok := chart.(*charts.Radar); ok {
		radarChart.SetGlobalOptions(charts.WithInitializationOpts(opts.Initialization{BackgroundColor: "#FFFFFF"}))
	} else if funnelChart, ok := chart.(*charts.Funnel); ok {
		funnelChart.SetGlobalOptions(charts.WithInitializationOpts(opts.Initialization{BackgroundColor: "#FFFFFF"}))
	} else {
		insyra.LogFatal("plot.SavePNG: unsupported chart type. Using default background color.")
	}
}
