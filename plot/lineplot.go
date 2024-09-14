// lineplot.go - 繪製折線圖的結構體和方法

package plot

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
)

// 定義 LTGray
var LTGray = color.RGBA{R: 211, G: 211, B: 211, A: 255}

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
	LineThickness   int    // 新增線條粗細選項
	Theme           string // 新增主題選項
	FontSize        int    // 新增字體大小選項
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
		LineThickness:   1,
		Theme:           "default", // 設定預設主題
		FontSize:        16,        // 設定預設字體大小
	}

	if plotter == nil {
		plotter = NewPlotter(data, nil)
	}

	if options != nil {
		// 合併用戶提供的選項到默認選項
		if options.LineStyle != 0 {
			defaultOptions.LineStyle = options.LineStyle
		}
		if options.XLabel != "" {
			defaultOptions.XLabel = options.XLabel
		}
		if options.YLabel != "" {
			defaultOptions.YLabel = options.YLabel
		}
		if options.GridColor != nil {
			defaultOptions.GridColor = options.GridColor
		}
		if options.AxisColor != nil {
			defaultOptions.AxisColor = options.AxisColor
		}
		if options.LineColor != nil {
			defaultOptions.LineColor = options.LineColor
		}
		if options.PointColor != nil {
			defaultOptions.PointColor = options.PointColor
		}
		if options.BackgroundColor != nil {
			defaultOptions.BackgroundColor = options.BackgroundColor
		}
		if options.LineThickness != 0 {
			defaultOptions.LineThickness = options.LineThickness
		}
		if options.Theme != "" {
			defaultOptions.Theme = options.Theme
		}
		if options.FontSize != 0 {
			defaultOptions.FontSize = options.FontSize
		}

		// 根據主題調整顏色
		switch defaultOptions.Theme {
		case "dark":
			defaultOptions.BackgroundColor = color.Black
			defaultOptions.GridColor = color.Gray{Y: 128}
			defaultOptions.AxisColor = color.White
			defaultOptions.LineColor = color.RGBA{R: 0, G: 255, B: 0, A: 255}
			defaultOptions.PointColor = color.RGBA{R: 255, G: 255, B: 0, A: 255}
		case "whitegrid":
			defaultOptions.BackgroundColor = color.White
			defaultOptions.GridColor = LTGray // 使用定義的 LTGray
			// 可根據需求添加更多顏色設定
		}
	}

	return &LinePlot{
		Plotter: plotter,
		Options: defaultOptions,
	}
}

// Draw 繪製折線圖
func (lp *LinePlot) Draw() *image.RGBA {
	width := lp.Plotter.Width
	height := lp.Plotter.Height

	margin := 120     // 增加左側邊界以留更多空間給縱軸文字
	rightMargin := 50 // 新增右側邊界

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 填充背景
	draw.Draw(img, img.Bounds(), &image.Uniform{C: lp.Options.BackgroundColor}, image.Point{}, draw.Src)

	// 調整網格線和軸線的位置，留出更大的邊界空間
	lp.drawGrid(img, margin, rightMargin)
	lp.drawAxes(img, margin, rightMargin)

	// 繪製數據折線，考慮邊界
	lp.drawLine(img, margin, rightMargin)

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
func (lp *LinePlot) drawGrid(img *image.RGBA, margin int, rightMargin int) {
	width := lp.Plotter.Width - margin - rightMargin
	height := lp.Plotter.Height - margin*2
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
func (lp *LinePlot) drawAxes(img *image.RGBA, margin int, rightMargin int) {
	width := lp.Plotter.Width - margin - rightMargin
	height := lp.Plotter.Height - margin*2
	axisColor := lp.Options.AxisColor

	// 畫X軸在下方
	for x := margin; x < width+margin; x++ {
		img.Set(x, margin+height, axisColor)
	}

	// 畫Y軸在左側
	for y := margin; y < height+margin; y++ {
		img.Set(margin, y, axisColor)
	}

	// 繪製X軸標籤
	drawText(img, lp.Options.XLabel, margin+width/2-50, margin+height+50, color.Black, lp.Options.FontSize) // 新增字體大小參數

	// 繪製Y軸標籤，增加更多左側空間
	drawText(img, lp.Options.YLabel, margin-100, height/2+margin, color.Black, lp.Options.FontSize) // 新增字體大小參數

	// 繪製X軸數字標籤
	for i := 0; i <= 10; i++ {
		x := margin + i*width/10
		y := margin + height + 20                                                              // 增加與X軸的距離
		drawText(img, fmt.Sprintf("%.1f", float64(i)), x, y, color.Black, lp.Options.FontSize) // 新增字體大小參數
	}

	// 繪製Y軸數字標籤，減少左側空間
	for i := 0; i <= 10; i++ {
		y := margin + height - i*height/10
		x := margin - 40                                                                       // 從 margin-60 調整為 margin-40
		drawText(img, fmt.Sprintf("%.1f", float64(i)), x, y, color.Black, lp.Options.FontSize) // 新增字體大小參數
	}
}

// drawLine 畫折線
func (lp *LinePlot) drawLine(img *image.RGBA, margin int, rightMargin int) {
	width := lp.Plotter.Width - margin - rightMargin
	height := lp.Plotter.Height - margin*2
	lineColor := lp.Options.LineColor
	pointColor := lp.Options.PointColor

	data := lp.Plotter.data
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
