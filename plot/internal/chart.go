package internal

import (
	"io"

	"github.com/go-echarts/go-echarts/v2/charts"
)

type Chart[T any] interface {
	SetGlobalOptions(...charts.GlobalOpts) T
	Render(w io.Writer) error
	RenderContent() []byte
}
