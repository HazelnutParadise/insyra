// charts.go
package plot

type LinePlot struct {
	plotter         *Plotter
	linePlotOptions *LinePlotOptions
}

type LinePlotOptions struct {
	GerenalOptions *GeneralOptions
	LineStyle      LineStyle
	XLabel         string
	YLabel         string
}
