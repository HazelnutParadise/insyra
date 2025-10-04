// fa/rotate_router.go
package fa

import (
	"errors"
	"strings"

	"gonum.org/v1/gonum/mat"
)

// 若你已經在其他檔案定義 RotOpts，保留一份即可。
// 這裡只加入 PromaxPower 參數，供 RouteRotation/Promax 使用。
type RotOpts struct {
	Eps         float64 // 收斂門檻（梯度投影範數）
	MaxIter     int     // 最大迭代
	Alpha0      float64 // 初始步長
	Verbose     bool
	Gamma       float64 // for oblimin
	GeominEps   float64 // for geomin
	PromaxPower int     // for Promax（預設 4）
}

// Rotate 是統一入口，模仿 psych::faRotations 字串介面。
// 支援：
//
//	"none"
//	"varimax", "quartimax", "geomin", "geominT"
//	"oblimin", "quartimin", "geominQ"
//	"promax", "Promax"
//
// 回傳：L（載荷）、rotmat（正交為 T，斜交為 Th，Promax 為 R）、Phi（正交為 I）
func Rotate(A *mat.Dense, method string, opts *RotOpts) (L, rotmat, Phi *mat.Dense, err error) {
	m := normMethod(method)
	switch m {
	case "none":
		// 不旋轉：rotmat = I, Phi = I
		_, q := A.Dims()
		L = mat.DenseCopyOf(A)
		rotmat = eye(q)
		Phi = eye(q)
		return L, rotmat, Phi, nil

	// --- 正交家族 ---
	case "varimax", "quartimax", "geomint", "geomin", "geomin_t":
		if opts == nil {
			opts = &RotOpts{}
		}
		if opts.GeominEps <= 0 {
			opts.GeominEps = 1e-6
		}
		// small aliases
		if m == "geomin_t" || m == "geomint" {
			m = "geomin"
		}
		L, T, err := RotateOrth(A, m, opts)
		if err != nil {
			return nil, nil, nil, err
		}
		Phi = eye(T.RawMatrix().Rows) // 正交：Phi=I
		return L, T, Phi, nil

	// --- 斜交家族 ---
	case "oblimin", "quartimin", "geominq", "geomin_q":
		if opts == nil {
			opts = &RotOpts{}
		}
		if opts.GeominEps <= 0 {
			opts.GeominEps = 1e-6
		}
		if m == "geomin_q" {
			m = "geominq"
		}
		L, Th, Phi, err := RotateOblq(A, m, opts)
		return L, Th, Phi, err

	// --- Promax ---
	case "promax":
		if opts == nil {
			opts = &RotOpts{}
		}
		pow := opts.PromaxPower
		if pow <= 1 {
			pow = 4
		}
		L, R, Phi, err := Promax(A, pow, false)
		return L, R, Phi, err

	default:
		return nil, nil, nil, errors.New("Rotate: unsupported method: " + method)
	}
}

// ----------------- helpers -----------------

// 正規化字串（處理大小寫、T/Q 尾碼）
func normMethod(s string) string {
	x := lower(strings.TrimSpace(s))
	switch x {
	case "none":
		return "none"
	case "varimax":
		return "varimax"
	case "quartimax":
		return "quartimax"
	case "geomint", "geomin", "geomintt", "geomin_t":
		return "geomin_t" // orth 版
	case "geominq", "geomin_q":
		return "geominq" // oblique 版
	case "oblimin":
		return "oblimin"
	case "quartimin":
		return "quartimin"
	case "promax":
		return "promax"
	default:
		// 嘗試把尾碼 T/Q 轉成內部鍵
		if strings.HasPrefix(x, "geomin") {
			if strings.HasSuffix(x, "q") {
				return "geominq"
			}
			return "geomin_t"
		}
		return x
	}
}

// 單位矩陣
func eye(n int) *mat.Dense {
	I := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		I.Set(i, i, 1)
	}
	return I
}
