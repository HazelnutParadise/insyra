// fa/GPArotation_vgQ_simplimax.go
package fa

import (
	"math"
	"sort"

	"gonum.org/v1/gonum/mat"
)

// vgQSimplimax is GPArotation's vgQ.simplimax(L, k): the criterion adds up
// exactly the k smallest squared loadings, taken in the order order(L^2)
// gives them, and the gradient is 2·L on those entries and zero elsewhere.
//
//	L2 <- L^2
//	Imat[order(L2)[1:k]] <- TRUE
//	Gq <- 2 * Imat * L
//	f <- sum(L2[Imat])
//
// order() is stable and walks a matrix column by column, so equal squared
// loadings are taken by column-major position, and a NaN sorts last as NA
// does. The sum runs in column-major order, as R's does.
func vgQSimplimax(L *mat.Dense, k int) (Gq *mat.Dense, f float64, method string) {
	rows, cols := L.Dims()
	n := rows * cols

	l2 := make([]float64, n)
	for j := 0; j < cols; j++ {
		for i := 0; i < rows; i++ {
			v := L.At(i, j)
			l2[j*rows+i] = v * v
		}
	}

	order := make([]int, n)
	for idx := range order {
		order[idx] = idx
	}
	sort.SliceStable(order, func(a, b int) bool {
		x, y := l2[order[a]], l2[order[b]]
		if math.IsNaN(x) {
			return false
		}
		return math.IsNaN(y) || x < y
	})

	selected := make([]bool, n)
	for _, idx := range order[:max(0, min(k, n))] {
		selected[idx] = true
	}

	Gq = mat.NewDense(rows, cols, nil)
	for idx, in := range selected {
		if !in {
			continue
		}
		i, j := idx%rows, idx/rows
		Gq.Set(i, j, 2*L.At(i, j))
		f += l2[idx]
	}

	method = "Simplimax"
	return
}

// simplimaxK is GPArotation::simplimax's default k, which psych 2.6.5 never
// overrides: the number of variables times one less than the number of
// factors, the loadings a simple structure leaves near zero.
func simplimaxK(L *mat.Dense) int {
	rows, cols := L.Dims()
	return rows * (cols - 1)
}
