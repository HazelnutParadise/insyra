package fa

import (
	"errors"
	"math"

	"gonum.org/v1/gonum/mat"
)

/*
等價於 GPArotation 的兩個框架：

- RotateOrth = GPForth：正交 T（TᵀT=I）
    L = A * T
    計算目標 f(L) 與對 L 的梯度 Gq(L)
    G = Aᵀ * Gq
    投影梯度：Gp = G - T * sym(TᵀG)
    用 line-search（Armijo）在 T 的 Stiefel 流形上更新：X = T - α Gp；T ← polar(X) = U Vᵀ

- RotateOblq = GPFoblq：斜交 Th（Φ = ThᵀTh）
    L = A * inv(Th)ᵀ
    計算 G = - ( (Lᵀ Gq) Th⁻¹ )ᵀ  （對應 R 版：-t(t(L) %*% Gq %*% solve(Tmat))）
    投影：Gp = G - Th * diag( sum(Th ∘ G, by row) )
    line-search：X = Th - α Gp；列向量正規化 v = 1/||X_j||，Th ← X * diag(v)
    更新 L、f、G，如此迭代

兩者都需要一個「目標 + 梯度」的 callback：
   type ObjGrad = func(L *mat.Dense) (f float64, Gq *mat.Dense)
這樣之後要擴充 oblimin(gam)、geomin、bentler… 只加函式即可。
*/

// --- pick objective for ORTHOGONAL rotation ---
func pickOrthObj(method string, opts *RotOpts) (ObjGrad, error) {
	switch lower(method) {
	case "varimax":
		return varimaxObjGrad, nil
	case "quartimax":
		return quartimaxObjGrad, nil
	case "geomin", "geomint":
		// geomin on orthogonal manifold
		eps := 1e-6
		if opts != nil && opts.GeominEps > 0 {
			eps = opts.GeominEps
		}
		return geominObjGrad(eps), nil
	case "oblimin", "quartimin":
		// oblimin can be run under orth framework too; usually used with oblique,
		// but GPArotation also exposes "geominT"/"geominQ" variants.
		gam := 0.0
		if opts != nil {
			gam = opts.Gamma
		}
		return obliminObjGrad(gam), nil
	default:
		return nil, errors.New("unsupported orthogonal method: " + method)
	}
}

// --- pick objective for OBLIQUE rotation ---
func pickOblqObj(method string, opts *RotOpts) (ObjGrad, error) {
	switch lower(method) {
	case "quartimin":
		// oblimin with gamma=0
		return obliminObjGrad(0), nil
	case "oblimin":
		gam := 0.0
		if opts != nil {
			gam = opts.Gamma
		}
		return obliminObjGrad(gam), nil
	case "geomin", "geominq":
		eps := 1e-6
		if opts != nil && opts.GeominEps > 0 {
			eps = opts.GeominEps
		}
		return geominObjGrad(eps), nil
	default:
		return nil, errors.New("unsupported oblique method: " + method)
	}
}

// ==== 正交：GPForth ====

type ObjGrad func(L *mat.Dense) (float64, *mat.Dense)

func RotateOrth(A *mat.Dense, method string, opts *RotOpts) (Lout, T *mat.Dense, err error) {
	_, q := A.Dims()
	if q < 2 {
		return nil, nil, errors.New("rotation does not make sense for single factor models")
	}
	// 預設參數
	eps := 1e-5
	maxit := 1000
	alpha := 1.0
	if opts != nil {
		if opts.Eps > 0 {
			eps = opts.Eps
		}
		if opts.MaxIter > 0 {
			maxit = opts.MaxIter
		}
		if opts.Alpha0 > 0 {
			alpha = opts.Alpha0
		}
	}

	// 目標/梯度
	var obj ObjGrad
	switch lower(method) {
	case "varimax":
		obj = varimaxObjGrad
	case "quartimax":
		obj = quartimaxObjGrad
	default:
		return nil, nil, errors.New("unsupported orthogonal method: " + method)
	}

	// 初始 T = I, L = A
	T = mat.NewDense(q, q, nil)
	for i := 0; i < q; i++ {
		T.Set(i, i, 1)
	}
	L := mat.DenseCopyOf(A)

	// 初始目標與梯度
	f, Gq := obj(L)
	// G = Aᵀ Gq
	var G mat.Dense
	G.Mul(A.T(), Gq)

	for iter := 0; iter <= maxit; iter++ {
		// 投影：Gp = G - T * sym(TᵀG)
		var M mat.Dense
		M.Mul(T.T(), &G)
		var S mat.Dense
		S.Add(&M, M.T())
		S.Scale(0.5, &S)

		var TS mat.Dense
		TS.Mul(T, &S)

		var Gp mat.Dense
		Gp.Sub(&G, &TS)

		// s = ||Gp||_F
		s := mat.Norm(&Gp, 2)
		if s < eps {
			break
		}

		// Backtracking line-search with Armijo
		// 嘗試加倍，失敗就折半（和 GPArotation 相同的 pattern）
		a := alpha
		for ls := 0; ls < 12; ls++ {
			// X = T - a * Gp，取極近的正交極分解 Tmatt = polar(X) = U Vᵀ
			var X mat.Dense
			X.Scale(-a, &Gp)
			X.Add(&X, T)

			Tmatt := polar(&X) // U Vᵀ

			// Lnew = A * Tmatt
			var Lnew mat.Dense
			Lnew.Mul(A, Tmatt)

			// fnew, Gqnew
			fnew, Gqnew := obj(&Lnew)

			// Armijo：fnew < f - 0.5 * s^2 * a
			if fnew < f-0.5*s*s*a {
				// 接受更新
				T = Tmatt
				L = &Lnew
				f = fnew
				// G = Aᵀ Gqnew
				G.Mul(A.T(), Gqnew)
				// 下次加速
				a *= 2
				break
			}
			// 否則縮小步長
			a *= 0.5

			if ls == 11 {
				// 仍不改善，接受最小步（很少見）
				T = Tmatt
				L = &Lnew
				f = fnew
				G.Mul(A.T(), Gqnew)
			}
		}
	}

	return L, T, nil
}

// ==== 斜交：GPFoblq（僅先做 quartimin，等同 oblimin(gamma=0)）====

func RotateOblq(A *mat.Dense, method string, opts *RotOpts) (Lout, Th, Phi *mat.Dense, err error) {
	_, q := A.Dims()
	if q < 2 {
		return nil, nil, nil, errors.New("rotation does not make sense for single factor models")
	}
	eps := 1e-5
	maxit := 1000
	alpha := 1.0
	if opts != nil {
		if opts.Eps > 0 {
			eps = opts.Eps
		}
		if opts.MaxIter > 0 {
			maxit = opts.MaxIter
		}
		if opts.Alpha0 > 0 {
			alpha = opts.Alpha0
		}
	}

	var obj ObjGrad
	switch lower(method) {
	case "quartimin":
		obj = quartiminObjGrad // γ=0 的 oblimin 家族
	default:
		return nil, nil, nil, errors.New("unsupported oblique method: " + method)
	}

	// 初始 Th = I
	Th = mat.NewDense(q, q, nil)
	for i := 0; i < q; i++ {
		Th.Set(i, i, 1)
	}

	// L = A * inv(Th)ᵀ = A * I = A
	L := mat.DenseCopyOf(A)

	f, Gq := obj(L)

	// G = - t( t(L) %*% Gq %*% solve(Th) )
	// 等價：G = - ( (Lᵀ Gq) Th⁻¹ )ᵀ
	var LtGq mat.Dense
	LtGq.Mul(L.T(), Gq)

	var ThInv mat.Dense
	ThInv.Inverse(Th)

	var LtGqThInv mat.Dense
	LtGqThInv.Mul(&LtGq, &ThInv)

	G := mat.NewDense(q, q, nil)
	G.CloneFrom(LtGqThInv.T())
	G.Scale(-1, G)

	for iter := 0; iter <= maxit; iter++ {
		// Gp = G - Th * diag( 1ᵀ (Th ∘ G) )    （對每一列求 sum(Th*G)，放到對角）
		diagVals := make([]float64, q)
		for i := 0; i < q; i++ {
			sum := 0.0
			for j := 0; j < q; j++ {
				sum += Th.At(i, j) * G.At(i, j)
			}
			diagVals[i] = sum
		}
		D := mat.NewDense(q, q, nil)
		for i := 0; i < q; i++ {
			D.Set(i, i, diagVals[i])
		}

		var ThD mat.Dense
		ThD.Mul(Th, D)

		var Gp mat.Dense
		Gp.Sub(G, &ThD)

		// s = ||Gp||_F
		s := mat.Norm(&Gp, 2)
		if s < eps {
			break
		}

		// line-search
		a := alpha
		for ls := 0; ls < 12; ls++ {
			// X = Th - a * Gp
			var X mat.Dense
			X.Scale(-a, &Gp)
			X.Add(&X, Th)

			// 列向量正規化：v = 1 / ||X_i||，Th_t = X * diag(v)
			v := make([]float64, q)
			for i := 0; i < q; i++ {
				n := 0.0
				for j := 0; j < q; j++ {
					n += X.At(i, j) * X.At(i, j)
				}
				if n <= 0 {
					v[i] = 1
				} else {
					v[i] = 1.0 / math.Sqrt(n)
				}
			}
			Dv := mat.NewDense(q, q, nil)
			for i := 0; i < q; i++ {
				Dv.Set(i, i, v[i])
			}

			var Tmatt mat.Dense
			Tmatt.Mul(&X, Dv)

			// Lnew = A * inv(Tmatt)ᵀ
			var TmattInv mat.Dense
			TmattInv.Inverse(&Tmatt)

			var Lnew mat.Dense
			Lnew.Mul(A, TmattInv.T())

			// fnew, Gqnew
			fnew, Gqnew := obj(&Lnew)

			// Gnew = - ( (Lnewᵀ Gqnew) Tmatt⁻¹ )ᵀ
			var LtGqNew mat.Dense
			LtGqNew.Mul(Lnew.T(), Gqnew)

			var LtGqNewTInv mat.Dense
			LtGqNewTInv.Mul(&LtGqNew, &TmattInv)

			Gnew := mat.NewDense(q, q, nil)
			Gnew.CloneFrom(LtGqNewTInv.T())
			Gnew.Scale(-1, Gnew)

			// Armijo
			if fnew < f-0.5*s*s*a {
				Th = &Tmatt
				L = &Lnew
				G = Gnew
				f = fnew
				a *= 2
				break
			}
			a *= 0.5
			if ls == 11 {
				Th = &Tmatt
				L = &Lnew
				G = Gnew
				f = fnew
			}
		}
	}

	// Phi = Thᵀ Th
	Phi = mat.NewDense(q, q, nil)
	Phi.Mul(Th.T(), Th)
	return L, Th, Phi, nil
}

// ========== 目標與梯度（ObjGrad） ==========

// varimax：對每一列 i、每一列的每一列元素：g_ij = l_ij * (l_ij^2 - mean_i(l_ij^2))
// 目標（最大化）: sum_j [ sum_i l_ij^4 - (1/p)(sum_i l_ij^2)^2 ]
func varimaxObjGrad(L *mat.Dense) (float64, *mat.Dense) {
	p, q := L.Dims()
	// col-wise sums of squares
	colSS := make([]float64, q)
	for j := 0; j < q; j++ {
		sum := 0.0
		for i := 0; i < p; i++ {
			lij := L.At(i, j)
			sum += lij * lij
		}
		colSS[j] = sum
	}

	// f
	f := 0.0
	for j := 0; j < q; j++ {
		ss := colSS[j]
		// sum_i l^4 - (1/p)(sum_i l^2)^2
		sum4 := 0.0
		for i := 0; i < p; i++ {
			lij := L.At(i, j)
			sum4 += lij * lij * lij * lij
		}
		f += sum4 - (ss*ss)/float64(p)
	}

	// Gq
	Gq := mat.NewDense(p, q, nil)
	for j := 0; j < q; j++ {
		m := colSS[j] / float64(p)
		for i := 0; i < p; i++ {
			lij := L.At(i, j)
			Gq.Set(i, j, lij*(lij*lij-m))
		}
	}
	return f, Gq
}

// quartimax：最大化 sum_{ij} l_ij^4；梯度（無常數因子影響） g_ij = l_ij^3
// quartimax：最小化 sum_j (sum_i l_ij^2)^2 / 4；梯度 g_ij = l_ij * sum_i l_ij^2
func quartimaxObjGrad(L *mat.Dense) (float64, *mat.Dense) {
	p, q := L.Dims()
	colSS := make([]float64, q) // sum_i l_ij^2 for each column j
	for j := 0; j < q; j++ {
		for i := 0; i < p; i++ {
			lij := L.At(i, j)
			colSS[j] += lij * lij
		}
	}
	f := 0.0
	for j := 0; j < q; j++ {
		f += colSS[j] * colSS[j]
	}
	f = -f / 4.0 // GPArotation minimizes this

	Gq := mat.NewDense(p, q, nil)
	for j := 0; j < q; j++ {
		sj := colSS[j]
		for i := 0; i < p; i++ {
			lij := L.At(i, j)
			Gq.Set(i, j, -lij*sj) // negative because we minimize
		}
	}
	return f, Gq
}

// quartimin（oblimin 家族的 γ=0），常見寫法：
//
//	f = 1/2 * sum_j [ (sum_i l_ij^2)^2 - sum_i l_ij^4 ]
//
// 導數：∂f/∂l_ij = 2 l_ij * S_j - 2 l_ij^3  = 2 l_ij (S_j - l_ij^2)
// 我們最大化 f（與 GPA 的方向一致），梯度直接用上式（常數因子不影響極值位置）。
func quartiminObjGrad(L *mat.Dense) (float64, *mat.Dense) {
	p, q := L.Dims()
	colSS := make([]float64, q)
	for j := 0; j < q; j++ {
		sum := 0.0
		for i := 0; i < p; i++ {
			lij := L.At(i, j)
			sum += lij * lij
		}
		colSS[j] = sum
	}
	// f
	f := 0.0
	for j := 0; j < q; j++ {
		S := colSS[j]
		sum4 := 0.0
		for i := 0; i < p; i++ {
			lij := L.At(i, j)
			sum4 += lij * lij * lij * lij
		}
		f += 0.5 * (S*S - sum4)
	}
	// Gq
	Gq := mat.NewDense(p, q, nil)
	for j := 0; j < q; j++ {
		S := colSS[j]
		for i := 0; i < p; i++ {
			lij := L.At(i, j)
			Gq.Set(i, j, 2*lij*(S-lij*lij))
		}
	}
	return f, Gq
}

// ====== 小工具 ======

func polar(X *mat.Dense) *mat.Dense {
	// X = U S Vᵀ => polar(X) = U Vᵀ
	var svd mat.SVD
	svd.Factorize(X, mat.SVDThin)
	var U, V mat.Dense
	svd.UTo(&U)
	svd.VTo(&V)
	var T mat.Dense
	T.Mul(&U, V.T())
	return &T
}

// lower case helper
func lower(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if 'A' <= r && r <= 'Z' {
			out = append(out, r+('a'-'A'))
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

// ---------- oblimin(gamma) ----------
// 參考常見寫法（與 GPArotation::vgQoblimin 一致的形式）：
// 令 c_j = sum_i l_ij^2；r_i = sum_j l_ij^2
// f = 1/2 * [ sum_j c_j^2 - (1 + gamma) * sum_{i,j} l_ij^4 + gamma * sum_i r_i^2 ]
//
//	= 1/2 * [ (1 - (1+γ)) * sum_{i,j} l_ij^4 + sum_j c_j^2 + γ * sum_i r_i^2 ]
//
// 導數：∂f/∂l_ij = l_ij * [ 2*c_j + 2*γ*r_i - 4*(1+γ)*l_ij^2 ] / 2
//
//	= l_ij * [ c_j + γ*r_i - 2*(1+γ)*l_ij^2 ]
func obliminObjGrad(gamma float64) ObjGrad {
	return func(L *mat.Dense) (float64, *mat.Dense) {
		p, q := L.Dims()
		colSS := make([]float64, q) // c_j
		rowSS := make([]float64, p) // r_i

		sum4 := 0.0
		for i := 0; i < p; i++ {
			for j := 0; j < q; j++ {
				l := L.At(i, j)
				v := l * l
				colSS[j] += v
				rowSS[i] += v
				sum4 += v * v
			}
		}
		// f
		sumColSq := 0.0
		for j := 0; j < q; j++ {
			sumColSq += colSS[j] * colSS[j]
		}
		sumRowSq := 0.0
		for i := 0; i < p; i++ {
			sumRowSq += rowSS[i] * rowSS[i]
		}
		f := 0.5 * (sumColSq - (1.0+gamma)*sum4 + gamma*sumRowSq)

		// Gq
		Gq := mat.NewDense(p, q, nil)
		for i := 0; i < p; i++ {
			ri := rowSS[i]
			for j := 0; j < q; j++ {
				l := L.At(i, j)
				cj := colSS[j]
				grad := l * (cj + gamma*ri - 2.0*(1.0+gamma)*l*l)
				Gq.Set(i, j, grad)
			}
		}
		return f, Gq
	}
}

// ---------- geomin(eps) ----------
// geomin 目標：最大化所有列/欄的幾何平均，常見實作：
//
//	f = sum_j ( (∏_i (l_ij^2 + eps))^(1/p) )
//
// 取 log 便於數值穩定：
//
//	f = sum_j exp( (1/p) * sum_i log(l_ij^2 + eps) )
//
// 導數 wrt l_ij：
//
//	∂f/∂l_ij = f_j * (1/p) * ( 2*l_ij / (l_ij^2 + eps) )
//
// 其中 f_j = exp( (1/p) * sum_i log(l_ij^2+eps) )
func geominObjGrad(eps float64) ObjGrad {
	if eps <= 0 {
		eps = 1e-6
	}
	return func(L *mat.Dense) (float64, *mat.Dense) {
		p, k := L.Dims() // p = variables, k = factors
		f := 0.0
		Gq := mat.NewDense(p, k, nil)

		// per-row (variable) stats - geometric mean across factors
		rowPro := make([]float64, p) // pro_i
		for i := 0; i < p; i++ {
			sumLog := 0.0
			for j := 0; j < k; j++ {
				l := L.At(i, j)
				sumLog += math.Log(l*l + eps)
			}
			rowPro[i] = math.Exp(sumLog / float64(k))
			f += rowPro[i]
		}

		// gradient: Gq[i,j] = (2/k) * (L[i,j] / (L[i,j]^2 + eps)) * pro[i]
		for i := 0; i < p; i++ {
			pro := rowPro[i]
			for j := 0; j < k; j++ {
				l := L.At(i, j)
				Gq.Set(i, j, (2.0/float64(k))*(l/(l*l+eps))*pro)
			}
		}
		return f, Gq
	}
}
