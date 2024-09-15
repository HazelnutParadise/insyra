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
	Labels  []string // 新增標籤欄位
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
func NewLinePlot(plotter *Plotter, options *LinePlotOptions, labels []string) *LinePlot {
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
		plotter = NewPlotter(nil, nil) // 更新為支援空資料
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
			defaultOptions.GridColor = LTGray // 使用定義 LTGray
			// 可根據需求添加更多顏色設定
		}
	}

	return &LinePlot{
		Plotter: plotter,
		Options: defaultOptions,
		Labels:  labels, // 設定標籤
	}
}

// Draw 繪製折線圖
func (lp *LinePlot) Draw() *image.RGBA {
	width := lp.Plotter.Width
	height := lp.Plotter.Height

	margin := 150     // 增加左側邊界以留更多空間給縱軸文字
	rightMargin := 80 // 新增右側邊界

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 填充背景
	draw.Draw(img, img.Bounds(), &image.Uniform{C: lp.Options.BackgroundColor}, image.Point{}, draw.Src)

	// 計算所有資料集的全局 minY  maxY 以及 tickSpacing
	overallMinY, overallMaxY, tickSpacing := lp.calculateOverallMinMax()

	// 計算橫軸的 gridStepsX 基於數據點數量，至少為1
	gridStepsX := len(lp.Plotter.data[0])
	if gridStepsX < 1 {
		gridStepsX = 1
	}

	// 計算縱軸的 gridStepsY 基於 tickSpacing，至少為1
	gridStepsY := int(math.Round((overallMaxY - overallMinY) / tickSpacing))
	if gridStepsY < 1 {
		gridStepsY = 1
	}

	// 調整網格線軸線的位置，留更大的邊界空間
	lp.drawGrid(img, margin, rightMargin, gridStepsX, gridStepsY)
	lp.drawAxes(img, margin, rightMargin, gridStepsX, gridStepsY, overallMinY, overallMaxY, tickSpacing)

	// 繪製數據折線
	for idx, dataset := range lp.Plotter.data {
		lineColor := lp.getColor(idx)
		lp.drawSingleLine(img, dataset, margin, rightMargin, overallMinY, overallMaxY, lineColor)
	}

	if len(lp.Labels) > 0 {
		// 顯示圖例
		lp.drawLegend(img)
	}

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

// drawGrid ���網格線
func (lp *LinePlot) drawGrid(img *image.RGBA, margin int, rightMargin int, gridStepsX int, gridStepsY int) {
	width := lp.Plotter.Width - margin - rightMargin
	height := lp.Plotter.Height - margin*2
	gridColor := lp.Options.GridColor

	minY, maxY, _ := lp.calculateOverallMinMax()
	rangeY := maxY - minY
	if rangeY == 0 {
		rangeY = 1 // 防止除以零
	}

	// 畫橫向網格線
	for i := 0; i <= gridStepsY; i++ {
		yValue := minY + float64(i)*rangeY/float64(gridStepsY)
		y := margin + int(math.Round((maxY-yValue)*float64(height)/rangeY))
		for x := margin; x < width+margin; x++ {
			img.Set(x, y, gridColor)
		}
	}

	// 畫縱向網格線，基於數據點數量
	for i := 0; i <= gridStepsX; i++ {
		x := margin + int(math.Round(float64(i)*float64(width)/float64(gridStepsX)))
		for y := margin; y < height+margin; y++ {
			img.Set(x, y, gridColor)
		}
	}
}

// drawAxes 畫軸線並添加軸標
func (lp *LinePlot) drawAxes(img *image.RGBA, margin int, rightMargin int, gridStepsX int, gridStepsY int, minY, maxY float64, tickSpacing float64) {
	width := lp.Plotter.Width - margin - rightMargin
	height := lp.Plotter.Height - margin*2
	axisColor := lp.Options.AxisColor

	rangeY := maxY - minY
	if rangeY == 0 {
		rangeY = 1 // 防止除以零
	}

	// 畫X軸在���方
	for x := margin; x < width+margin; x++ {
		img.Set(x, margin+height, axisColor)
	}

	// 畫Y軸在左側
	for y := margin; y < height+margin; y++ {
		img.Set(margin, y, axisColor)
	}

	// 繪製X軸標籤
	drawText(img, lp.Options.XLabel, margin+width/2-50, margin+height+60, color.Black, lp.Options.FontSize)

	// 繪製Y軸標籤
	drawText(img, lp.Options.YLabel, margin-100, height/2+margin, color.Black, lp.Options.FontSize)

	// 繪製X軸數字標籤，基於數據點數量
	for i := 0; i < gridStepsX; i++ {
		x := margin + (i * width / gridStepsX)
		y := margin + height + 20
		drawText(img, fmt.Sprintf("%d", i+1), x, y, color.Black, lp.Options.FontSize) // 使用 i+1
	}

	// 繪製Y軸數字標籤，基於 tickSpacing
	for i := 0; i <= gridStepsY; i++ {
		yValue := minY + float64(i)*tickSpacing
		y := margin + int(math.Round((maxY-yValue)*float64(height)/rangeY))
		x := margin - 40
		// 使用整數或1位小數格式化
		if math.Mod(yValue, 1) == 0 {
			drawText(img, fmt.Sprintf("%.0f", yValue), x, y, color.Black, lp.Options.FontSize)
		} else {
			drawText(img, fmt.Sprintf("%.1f", yValue), x, y, color.Black, lp.Options.FontSize)
		}
	}
}

// drawSingleLine 繪製單一折線，使用全局 minY 和 maxY 進行縮放，並使用指定的顏色
func (lp *LinePlot) drawSingleLine(img *image.RGBA, data []float64, margin int, rightMargin int, minY, maxY float64, lineColor color.Color) {
	width := lp.Plotter.Width - margin - rightMargin
	height := lp.Plotter.Height - margin*2
	pointColor := lp.Options.PointColor

	scaleY := float64(height) / (maxY - minY)
	scaleX := float64(width) / float64(len(data)) // 修改為 len(data) 而不是 len(data)-1

	// 繪製每個數據點，從第一個點開始
	for i := 0; i < len(data); i++ {
		val := data[i]
		x := margin + int(math.Round(float64(i)*scaleX))
		y := margin + int(math.Round((maxY-val)*scaleY))

		// 畫點
		img.Set(x, y, pointColor)

		// 畫線，從第二個點開始
		if i > 0 {
			prevX := margin + int(math.Round(float64(i-1)*scaleX))
			prevY := margin + int(math.Round((maxY-data[i-1])*scaleY))
			lp.drawLineSegment(img, prevX, prevY, x, y, lineColor)
		}
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

// getColor 根據索引返回顏色
func (lp *LinePlot) getColor(index int) color.Color {
	colors := []color.Color{
		color.RGBA{R: 255, G: 0, B: 0, A: 255},
		color.RGBA{R: 0, G: 255, B: 0, A: 255},
		color.RGBA{R: 0, G: 0, B: 255, A: 255},
		// 可擴展更多顏色
	}
	if index < len(colors) {
		return colors[index]
	}
	return color.Black
}

// drawLegend 繪製圖例
func (lp *LinePlot) drawLegend(img *image.RGBA) {
	// 實作圖例繪製，根據 lp.Labels
	// ...
}

// calculateOverallMinMax 計算所有資料集的全局最小值和最大值
func (lp *LinePlot) calculateOverallMinMax() (float64, float64, float64) {
	if len(lp.Plotter.data) == 0 {
		return 0, 0, 1
	}
	minY := min(lp.Plotter.data[0])
	maxY := max(lp.Plotter.data[0])

	for _, dataset := range lp.Plotter.data[1:] {
		currentMin := min(dataset)
		currentMax := max(dataset)
		if currentMin < minY {
			minY = currentMin
		}
		if currentMax > maxY {
			maxY = currentMax
		}
	}

	niceMin, niceMax, tickSpacing := getNiceScale(minY, maxY, 10)
	return niceMin, niceMax, tickSpacing
}

// 添加 getNiceScale 函數來計算適合的軸範圍和刻度間隔
func getNiceScale(min, max float64, maxTicks int) (niceMin, niceMax, tickSpacing float64) {
	rangeY := niceNum(max-min, false)
	tickSpacing = niceNum(rangeY/float64(maxTicks-1), true)
	niceMin = math.Floor(min/tickSpacing) * tickSpacing
	niceMax = math.Ceil(max/tickSpacing) * tickSpacing

	// 增加 padding 以避免第一個點繪製在原點
	if niceMin == min {
		niceMin -= tickSpacing
	}
	if niceMax == max {
		niceMax += tickSpacing
	}
	return
}

// niceNum 根據是否向上取整來計算"好看的數字"
func niceNum(rangeY float64, round bool) float64 {
	exponent := math.Floor(math.Log10(rangeY))
	fraction := rangeY / math.Pow(10, exponent)

	var niceFraction float64
	if round {
		if fraction < 1.5 {
			niceFraction = 1
		} else if fraction < 3 {
			niceFraction = 2
		} else if fraction < 7 {
			niceFraction = 5
		} else {
			niceFraction = 10
		}
	} else {
		if fraction <= 1 {
			niceFraction = 1
		} else if fraction <= 2 {
			niceFraction = 2
		} else if fraction <= 5 {
			niceFraction = 5
		} else {
			niceFraction = 10
		}
	}

	return niceFraction * math.Pow(10, exponent)
}
