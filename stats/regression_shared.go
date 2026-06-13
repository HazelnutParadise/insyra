package stats

import (
	"fmt"
	"sync"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats/internal/parutil"
	"gonum.org/v1/gonum/mat"
)

func gatherRegressionInputs(dlY insyra.IDataList, dlXs []insyra.IDataList, extra ...insyra.IDataList) (y []float64, xs [][]float64, extras [][]float64, n int, err error) {
	if dlY == nil {
		return nil, nil, nil, 0, fmt.Errorf("y data list is nil")
	}
	for j, dlX := range dlXs {
		if dlX == nil {
			return nil, nil, nil, 0, fmt.Errorf("predictor %d data list is nil", j)
		}
	}
	for j, dlExtra := range extra {
		if dlExtra == nil {
			return nil, nil, nil, 0, fmt.Errorf("extra data list %d is nil", j)
		}
	}

	xs = make([][]float64, len(dlXs))
	xLens := make([]int, len(dlXs))
	extras = make([][]float64, len(extra))
	extraLens := make([]int, len(extra))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		dlY.AtomicDo(func(dly *insyra.DataList) {
			n = dly.Len()
			y = dly.ToF64Slice()
		})
	}()
	for j, dlX := range dlXs {
		wg.Add(1)
		go func(j int, dlX insyra.IDataList) {
			defer wg.Done()
			dlX.AtomicDo(func(l *insyra.DataList) {
				xLens[j] = l.Len()
				xs[j] = l.ToF64Slice()
			})
		}(j, dlX)
	}
	for j, dlExtra := range extra {
		wg.Add(1)
		go func(j int, dlExtra insyra.IDataList) {
			defer wg.Done()
			dlExtra.AtomicDo(func(l *insyra.DataList) {
				extraLens[j] = l.Len()
				extras[j] = l.ToF64Slice()
			})
		}(j, dlExtra)
	}
	wg.Wait()

	for j, xLen := range xLens {
		if xLen != n {
			return nil, nil, nil, 0, fmt.Errorf("x and y must have the same length for predictor %d", j)
		}
	}
	for j, extraLen := range extraLens {
		if extraLen != n {
			return nil, nil, nil, 0, fmt.Errorf("extra data list %d and y must have the same length", j)
		}
	}
	return y, xs, extras, n, nil
}

func buildDesignMatrix(xs [][]float64, n int) *mat.Dense {
	p := len(xs)
	X := mat.NewDense(n, p+1, nil)
	xRaw := X.RawMatrix()
	stride := xRaw.Stride
	parutil.Run(n, n*(p+1) >= 50_000, func(i int) {
		base := i * stride
		xRaw.Data[base] = 1.0
		for j := range p {
			xRaw.Data[base+j+1] = xs[j][i]
		}
	})
	return X
}
