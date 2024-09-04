// lineplot.go - 繪製折線圖的結構體和方法

package plot

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
)

// LinePlot 繪製折線圖的結構體
type LinePlot struct {
	*Plotter
	Options *LinePlotOptions
}

type LinePlotOptions struct {
	LineStyle       LineStyle
	XLabel          string
	YLabel          string
	GridColor       color.Color
	AxisColor       color.Color
	LineColor       color.Color
	PointColor      color.Color
	BackgroundColor color.Color
	LineThickness   int // 新增線條粗細選項
}

// NewLinePlot 創建一個新的 LinePlot
func NewLinePlot(data []float64, plotter *Plotter, options *LinePlotOptions) *LinePlot {
	defaultOptions := &LinePlotOptions{
		LineStyle:       Solid,
		XLabel:          "X-Axis",
		YLabel:          "Y-Axis",
		GridColor:       color.RGBA{R: 200, G: 200, B: 200, A: 255},
		AxisColor:       color.Black,
		LineColor:       color.RGBA{R: 255, G: 0, B: 0, A: 255},
		PointColor:      color.RGBA{R: 0, G: 0, B: 255, A: 255},
		BackgroundColor: color.White,
	}

	if plotter == nil {
		plotter = NewPlotter(data, nil)
	}

	if options != nil {
		defaultOptions = options
	}

	return &LinePlot{
		Plotter: plotter,
		Options: defaultOptions,
	}
}

// Draw 繪製折線圖
func (lp *LinePlot) Draw() *image.RGBA {
	width := lp.options.Width
	height := lp.options.Height

	margin := 50 // 設置邊界

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 填充背景
	draw.Draw(img, img.Bounds(), &image.Uniform{C: lp.Options.BackgroundColor}, image.Point{}, draw.Src)

	// 調整網格線和軸線的位置，留出邊界空間
	lp.drawGrid(img, margin)
	lp.drawAxes(img, margin)

	// 繪製數據折線，考慮邊界
	lp.drawLine(img, margin)

	return img
}

// Save 保存繪製的圖到文件
func (lp *LinePlot) Save(outputFile string) error {
	img := lp.Draw()
	file, err := os.Create(outputFile)
	if err != nil {
		panic("LinePlot.Save(): " + err.Error())
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		panic("LinePlot.Save(): " + err.Error())
	}
	return nil
}

// drawGrid 畫網格線
func (lp *LinePlot) drawGrid(img *image.RGBA, margin int) {
	width := lp.options.Width - margin*2
	height := lp.options.Height - margin*2
	gridColor := lp.Options.GridColor

	// 畫橫向網格線
	for i := 1; i <= 10; i++ {
		y := margin + i*height/10
		for x := margin; x < width+margin; x++ {
			img.Set(x, y, gridColor)
		}
	}

	// 畫縱向網格線
	for i := 1; i <= 10; i++ {
		x := margin + i*width/10
		for y := margin; y < height+margin; y++ {
			img.Set(x, y, gridColor)
		}
	}
}

// drawAxes 畫軸線並添加軸標
func (lp *LinePlot) drawAxes(img *image.RGBA, margin int) {
	width := lp.options.Width - margin*2
	height := lp.options.Height - margin*2
	axisColor := lp.Options.AxisColor

	// 畫X軸
	for x := margin; x < width+margin; x++ {
		img.Set(x, height/2+margin, axisColor)
	}

	// 畫Y軸
	for y := margin; y < height+margin; y++ {
		img.Set(width/2+margin, y, axisColor)
	}

	// 繪製X軸標籤
	drawText(img, lp.Options.XLabel, width/2+margin-50, lp.options.Height-margin/4, color.Black)

	// 繪製Y軸標籤
	drawText(img, lp.Options.YLabel, margin/4, height/2+margin, color.Black)
}

// drawLine 畫折線
func (lp *LinePlot) drawLine(img *image.RGBA, margin int) {
	width := lp.options.Width - margin*2
	height := lp.options.Height - margin*2
	lineColor := lp.Options.LineColor
	pointColor := lp.Options.PointColor

	data := lp.Plotter.data.([]float64)
	maxY := max(data)
	minY := min(data)
	scaleY := float64(height) / (maxY - minY)
	scaleX := float64(width) / float64(len(data)-1)

	prevX, prevY := 0, 0
	for i, val := range data {
		x := margin + int(float64(i)*scaleX)
		y := margin + int(float64(height)-(val-minY)*scaleY)

		// 畫點
		img.Set(x, y, pointColor)

		// 畫線
		if i > 0 {
			lp.drawLineSegment(img, prevX, prevY, x, y, lineColor)
		}

		prevX = x
		prevY = y
	}
}

// drawLineSegment 畫一段線
func (lp *LinePlot) drawLineSegment(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
	dx := math.Abs(float64(x2 - x1))
	dy := math.Abs(float64(y2 - y1))
	sx := 1
	if x1 >= x2 {
		sx = -1
	}
	sy := 1
	if y1 >= y2 {
		sy = -1
	}
	err := dx - dy

	thickness := lp.Options.LineThickness // 使用設定的線條粗細

	for {
		// 使用厚度填充線條
		for i := -thickness / 2; i <= thickness/2; i++ {
			img.Set(x1+i, y1, c)
			img.Set(x1, y1+i, c)
		}
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := err * 2
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// max 返回數組中的最大值
func max(data []float64) float64 {
	maxVal := data[0]
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

// min 返回數組中的最小值
func min(data []float64) float64 {
	minVal := data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}
