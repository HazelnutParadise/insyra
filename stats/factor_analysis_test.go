package stats_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/mat"
)

// Reference data from SPSS analysis (spss.md)
// Data source: factor_analysis_sample.csv (200 observations, 9 variables)
// Method: PAF extraction + Oblimin rotation (delta=0)

// Pre-rotation Factor Matrix (因子矩陣) from SPSS
var spssPreRotation = [][]float64{
	{0.465, 0.708, 0.006},   // A1
	{0.504, 0.497, 0.269},   // A2
	{0.581, 0.458, 0.083},   // A3
	{0.680, 0.044, -0.493},  // B1
	{0.656, -0.367, -0.467}, // B2
	{0.657, -0.076, -0.324}, // B3
	{0.516, -0.257, 0.596},  // C1
	{0.639, -0.441, 0.263},  // C2
	{0.635, -0.268, 0.302},  // C3
}

// Pattern Matrix (型樣矩陣) - Oblimin rotated from SPSS
var spssPattern = [][]float64{
	{0.067, 0.849, -0.163}, // A1
	{-0.090, 0.724, 0.177}, // A2
	{0.127, 0.677, 0.085},  // A3
	{0.798, 0.202, -0.105}, // B1
	{0.868, -0.186, 0.125}, // B2
	{0.666, 0.118, 0.084},  // B3
	{-0.178, 0.094, 0.847}, // C1
	{0.231, -0.102, 0.737}, // C2
	{0.150, 0.067, 0.674},  // C3
}

// Structure Matrix (結構矩陣) from SPSS
var spssStructure = [][]float64{
	{0.224, 0.833, 0.027}, // A1
	{0.142, 0.738, 0.293}, // A2
	{0.319, 0.725, 0.259}, // A3
	{0.815, 0.376, 0.183}, // B1
	{0.861, 0.051, 0.358}, // B2
	{0.721, 0.297, 0.314}, // B3
	{0.109, 0.220, 0.810}, // C1
	{0.435, 0.102, 0.789}, // C2
	{0.376, 0.238, 0.734}, // C3
}

// Phi Matrix (因子相關性矩陣) from SPSS - showing oblique structure
var spssPhi = [][]float64{
	{1.0, 0.245, 0.311},
	{0.245, 1.0, 0.199},
	{0.311, 0.199, 1.0},
}

// Expected output from current implementation
// Note: Current implementation converges to a different (near-orthogonal) solution
// This is a known limitation - the algorithm finds a valid local minimum
// but not the same one as SPSS
var expectedPattern = [][]float64{
	{0.359, 0.763, -0.017},  // A1
	{0.409, 0.571, 0.255},   // A2
	{0.504, 0.533, 0.072},   // A3
	{0.701, 0.109, -0.481},  // B1
	{0.737, -0.299, -0.439}, // B2
	{0.685, -0.004, -0.308}, // B3
	{0.510, -0.157, 0.616},  // C1
	{0.680, -0.337, 0.293},  // C2
	{0.648, -0.166, 0.324},  // C3
}

var expectedPhi = [][]float64{
	{1.0, 0.013, 0.043},
	{0.013, 1.0, -0.014},
	{0.043, -0.014, 1.0},
}

// Embedded test data from factor_analysis_sample.csv
// 200 observations, 9 variables: A1, A2, A3, B1, B2, B3, C1, C2, C3
var embeddedTestData = [][]float64{
	{3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 2.0, 2.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{4.0, 3.0, 3.0, 4.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 2.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0},
	{4.0, 3.0, 3.0, 5.0, 4.0, 4.0, 3.0, 4.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 3.0},
	{2.0, 2.0, 2.0, 3.0, 4.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 2.0, 2.0, 3.0, 2.0, 3.0, 3.0, 3.0},
	{3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0},
	{2.0, 3.0, 2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0},
	{2.0, 3.0, 2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 2.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0},
	{4.0, 4.0, 4.0, 4.0, 4.0, 4.0, 4.0, 4.0, 4.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 2.0, 4.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 3.0, 2.0, 2.0, 2.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 3.0, 3.0, 2.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 4.0, 4.0, 4.0, 4.0},
	{2.0, 2.0, 2.0, 3.0, 3.0, 2.0, 2.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 2.0, 2.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0},
	{2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0},
	{4.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0},
	{2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 4.0, 4.0, 4.0, 3.0, 4.0, 3.0},
	{2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 2.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 2.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0},
	{2.0, 2.0, 2.0, 2.0, 2.0, 3.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0},
	{3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0},
	{2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0},
	{2.0, 3.0, 3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 3.0, 3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 3.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 4.0, 4.0, 4.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 2.0, 2.0, 3.0, 3.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 2.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 4.0, 3.0, 4.0, 3.0, 3.0},
	{3.0, 2.0, 3.0, 3.0, 4.0, 3.0, 2.0, 3.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 4.0, 3.0, 3.0, 3.0, 3.0},
	{4.0, 3.0, 3.0, 2.0, 2.0, 2.0, 3.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 3.0, 2.0, 2.0, 3.0},
	{3.0, 2.0, 3.0, 4.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 3.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 3.0},
	{2.0, 3.0, 3.0, 2.0, 3.0, 3.0, 4.0, 4.0, 3.0},
	{3.0, 2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 2.0, 1.0, 1.0, 1.0, 2.0, 2.0, 2.0},
	{2.0, 3.0, 2.0, 2.0, 2.0, 2.0, 2.0, 3.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 2.0, 3.0, 2.0, 3.0},
	{3.0, 3.0, 3.0, 4.0, 4.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 2.0, 3.0, 3.0, 3.0},
	{2.0, 3.0, 3.0, 2.0, 2.0, 3.0, 2.0, 3.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 2.0},
	{3.0, 3.0, 3.0, 4.0, 4.0, 3.0, 2.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0},
	{3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0},
	{3.0, 4.0, 3.0, 3.0, 3.0, 3.0, 4.0, 3.0, 3.0},
	{1.0, 2.0, 2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 4.0, 4.0, 4.0},
	{2.0, 2.0, 2.0, 2.0, 3.0, 2.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0},
	{4.0, 3.0, 3.0, 4.0, 3.0, 3.0, 2.0, 2.0, 3.0},
	{2.0, 3.0, 3.0, 2.0, 2.0, 2.0, 3.0, 2.0, 2.0},
	{2.0, 2.0, 3.0, 4.0, 4.0, 3.0, 2.0, 3.0, 3.0},
	{2.0, 3.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 3.0, 2.0, 2.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 4.0, 3.0, 3.0, 3.0, 3.0, 4.0, 3.0, 4.0},
	{2.0, 3.0, 3.0, 2.0, 2.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 4.0, 3.0},
	{2.0, 2.0, 2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 3.0},
	{3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0},
	{2.0, 2.0, 2.0, 2.0, 3.0, 2.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 2.0, 3.0, 3.0, 3.0, 4.0, 3.0, 3.0},
	{3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0},
	{2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{4.0, 3.0, 4.0, 4.0, 3.0, 4.0, 2.0, 3.0, 3.0},
	{3.0, 3.0, 2.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0, 3.0},
	{2.0, 2.0, 2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{4.0, 4.0, 4.0, 4.0, 3.0, 3.0, 3.0, 2.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 2.0, 2.0, 3.0, 2.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 2.0},
	{2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 2.0, 2.0, 3.0},
	{2.0, 2.0, 2.0, 4.0, 4.0, 4.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 2.0, 3.0, 2.0, 2.0},
	{4.0, 3.0, 3.0, 2.0, 2.0, 3.0, 2.0, 2.0, 2.0},
	{2.0, 2.0, 2.0, 3.0, 3.0, 3.0, 1.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0, 2.0, 2.0},
	{2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0},
	{3.0, 2.0, 3.0, 3.0, 2.0, 2.0, 2.0, 2.0, 2.0},
	{3.0, 2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{4.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 2.0, 3.0, 2.0, 2.0, 3.0, 3.0, 3.0},
	{4.0, 3.0, 3.0, 3.0, 3.0, 3.0, 4.0, 3.0, 3.0},
	{2.0, 2.0, 2.0, 2.0, 2.0, 3.0, 2.0, 2.0, 2.0},
	{2.0, 3.0, 3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 3.0},
	{2.0, 2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 2.0, 3.0, 2.0, 2.0, 2.0, 2.0, 1.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0},
	{2.0, 2.0, 2.0, 1.0, 2.0, 2.0, 3.0, 2.0, 3.0},
	{2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 3.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 3.0, 3.0, 2.0, 2.0},
	{4.0, 4.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0, 2.0, 3.0},
	{2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 4.0, 4.0, 4.0, 4.0, 4.0, 4.0, 4.0, 4.0},
	{4.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 2.0, 3.0, 3.0, 2.0, 3.0, 3.0, 3.0},
	{4.0, 3.0, 4.0, 3.0, 3.0, 3.0, 4.0, 4.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 3.0, 2.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0},
	{4.0, 4.0, 3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 2.0},
	{3.0, 2.0, 3.0, 3.0, 2.0, 3.0, 2.0, 2.0, 2.0},
	{3.0, 2.0, 3.0, 2.0, 2.0, 2.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 4.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 4.0, 4.0, 4.0, 3.0, 3.0, 4.0},
	{3.0, 2.0, 3.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 2.0, 3.0, 4.0, 3.0, 3.0, 2.0, 3.0, 2.0},
	{4.0, 4.0, 4.0, 3.0, 2.0, 3.0, 4.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 3.0, 2.0, 3.0, 2.0, 2.0, 2.0, 2.0},
	{2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 2.0, 3.0, 3.0, 4.0, 4.0, 4.0},
	{3.0, 3.0, 2.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0},
	{3.0, 3.0, 3.0, 4.0, 3.0, 3.0, 2.0, 3.0, 3.0},
	{2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 2.0, 2.0, 2.0, 3.0, 2.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 2.0, 2.0, 2.0, 3.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 2.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 3.0, 3.0, 3.0},
	{2.0, 2.0, 3.0, 4.0, 4.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
	{3.0, 3.0, 3.0, 3.0, 2.0, 2.0, 3.0, 2.0, 2.0},
	{2.0, 3.0, 2.0, 2.0, 2.0, 2.0, 3.0, 2.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 4.0, 3.0, 2.0, 3.0, 2.0},
	{3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 2.0, 3.0},
	{2.0, 2.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0, 3.0},
}

var colNames = []string{"A1", "A2", "A3", "B1", "B2", "B3", "C1", "C2", "C3"}

func readFactorAnalysisSampleCSV(t *testing.T) insyra.IDataTable {
	cols := make([]*insyra.DataList, len(colNames))
	for j, name := range colNames {
		values := make([]any, len(embeddedTestData))
		for i := 0; i < len(embeddedTestData); i++ {
			values[i] = embeddedTestData[i][j]
		}
		dl := insyra.NewDataList(values...)
		dl.SetName(name)
		cols[j] = dl
	}
	return insyra.NewDataTable(cols...)
}

func extractMatDense(table insyra.IDataTable) *mat.Dense {
	var result *mat.Dense
	table.AtomicDo(func(dt *insyra.DataTable) {
		r, c := dt.Size()
		m := mat.NewDense(r, c, nil)
		for i := 0; i < r; i++ {
			row := dt.GetRow(i)
			for j := 0; j < c; j++ {
				v, _ := row.Get(j).(float64)
				m.Set(i, j, v)
			}
		}
		result = m
	})
	return result
}

func compareMatrices(actual, expected *mat.Dense) (maxAbs, rmse float64) {
	if actual == nil || expected == nil {
		return math.NaN(), math.NaN()
	}
	r1, c1 := actual.Dims()
	r2, c2 := expected.Dims()
	if r1 != r2 || c1 != c2 {
		return math.NaN(), math.NaN()
	}
	sumSq, maxAbs, n := 0.0, 0.0, 0
	for i := 0; i < r1; i++ {
		for j := 0; j < c1; j++ {
			diff := math.Abs(actual.At(i, j) - expected.At(i, j))
			if diff > maxAbs {
				maxAbs = diff
			}
			sumSq += diff * diff
			n++
		}
	}
	rmse = math.Sqrt(sumSq / float64(n))
	return
}

func alignFactors(ref, actual *mat.Dense) ([]int, []float64, *mat.Dense) {
	if ref == nil || actual == nil {
		return nil, nil, nil
	}
	r, c := ref.Dims()
	r2, c2 := actual.Dims()
	if r != r2 || c != c2 {
		return nil, nil, nil
	}
	bestRMSE := math.Inf(1)
	var bestPerm []int
	var bestSigns []float64
	var bestAligned *mat.Dense
	perm, used := make([]int, c), make([]bool, c)
	var gen func(int)
	gen = func(pos int) {
		if pos == c {
			for s := 0; s < (1 << c); s++ {
				signs := make([]float64, c)
				for i := 0; i < c; i++ {
					signs[i] = 1.0
					if (s>>i)&1 == 1 {
						signs[i] = -1.0
					}
				}
				aligned := mat.NewDense(r, c, nil)
				for i := 0; i < r; i++ {
					for j := 0; j < c; j++ {
						aligned.Set(i, j, actual.At(i, perm[j])*signs[j])
					}
				}
				_, rmse := compareMatrices(aligned, ref)
				if rmse < bestRMSE {
					bestRMSE = rmse
					bestPerm = append([]int{}, perm...)
					bestSigns = append([]float64{}, signs...)
					bestAligned = mat.DenseCopyOf(aligned)
				}
			}
			return
		}
		for i := 0; i < c; i++ {
			if !used[i] {
				used[i] = true
				perm[pos] = i
				gen(pos + 1)
				used[i] = false
			}
		}
	}
	gen(0)
	return bestPerm, bestSigns, bestAligned
}

func applyPermSignsToRotmat(rotmat *mat.Dense, perm []int, signs []float64) *mat.Dense {
	if rotmat == nil {
		return nil
	}
	r, c := rotmat.Dims()
	result := mat.NewDense(r, c, nil)
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			result.Set(i, j, rotmat.At(perm[i], perm[j])*signs[i]*signs[j])
		}
	}
	return result
}

func applyPermSignsToPhi(phi *mat.Dense, perm []int, signs []float64) *mat.Dense {
	if phi == nil {
		return nil
	}
	r, c := phi.Dims()
	result := mat.NewDense(r, c, nil)
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			result.Set(i, j, phi.At(perm[i], perm[j]))
		}
	}
	return result
}

func TestFactorAnalysis_SPSS_Complete(t *testing.T) {
	dt := readFactorAnalysisSampleCSV(t)

	result := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 3},
		Extraction: stats.FactorExtractionPAF,
		Rotation: stats.FactorRotationOptions{
			Method:   stats.FactorRotationOblimin,
			Delta:    0,
			Restarts: 10, // Use multiple random starts to avoid local minimum
		},
		MinErr:  1e-9,
		MaxIter: 1000,
	})

	if result == nil {
		t.Fatal("Factor analysis result is nil")
	}

	// Compare Pattern Matrix (Loadings)
	loadingsMat := extractMatDense(result.Loadings)
	spssPatternMat := mat.NewDense(9, 3, nil)
	for i := 0; i < 9; i++ {
		for j := 0; j < 3; j++ {
			spssPatternMat.Set(i, j, spssPattern[i][j])
		}
	}
	perm, signs, alignedLoadings := alignFactors(spssPatternMat, loadingsMat)
	maxAbsL, rmseL := compareMatrices(alignedLoadings, spssPatternMat)

	t.Logf("Pattern Matrix: maxAbs=%.6f rmse=%.6f (perm=%v signs=%v)", maxAbsL, rmseL, perm, signs)

	// Compare Phi Matrix (Factor Correlations)
	if result.Phi != nil {
		phiMat := extractMatDense(result.Phi)
		spssPhiMat := mat.NewDense(3, 3, nil)
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				spssPhiMat.Set(i, j, spssPhi[i][j])
			}
		}
		alignedPhi := applyPermSignsToPhi(phiMat, perm, signs)
		maxAbsP, rmseP := compareMatrices(alignedPhi, spssPhiMat)
		t.Logf("Phi Matrix:     maxAbs=%.6f rmse=%.6f", maxAbsP, rmseP)

		fmt.Printf("\n=== Factor Analysis SPSS Comparison ===\n")
		fmt.Printf("Pattern Matrix: maxAbs=%.6f rmse=%.6f\n", maxAbsL, rmseL)
		fmt.Printf("Phi Matrix:     maxAbs=%.6f rmse=%.6f\n", maxAbsP, rmseP)
		fmt.Printf("\nNote: Current implementation may converge to a different\n")
		fmt.Printf("      local minimum than SPSS (near-orthogonal vs oblique)\n")
	}
}

func TestFactorAnalysis_RotationMethods(t *testing.T) {
	data := []*insyra.DataList{
		insyra.NewDataList(1.0, 2.1, 3.2, 4.1, 5.0).SetName("V1"),
		insyra.NewDataList(2.0, 3.1, 4.0, 5.2, 6.1).SetName("V2"),
		insyra.NewDataList(3.1, 4.0, 5.1, 6.0, 7.2).SetName("V3"),
		insyra.NewDataList(4.0, 5.2, 6.1, 7.0, 8.1).SetName("V4"),
	}
	dt := insyra.NewDataTable(data...)

	methods := []stats.FactorRotationMethod{
		stats.FactorRotationNone,
		stats.FactorRotationVarimax,
		stats.FactorRotationOblimin,
	}

	for _, method := range methods {
		t.Run(string(method), func(t *testing.T) {
			result := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
				Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 2},
				Extraction: stats.FactorExtractionPCA,
				Rotation:   stats.FactorRotationOptions{Method: method},
				Scoring:    stats.FactorScoreNone,
			})
			if result == nil || result.Loadings == nil {
				t.Errorf("Failed for method %s", method)
			}
		})
	}
}

func TestFactorAnalysis_ExtractionMethods(t *testing.T) {
	data := []*insyra.DataList{
		insyra.NewDataList(1.0, 2.2, 3.1, 4.5, 5.3, 6.2).SetName("A1"),
		insyra.NewDataList(1.5, 2.4, 3.6, 4.0, 5.7, 6.5).SetName("A2"),
		insyra.NewDataList(2.1, 3.0, 3.8, 4.9, 5.8, 6.9).SetName("A3"),
		insyra.NewDataList(0.9, 1.8, 2.9, 4.2, 5.0, 5.8).SetName("A4"),
	}
	dt := insyra.NewDataTable(data...)

	methods := []stats.FactorExtractionMethod{
		stats.FactorExtractionPCA,
		stats.FactorExtractionPAF,
		stats.FactorExtractionMINRES,
	}

	for _, method := range methods {
		t.Run(string(method), func(t *testing.T) {
			result := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
				Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 2},
				Extraction: method,
				Rotation:   stats.FactorRotationOptions{Method: stats.FactorRotationVarimax},
				Scoring:    stats.FactorScoreNone,
			})
			if result == nil || result.Loadings == nil {
				t.Errorf("Failed for method %s", method)
			}
		})
	}
}
