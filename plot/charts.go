// charts.go
package plot

type LinePlot struct {
	*Plotter
	Options *LinePlotOptions
}

type LinePlotOptions struct {
	LineStyle LineStyle
	XLabel    string
	YLabel    string
}
