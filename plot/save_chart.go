// plot/save_chart.go

package plot

import (
	"bytes"
	"io"
	"net/http"
	"os"

	"github.com/HazelnutParadise/insyra"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-rod/rod/lib/proto"
	"github.com/luabagg/orcgen/v2"
	"github.com/luabagg/orcgen/v2/pkg/handlers/screenshot"
)

// Renderable 定義了可以被渲染的圖表接口
type Renderable interface {
	Render(w io.Writer) error
}

// SaveHTML 將圖表渲染並保存為 HTML 文件
func SaveHTML(chart Renderable, path string) {
	insyra.LogInfo("plot.SaveHTML: saving HTML file...")

	// 創建輸出文件
	f, err := os.Create(path)
	if err != nil {
		insyra.LogFatal("plot.SaveHTML: failed to create file %s: %w", path, err)
	}
	defer f.Close()

	// 渲染圖表到指定文件
	if err := chart.Render(f); err != nil {
		insyra.LogFatal("plot.SaveHTML: failed to render chart: %w", err)
	}
}

// SavePNG 將圖表渲染為 PNG 文件，使用 orcgen
func SavePNG(chart Renderable, pngPath string) {
	defer func() {
		// 使用 recover 捕捉 panic 並嘗試使用備援服務
		r := recover()
		if r != nil {
			insyra.LogWarning("plot.SavePNG: failed to render chart locally. Trying to use HazelnutParadise online service.")

			// 使用備援服務
			resp, err := http.Post("https://server3.hazelnut-paradise.com/htmltoimage", "application/x-www-form-urlencoded", nil)
			if err != nil {
				insyra.LogFatal("plot.SavePNG: failed to use online service", err)
			}
			defer resp.Body.Close()

			// 讀取備援服務返回的圖片數據
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				insyra.LogFatal("plot.SavePNG: failed to read response from online service", err)
			}

			// 將接收到的圖片數據寫入本地 PNG 文件
			err = os.WriteFile(pngPath, body, 0644)
			if err != nil {
				insyra.LogFatal("plot.SavePNG: failed to save PNG file from online service", err)
			}

			insyra.LogInfo("plot.SavePNG: successfully saved PNG file from online service.")
		}
	}()

	// Render the chart to a buffer
	var buf bytes.Buffer
	if err := chart.Render(&buf); err != nil {
		insyra.LogFatal("plot.SavePNG: failed to render chart: %w", err)
	}

	// Configure the screenshot handler
	screenshotHandler := screenshot.New().SetConfig(proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatPng,
		Quality: opts.Int(100),

		// 可根據需要設置其他配置，如 Clip、Quality、Delay 等
	})

	insyra.LogInfo("plot.SavePNG: rendering chart to PNG...")

	// 使用 orcgen 轉換 HTML 為 PNG
	fileinfo, err := orcgen.ConvertHTML(screenshotHandler, buf.Bytes())
	if err != nil {
		insyra.LogFatal("plot.SavePNG: failed to convert HTML to PNG: %w", err)
	}

	insyra.LogInfo("plot.SavePNG: saving PNG file...")

	// 保存 PNG 文件
	if err := fileinfo.Output(pngPath); err != nil {
		insyra.LogFatal("plot.SavePNG: failed to save PNG file: %w", err)
	}
}
