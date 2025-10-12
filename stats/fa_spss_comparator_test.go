package stats_test

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"

	"gonum.org/v1/gonum/mat"
)

// Embedded SPSS target (copied from local/cmd/export/spss_target.json)
var spssTargetJSON = `{
  "loadings": [
    [0.356343, 0.761905, -0.016447],
    [0.410697, 0.569859, 0.255471],
    [0.503369, 0.533348, 0.07185],
    [0.695961, 0.112845, -0.485805],
    [0.731736, -0.294728, -0.443574],
    [0.681167, -0.001737, -0.311898],
    [0.516291, -0.160115, 0.611136],
    [0.684009, -0.33903, 0.28828],
    [0.652246, -0.167228, 0.320202]
  ],
  "rotmat": [
    [0.987413, 0.149994, 0.050166],
    [-0.132469, 0.989933, -0.049839],
    [0.017541, -0.037383, -0.999147]
  ],
  "Phi": [
    [1.0, 0.015183, 0.03841],
    [0.015183, 1.0, -0.010466],
    [0.03841, -0.010466, 1.0]
  ]
}`

type spssRef struct {
	Loadings [][]float64 `json:"loadings"`
	Rotmat   [][]float64 `json:"rotmat"`
	Phi      [][]float64 `json:"Phi"`
}

func toDense(arr [][]float64) *mat.Dense {
	r := len(arr)
	if r == 0 {
		return nil
	}
	c := len(arr[0])
	data := make([]float64, 0, r*c)
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			data = append(data, arr[i][j])
		}
	}
	return mat.NewDense(r, c, data)
}

func compareMatrices(a, b *mat.Dense) (maxAbs, rmse float64) {
	if a == nil || b == nil {
		return math.NaN(), math.NaN()
	}
	r1, c1 := a.Dims()
	r2, c2 := b.Dims()
	if r1 != r2 || c1 != c2 {
		return math.NaN(), math.NaN()
	}
	sqSum := 0.0
	maxAbs = 0.0
	n := 0
	for i := 0; i < r1; i++ {
		for j := 0; j < c1; j++ {
			d := a.At(i, j) - b.At(i, j)
			if math.Abs(d) > maxAbs {
				maxAbs = math.Abs(d)
			}
			sqSum += d * d
			n++
		}
	}
	rmse = math.Sqrt(sqSum / float64(n))
	return
}

// alignFactors finds a permutation and sign flips for actual so it best matches ref.
func alignFactors(ref, actual *mat.Dense) (perm []int, signs []float64, aligned *mat.Dense) {
	if ref == nil || actual == nil {
		return nil, nil, nil
	}
	r, c := ref.Dims()
	r2, c2 := actual.Dims()
	if r != r2 || c != c2 {
		return nil, nil, nil
	}
	nf := c

	// generate permutations
	bestRMSE := math.Inf(1)
	var bestPerm []int
	var bestSigns []float64
	var bestAligned *mat.Dense

	perm = make([]int, nf)
	used := make([]bool, nf)
	var genPerm func(int)
	genPerm = func(pos int) {
		if pos == nf {
			// for this permutation try all sign combos
			signCount := 1 << nf
			for s := 0; s < signCount; s++ {
				signs := make([]float64, nf)
				for i := 0; i < nf; i++ {
					if (s>>i)&1 == 1 {
						signs[i] = -1.0
					} else {
						signs[i] = 1.0
					}
				}
				// build aligned matrix
				alignedMat := mat.NewDense(r, nf, nil)
				for col := 0; col < nf; col++ {
					for row := 0; row < r; row++ {
						alignedMat.Set(row, col, actual.At(row, perm[col])*signs[col])
					}
				}
				_, rmse := compareMatrices(alignedMat, ref)
				if rmse < bestRMSE {
					bestRMSE = rmse
					bestPerm = append([]int(nil), perm...)
					bestSigns = append([]float64(nil), signs...)
					bestAligned = mat.DenseCopyOf(alignedMat)
				}
			}
			return
		}
		for i := 0; i < nf; i++ {
			if used[i] {
				continue
			}
			used[i] = true
			perm[pos] = i
			genPerm(pos + 1)
			used[i] = false
		}
	}
	genPerm(0)
	return bestPerm, bestSigns, bestAligned
}

func applyPermSignsToRotmat(actual *mat.Dense, perm []int, signs []float64) *mat.Dense {
	if actual == nil {
		return nil
	}
	r, c := actual.Dims()
	if r != c {
		return nil
	}
	nf := r
	out := mat.NewDense(nf, nf, nil)
	for i := 0; i < nf; i++ {
		for j := 0; j < nf; j++ {
			// newCol j comes from oldCol perm[j], and sign applied
			out.Set(i, j, actual.At(i, perm[j])*signs[j])
		}
	}
	return out
}

func applyPermSignsToPhi(actual *mat.Dense, perm []int, signs []float64) *mat.Dense {
	if actual == nil {
		return nil
	}
	r, c := actual.Dims()
	if r != c {
		return nil
	}
	nf := r
	out := mat.NewDense(nf, nf, nil)
	for i := 0; i < nf; i++ {
		for j := 0; j < nf; j++ {
			out.Set(i, j, signs[i]*signs[j]*actual.At(perm[i], perm[j]))
		}
	}
	return out
}

func TestCompareWithSpssTarget(t *testing.T) {
	// Parse reference
	var ref spssRef
	if err := json.Unmarshal([]byte(spssTargetJSON), &ref); err != nil {
		t.Fatalf("failed to parse embedded SPSS JSON: %v", err)
	}

	// Parse embedded CSV sample (this matches the SPSS reference source)
	csvData := `A1,A2,A3,B1,B2,B3,C1,C2,C3
3.0,3.0,3.0,3.0,2.0,3.0,2.0,2.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
4.0,3.0,3.0,4.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,2.0,2.0,2.0,2.0,2.0,2.0
3.0,3.0,3.0,2.0,2.0,2.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,2.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
2.0,3.0,3.0,3.0,3.0,3.0,2.0,3.0,3.0
4.0,3.0,3.0,5.0,4.0,4.0,3.0,4.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,2.0,3.0,3.0,3.0,3.0,2.0,2.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,2.0,3.0
2.0,2.0,2.0,3.0,4.0,3.0,3.0,3.0,3.0
2.0,2.0,2.0,2.0,3.0,2.0,3.0,3.0,3.0
3.0,2.0,3.0,3.0,3.0,3.0,2.0,3.0,3.0
2.0,3.0,2.0,2.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,2.0,3.0
2.0,3.0,2.0,2.0,3.0,3.0,3.0,3.0,3.0
2.0,2.0,2.0,3.0,3.0,3.0,2.0,3.0,3.0
4.0,4.0,4.0,4.0,4.0,4.0,4.0,4.0,4.0
3.0,3.0,3.0,2.0,2.0,2.0,4.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
2.0,2.0,3.0,2.0,2.0,2.0,3.0,3.0,3.0
3.0,3.0,3.0,2.0,2.0,3.0,3.0,2.0,3.0
3.0,3.0,3.0,3.0,3.0,4.0,4.0,4.0,4.0
2.0,2.0,2.0,3.0,3.0,2.0,2.0,3.0,3.0
3.0,3.0,3.0,2.0,2.0,3.0,3.0,3.0,3.0
2.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,2.0,2.0,3.0,3.0,3.0,2.0,2.0,2.0
2.0,2.0,2.0,2.0,2.0,2.0,2.0,2.0,2.0
4.0,3.0,3.0,3.0,3.0,3.0,2.0,2.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,2.0
2.0,2.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,4.0,4.0,4.0,3.0,4.0,3.0
2.0,2.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,2.0,2.0,2.0,3.0,3.0,3.0
2.0,2.0,2.0,3.0,3.0,3.0,2.0,2.0,2.0
2.0,2.0,2.0,2.0,2.0,3.0,2.0,2.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,2.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,3.0,3.0
3.0,2.0,3.0,3.0,3.0,3.0,2.0,2.0,2.0
2.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
2.0,2.0,2.0,2.0,2.0,2.0,2.0,2.0,2.0
2.0,3.0,3.0,2.0,3.0,3.0,3.0,3.0,3.0
2.0,3.0,3.0,2.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,2.0,2.0,3.0,2.0,2.0,2.0
3.0,3.0,3.0,4.0,4.0,4.0,3.0,3.0,3.0
2.0,2.0,2.0,2.0,3.0,3.0,2.0,2.0,2.0
3.0,3.0,3.0,2.0,2.0,2.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,4.0,3.0,4.0,3.0,3.0
3.0,2.0,3.0,3.0,4.0,3.0,2.0,3.0,2.0
3.0,3.0,3.0,3.0,4.0,3.0,3.0,3.0,3.0
4.0,3.0,3.0,2.0,2.0,2.0,3.0,2.0,2.0
3.0,3.0,3.0,2.0,2.0,3.0,2.0,2.0,3.0
3.0,2.0,3.0,4.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,2.0,2.0,3.0,2.0,2.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,2.0,3.0
2.0,3.0,3.0,2.0,3.0,3.0,4.0,4.0,3.0
3.0,2.0,2.0,3.0,3.0,3.0,3.0,3.0,3.0
2.0,2.0,2.0,1.0,1.0,1.0,2.0,2.0,2.0
2.0,3.0,2.0,2.0,2.0,2.0,2.0,3.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,2.0,2.0,2.0,3.0,2.0,3.0
3.0,3.0,3.0,4.0,4.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,2.0,2.0,2.0,3.0,3.0,3.0
2.0,3.0,3.0,2.0,2.0,3.0,2.0,3.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,3.0,2.0
3.0,3.0,3.0,4.0,4.0,3.0,2.0,3.0,3.0
3.0,3.0,3.0,2.0,2.0,2.0,2.0,2.0,2.0
3.0,2.0,3.0,3.0,3.0,3.0,2.0,2.0,2.0
3.0,4.0,3.0,3.0,3.0,3.0,4.0,3.0,3.0
1.0,2.0,2.0,2.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,4.0,4.0,4.0
2.0,2.0,2.0,2.0,3.0,2.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,2.0
4.0,3.0,3.0,4.0,3.0,3.0,2.0,2.0,3.0
2.0,3.0,3.0,2.0,2.0,2.0,3.0,2.0,2.0
2.0,2.0,3.0,4.0,4.0,3.0,2.0,3.0,3.0
2.0,3.0,2.0,2.0,2.0,2.0,2.0,2.0,2.0
3.0,3.0,3.0,2.0,2.0,3.0,2.0,2.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,2.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,2.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,4.0,3.0,3.0,3.0,3.0,4.0,3.0,4.0
2.0,3.0,3.0,2.0,2.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,2.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,4.0,3.0
2.0,2.0,2.0,2.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,2.0,3.0
3.0,2.0,3.0,3.0,3.0,3.0,2.0,2.0,2.0
2.0,2.0,2.0,2.0,3.0,2.0,3.0,3.0,3.0
3.0,3.0,2.0,3.0,3.0,3.0,4.0,3.0,3.0
3.0,2.0,3.0,3.0,3.0,3.0,2.0,2.0,2.0
2.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
2.0,3.0,3.0,3.0,3.0,3.0,2.0,2.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
4.0,3.0,4.0,4.0,3.0,4.0,2.0,3.0,3.0
3.0,3.0,2.0,3.0,3.0,3.0,2.0,2.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,2.0,2.0,2.0,3.0
2.0,2.0,2.0,2.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
4.0,4.0,4.0,4.0,3.0,3.0,3.0,2.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
2.0,2.0,2.0,2.0,3.0,2.0,2.0,2.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,2.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,3.0,2.0
2.0,2.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,2.0,3.0,2.0,2.0,3.0
2.0,2.0,2.0,4.0,4.0,4.0,3.0,3.0,3.0
3.0,3.0,3.0,2.0,2.0,2.0,3.0,2.0,2.0
4.0,3.0,3.0,2.0,2.0,3.0,2.0,2.0,2.0
2.0,2.0,2.0,3.0,3.0,3.0,1.0,2.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
2.0,2.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
2.0,2.0,2.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,2.0,2.0,2.0,2.0,2.0
2.0,2.0,3.0,3.0,3.0,3.0,3.0,3.0,2.0
3.0,2.0,3.0,3.0,2.0,2.0,2.0,2.0,2.0
3.0,2.0,2.0,3.0,3.0,3.0,3.0,3.0,3.0
4.0,3.0,3.0,3.0,2.0,3.0,3.0,3.0,3.0
2.0,2.0,2.0,2.0,2.0,2.0,2.0,2.0,2.0
3.0,3.0,2.0,3.0,2.0,2.0,3.0,3.0,3.0
4.0,3.0,3.0,3.0,3.0,3.0,4.0,3.0,3.0
2.0,2.0,2.0,2.0,2.0,3.0,2.0,2.0,2.0
2.0,3.0,3.0,2.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,2.0,3.0
2.0,2.0,2.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,2.0,3.0,2.0,2.0,2.0,2.0,1.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,3.0,3.0
2.0,2.0,2.0,1.0,2.0,2.0,3.0,2.0,3.0
2.0,2.0,2.0,2.0,2.0,2.0,3.0,2.0,2.0
3.0,3.0,3.0,2.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,2.0,2.0
3.0,3.0,3.0,3.0,3.0,2.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
2.0,2.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,2.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,2.0,2.0,3.0,3.0,2.0,2.0
4.0,4.0,3.0,3.0,3.0,3.0,2.0,2.0,3.0
3.0,3.0,3.0,3.0,2.0,3.0,3.0,2.0,3.0
2.0,2.0,3.0,3.0,3.0,3.0,3.0,3.0,2.0
3.0,3.0,3.0,2.0,2.0,3.0,3.0,3.0,3.0
2.0,3.0,2.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,4.0,4.0,4.0,4.0,4.0,4.0,4.0,4.0
4.0,3.0,3.0,3.0,2.0,3.0,3.0,3.0,3.0
2.0,2.0,2.0,3.0,3.0,2.0,3.0,3.0,3.0
4.0,3.0,4.0,3.0,3.0,3.0,4.0,4.0,3.0
3.0,3.0,3.0,3.0,2.0,2.0,3.0,2.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,2.0,2.0,2.0
4.0,4.0,3.0,3.0,3.0,3.0,2.0,2.0,2.0
3.0,2.0,3.0,3.0,2.0,3.0,2.0,2.0,2.0
3.0,2.0,3.0,2.0,2.0,2.0,3.0,3.0,3.0
2.0,2.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
2.0,3.0,3.0,3.0,3.0,3.0,3.0,4.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,4.0,4.0,4.0,3.0,3.0,4.0
3.0,2.0,3.0,2.0,2.0,2.0,2.0,2.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,2.0,3.0,4.0,3.0,3.0,2.0,3.0,2.0
4.0,4.0,4.0,3.0,2.0,3.0,4.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,2.0,3.0,3.0,3.0
2.0,2.0,3.0,2.0,3.0,2.0,2.0,2.0,2.0
2.0,2.0,2.0,2.0,2.0,2.0,3.0,3.0,3.0
3.0,3.0,3.0,2.0,3.0,3.0,4.0,4.0,4.0
3.0,3.0,2.0,3.0,3.0,3.0,3.0,2.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,2.0,3.0
3.0,3.0,3.0,4.0,3.0,3.0,2.0,3.0,3.0
2.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
2.0,2.0,2.0,2.0,2.0,3.0,2.0,2.0,2.0
3.0,3.0,3.0,2.0,2.0,2.0,3.0,2.0,2.0
3.0,3.0,3.0,3.0,2.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,2.0,2.0,3.0,3.0,3.0
2.0,2.0,3.0,4.0,4.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0,3.0
3.0,3.0,3.0,3.0,2.0,2.0,3.0,2.0,2.0
2.0,3.0,2.0,2.0,2.0,2.0,3.0,2.0,2.0
3.0,3.0,3.0,3.0,4.0,3.0,2.0,3.0,2.0
3.0,3.0,3.0,3.0,3.0,3.0,3.0,2.0,3.0
`

	r := csv.NewReader(strings.NewReader(csvData))
	recs, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to read embedded csv: %v", err)
	}
	if len(recs) < 2 {
		t.Fatalf("not enough rows in embedded csv")
	}
	header := recs[0]
	nVars := len(header)
	cols := make([][]float64, nVars)
	for i := 1; i < len(recs); i++ {
		row := recs[i]
		for j := 0; j < nVars; j++ {
			v, err := strconv.ParseFloat(row[j], 64)
			if err != nil {
				t.Fatalf("parse float: %v", err)
			}
			cols[j] = append(cols[j], v)
		}
	}

	data := make([]*insyra.DataList, nVars)
	for j := 0; j < nVars; j++ {
		vals := make([]any, len(cols[j]))
		for i := range cols[j] {
			vals[i] = cols[j][i]
		}
		dl := insyra.NewDataList(vals...)
		dl.SetName(header[j])
		data[j] = dl
	}

	dt := insyra.NewDataTable(data...)

	// Perform factor analysis with PAF + Oblimin to match SPSS case
	opt := stats.FactorAnalysisOptions{
		Preprocess: stats.FactorPreprocessOptions{Standardize: true},
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 3},
		Extraction: stats.FactorExtractionPAF,
		Rotation:   stats.FactorRotationOptions{Method: stats.FactorRotationOblimin, Kappa: 4, Delta: 0},
		Scoring:    stats.FactorScoreNone,
		MaxIter:    1000,
		MinErr:     1e-9,
	}

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatalf("FactorAnalysis returned nil")
	}

	// Extract matrices
	var loadingsDense, rotmatDense, phiDense *mat.Dense
	model.Loadings.AtomicDo(func(table *insyra.DataTable) {
		r, c := table.Size()
		m := mat.NewDense(r, c, nil)
		for i := 0; i < r; i++ {
			row := table.GetRow(i)
			for j := 0; j < c; j++ {
				v, _ := row.Get(j).(float64)
				m.Set(i, j, v)
			}
		}
		loadingsDense = m
	})

	if model.RotationMatrix != nil {
		model.RotationMatrix.AtomicDo(func(table *insyra.DataTable) {
			r, c := table.Size()
			m := mat.NewDense(r, c, nil)
			for i := 0; i < r; i++ {
				row := table.GetRow(i)
				for j := 0; j < c; j++ {
					v, _ := row.Get(j).(float64)
					m.Set(i, j, v)
				}
			}
			rotmatDense = m
		})
	}

	if model.Phi != nil {
		model.Phi.AtomicDo(func(table *insyra.DataTable) {
			r, c := table.Size()
			m := mat.NewDense(r, c, nil)
			for i := 0; i < r; i++ {
				row := table.GetRow(i)
				for j := 0; j < c; j++ {
					v, _ := row.Get(j).(float64)
					m.Set(i, j, v)
				}
			}
			phiDense = m
		})
	}

	refLoad := toDense(ref.Loadings)
	refRot := toDense(ref.Rotmat)
	refPhi := toDense(ref.Phi)

	// Align actual loadings to reference (permute factors and flip signs)
	perm, signs, alignedLoad := alignFactors(refLoad, loadingsDense)
	if alignedLoad == nil {
		t.Fatalf("failed to align factors")
	}

	alignedRot := applyPermSignsToRotmat(rotmatDense, perm, signs)
	alignedPhi := applyPermSignsToPhi(phiDense, perm, signs)

	maxAbsL, rmseL := compareMatrices(alignedLoad, refLoad)
	maxAbsR, rmseR := compareMatrices(alignedRot, refRot)
	maxAbsP, rmseP := compareMatrices(alignedPhi, refPhi)

	if maxAbsL > 1e-3 || rmseL > 1e-4 {
		t.Fatalf("Loadings differ from SPSS reference: maxAbs=%.6g rmse=%.6g", maxAbsL, rmseL)
	}
	if maxAbsR > 1e-3 || rmseR > 1e-4 {
		t.Fatalf("Rotmat differ from SPSS reference: maxAbs=%.6g rmse=%.6g", maxAbsR, rmseR)
	}
	if maxAbsP > 1e-3 || rmseP > 1e-4 {
		t.Fatalf("Phi differ from SPSS reference: maxAbs=%.6g rmse=%.6g", maxAbsP, rmseP)
	}
}
