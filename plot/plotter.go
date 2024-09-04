// plotter.go

package plot

import (
	"image/color"
	"os"
)

// Plotter is a struct that contains the data to be plotted
type Plotter struct {
	data    interface{}     // 要繪製的數據
	options *GeneralOptions // 圖表選項
}

type GeneralOptions struct {
	Title           string      // 圖表標題
	Width           int         // 圖表寬度
	Height          int         // 圖表高度
	BackgroundColor color.Color // 背景顏色
}

// Plot is an interface that defines the methods for plotting
type Plot interface {
	Draw()
	Save(outputFile string)
}

// NewPlotter creates a new Plotter instance
func NewPlotter(data interface{}, options *GeneralOptions) *Plotter {
	defaultOptions := &GeneralOptions{
		Title:           "Insyra Plot",
		Width:           800,
		Height:          600,
		BackgroundColor: color.White,
	}
	if options != nil {
		defaultOptions = options
	}
	return &Plotter{
		data:    data,
		options: defaultOptions,
	}
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
