// plotter.go - 定義 Plotter 結構體和相關函數

package plot

type Plotter struct {
	data   [][]float64 // 修改為二維切片以支援多組資料
	Width  int
	Height int
}

type PlotterOptions struct {
	Width  int
	Height int
}

// 修改 NewPlotter，使其支援初始化時無資料
func NewPlotter(data [][]float64, options *PlotterOptions) *Plotter {
	defaultOptions := &PlotterOptions{
		Width:  1920,
		Height: 1080,
	}

	if options != nil {
		defaultOptions = options
	}

	return &Plotter{
		data:   data,
		Width:  defaultOptions.Width,
		Height: defaultOptions.Height,
	}
}
