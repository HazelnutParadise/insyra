package insyra

type Config struct {
	DefaultPlotType string
	Precision       int
}

var DefaultConfig = Config{
	DefaultPlotType: "line",
	Precision:       2,
}
