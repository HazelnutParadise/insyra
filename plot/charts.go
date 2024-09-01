// charts.go
package plot

type LinePlot struct {
	*Plotter
	Options *LinePlotOptions
}

type LinePlotOptions struct {
	GerenalOptions *GeneralOptions
	LineStyle      LineStyle
	XLabel         string
	YLabel         string
}
