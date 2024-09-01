// styles.go

package plot

type LineStyle int

const (
	Solid   LineStyle = iota // 實線
	Dashed                   // 虛線
	Dotted                   // 點線
	DashDot                  // 虛點線
)
