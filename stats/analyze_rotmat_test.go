package stats_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestAnalyzeRotmat(t *testing.T) {
	// SPSS target
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

	var ref spssRef
	json.Unmarshal([]byte(spssTargetJSON), &ref)

	rotmat := mat.NewDense(3, 3, nil)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			rotmat.Set(i, j, ref.Rotmat[i][j])
		}
	}

	phi := mat.NewDense(3, 3, nil)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			phi.Set(i, j, ref.Phi[i][j])
		}
	}

	// Test 1: rotmat^T * rotmat = ?
	var rtR mat.Dense
	rtR.Mul(rotmat.T(), rotmat)
	fmt.Println("=== rotmat^T * rotmat ===")
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			fmt.Printf("%10.6f ", rtR.At(i, j))
		}
		fmt.Println()
	}

	// Test 2: rotmat * rotmat^T = ?
	var rRt mat.Dense
	rRt.Mul(rotmat, rotmat.T())
	fmt.Println("\n=== rotmat * rotmat^T ===")
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			fmt.Printf("%10.6f ", rRt.At(i, j))
		}
		fmt.Println()
	}

	// Test 3: rotmat^T * Phi * rotmat = ?
	var tmp, result mat.Dense
	tmp.Mul(rotmat.T(), phi)
	result.Mul(&tmp, rotmat)
	fmt.Println("\n=== rotmat^T * Phi * rotmat ===")
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			fmt.Printf("%10.6f ", result.At(i, j))
		}
		fmt.Println()
	}

	// Test 4: rotmat * Phi * rotmat^T = ?
	var tmp2, result2 mat.Dense
	tmp2.Mul(rotmat, phi)
	result2.Mul(&tmp2, rotmat.T())
	fmt.Println("\n=== rotmat * Phi * rotmat^T ===")
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			fmt.Printf("%10.6f ", result2.At(i, j))
		}
		fmt.Println()
	}

	// Test 5: Phi compared to identity
	fmt.Println("\n=== Phi ===")
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			fmt.Printf("%10.6f ", phi.At(i, j))
		}
		fmt.Println()
	}

	// Test 6: inv(rotmat)
	var invR mat.Dense
	err := invR.Inverse(rotmat)
	if err == nil {
		fmt.Println("\n=== inv(rotmat) ===")
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				fmt.Printf("%10.6f ", invR.At(i, j))
			}
			fmt.Println()
		}

		// Test 7: t(inv(rotmat))
		fmt.Println("\n=== t(inv(rotmat)) ===")
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				fmt.Printf("%10.6f ", invR.At(j, i))
			}
			fmt.Println()
		}
	}
}
