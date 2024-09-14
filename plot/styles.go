// styles.go

package plot

// LineStyle 定義線條樣式
type LineStyle int

const (
	Solid LineStyle = iota
	Dashed
	Dotted
)
