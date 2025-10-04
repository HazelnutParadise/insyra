package fa

import (
	"math"
	"sort"

	"gonum.org/v1/gonum/mat"
)

// ApplyRotation 將載荷矩陣套用旋轉（包含因子重排與 Phi 修正）。
// 對應 psych::faRotations() + psych::fa() 的旋轉階段。
func ApplyRotation(loadings *mat.Dense, method string, opts *RotOpts) (
	Lrot, rotmat, Phi *mat.Dense, order []int, err error) {

	p, q := loadings.Dims()

	// 1️⃣ 呼叫 Rotate()
	Lrot, rotmat, Phi, err = Rotate(loadings, method, opts)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// 2️⃣ 計算每個因子的總平方載荷（sum of squares）
	ss := make([]float64, q)
	for j := 0; j < q; j++ {
		s := 0.0
		for i := 0; i < p; i++ {
			v := Lrot.At(i, j)
			s += v * v
		}
		ss[j] = s
	}

	// 3️⃣ 依 psych 規則排序因子（由大到小）
	order = make([]int, q)
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool {
		return ss[order[i]] > ss[order[j]]
	})

	// 4️⃣ 重新排序載荷與 rotmat、Phi
	Lrot = reorderCols(Lrot, order)
	rotmat = reorderCols(rotmat, order)
	if Phi != nil && Phi.RawMatrix().Rows == q {
		Phi = reorderSym(Phi, order)
	}

	// 5️⃣ 若是斜交，重算 Phi（確保數值穩定）
	if Phi != nil && !isIdentity(Phi, 1e-12) {
		// Phi = rotmat^{-1} rotmat^{-T}
		var inv mat.Dense
		inv.Inverse(rotmat)
		var tmp mat.Dense
		tmp.Mul(inv.T(), &inv)
		Phi = mat.DenseCopyOf(&tmp)
	}

	return Lrot, rotmat, Phi, order, nil
}

// ---- helpers ----

// 重排矩陣的欄順序
func reorderCols(X *mat.Dense, order []int) *mat.Dense {
	r, c := X.Dims()
	Y := mat.NewDense(r, c, nil)
	for j, idx := range order {
		for i := 0; i < r; i++ {
			Y.Set(i, j, X.At(i, idx))
		}
	}
	return Y
}

// 重排對稱矩陣
func reorderSym(X *mat.Dense, order []int) *mat.Dense {
	n, _ := X.Dims()
	Y := mat.NewDense(n, n, nil)
	for i, oi := range order {
		for j, oj := range order {
			Y.Set(i, j, X.At(oi, oj))
		}
	}
	return Y
}

// 判斷是否接近單位矩陣
func isIdentity(X *mat.Dense, tol float64) bool {
	n, _ := X.Dims()
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			v := X.At(i, j)
			if i == j {
				if math.Abs(v-1) > tol {
					return false
				}
			} else {
				if math.Abs(v) > tol {
					return false
				}
			}
		}
	}
	return true
}
