package plot

import (
	"image/color"
	"os"
)

// Plotter is a struct that contains the data to be plotted
type Plotter struct {
	data    interface{} // 要繪製的數據
	options Options     // 圖表選項
}

type Options struct {
	Title           string // 圖表標題
	XLabel          string // X軸標籤
	YLabel          string // Y軸標籤
	LineStyle       LineStyle
	Width           int         // 圖表寬度
	Height          int         // 圖表高度
	BackgroundColor color.Color // 背景顏色
}

type LineStyle int

const (
	Solid   LineStyle = iota // 實線
	Dashed                   // 虛線
	Dotted                   // 點線
	DashDot                  // 虛點線
)

// NewPlotter creates a new Plotter instance
func NewPlotter(data interface{}, options *Options) *Plotter {
	defaultOptions := Options{
		Title:           "Insyra Plot",
		XLabel:          "X Axis",
		YLabel:          "Y Axis",
		Style:           StyleLine, // 默認為折線圖
		Width:           800,
		Height:          600,
		BackgroundColor: color.White,
	}
	if options != nil {
		defaultOptions = *options
	}
	return &Plotter{
		data:    data,
		options: defaultOptions,
	}
}

// LinePlot draws a line plot
func (p *Plotter) LinePlot(outputFile string) error {
	// 這裡實現折線圖繪製的邏輯
	// 使用第三方包或者原生的 Go 庫來處理圖形繪製
	return nil
}

// BarPlot draws a bar plot
func (p *Plotter) BarPlot(outputFile string) error {
	// 這裡實現柱狀圖繪製的邏輯
	return nil
}

// SavePlot 將圖形保存到文件中
func (p *Plotter) SavePlot(outputFile string) error {
	// 使用原生的 Go 庫或者第三方包來保存圖像
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()
	// 繪製完成的圖像寫入文件中
	return nil
}
