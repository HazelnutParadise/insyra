// charts.go
package plot

import (
	"os"
)

type LinePlot struct {
	*Plotter
	Options *LinePlotOptions
}

type LinePlotOptions struct {
	LineStyle LineStyle
	XLabel    string
	YLabel    string
}

// Draw 繪製折線圖，如果遇到錯誤則使用 panic
func (lp *LinePlot) Draw() {
	// 這裡是實際繪製折線圖的邏輯，假設使用了原生 Go 的 image 包
	// img := image.NewRGBA(image.Rect(0, 0, lp.Options.Width, lp.Options.Height))
	// draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)

	// 如果在繪圖過程中發生錯誤，使用 panic 終止並報告錯誤
	// if err := lp.someInternalDrawingFunction(img); err != nil {
	// 	panic("LinePlot.Draw(): " + err.Error())
	// }

	// 其他繪圖邏輯...
}

// Save 將圖表保存到文件，如果遇到錯誤則使用 panic
func (lp *LinePlot) Save(outputFile string) {
	// 使用原生 Go 庫保存圖像到文件
	file, err := os.Create(outputFile)
	if err != nil {
		panic("LinePlot.Save(): " + err.Error())
	}
	defer file.Close()

	// 假設這裡有保存圖像的邏輯
	// if err := lp.someInternalSaveFunction(file); err != nil {
	// 	panic("LinePlot.Save(): " + err.Error())
	// }
}
