package stats_test

import (
	"fmt"
	"math"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/mat"
)

func TestDebugPromaxTransformation(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dataPath := filepath.Join(filepath.Dir(thisFile), "..", "local", "fa_test_dataset.csv")

	dt, err := insyra.ReadCSV(dataPath, false, true)
	if err != nil {
		t.Fatalf("failed to load dataset: %v", err)
	}

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Extraction = stats.FactorExtractionPAF
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Rotation.Method = stats.FactorRotationPromax
	opt.Rotation.Kappa = 4
	opt.Scoring = stats.FactorScoreRegression
	opt.MaxIter = 200

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatalf("model nil")
	}

	loadings := toMatrix(model.Loadings)
	fmt.Printf("Loadings:\n%v\n", loadings)
	if model.Phi != nil {
		phiMatrix := toMatrix(model.Phi)
		fmt.Printf("Model phi:\n%v\n", phiMatrix)
		loadMat := mat.NewDense(len(loadings), len(loadings[0]), nil)
		for i := 0; i < len(loadings); i++ {
			for j := 0; j < len(loadings[0]); j++ {
				loadMat.Set(i, j, loadings[i][j])
			}
		}
		phiMat := mat.NewDense(len(phiMatrix), len(phiMatrix[0]), nil)
		for i := 0; i < len(phiMatrix); i++ {
			for j := 0; j < len(phiMatrix[0]); j++ {
				phiMat.Set(i, j, phiMatrix[i][j])
			}
		}
		var structure mat.Dense
		structure.Mul(loadMat, phiMat)
		fmt.Printf("Model structure:\n%v\n", mat.Formatted(&structure, mat.Prefix(" "), mat.Squeeze()))
	}
	if model.RotationMatrix != nil {
		rotMatrix := toMatrix(model.RotationMatrix)
		fmt.Printf("Model rotation:\n%v\n", rotMatrix)
		rotMat := mat.NewDense(len(rotMatrix), len(rotMatrix[0]), nil)
		for i := 0; i < len(rotMatrix); i++ {
			for j := 0; j < len(rotMatrix[0]); j++ {
				rotMat.Set(i, j, rotMatrix[i][j])
			}
		}
		loadMat := mat.NewDense(len(loadings), len(loadings[0]), nil)
		for i := 0; i < len(loadings); i++ {
			for j := 0; j < len(loadings[0]); j++ {
				loadMat.Set(i, j, loadings[i][j])
			}
		}
		var loadingsTimesRotation mat.Dense
		loadingsTimesRotation.Mul(loadMat, rotMat)
		fmt.Printf("Loadings * rotation:\n%v\n", mat.Formatted(&loadingsTimesRotation, mat.Prefix(" "), mat.Squeeze()))
	}

	optNoRot := opt
	optNoRot.Rotation.Method = stats.FactorRotationNone
	modelNoRot := stats.FactorAnalysis(dt, optNoRot)
	if modelNoRot != nil {
		fmt.Printf("Unrotated loadings:\n%v\n", toMatrix(modelNoRot.Loadings))
	}

	optVarimax := opt
	optVarimax.Rotation.Method = stats.FactorRotationVarimax
	modelVarimax := stats.FactorAnalysis(dt, optVarimax)
	if modelVarimax != nil {
		fmt.Printf("Varimax loadings:\n%v\n", toMatrix(modelVarimax.Loadings))
	}

	var transManual mat.Dense
	transManualReady := false
	if modelVarimax != nil {
		vLoadings := toMatrix(modelVarimax.Loadings)
		rows := len(vLoadings)
		cols := 0
		if rows > 0 {
			cols = len(vLoadings[0])
		}
		vMat := mat.NewDense(rows, cols, nil)
		for i := 0; i < rows; i++ {
			for j := 0; j < cols; j++ {
				vMat.Set(i, j, vLoadings[i][j])
			}
		}

		weights := make([]float64, rows)
		working := mat.NewDense(rows, cols, nil)
		for i := 0; i < rows; i++ {
			sum := 0.0
			for j := 0; j < cols; j++ {
				val := vMat.At(i, j)
				sum += val * val
			}
			w := math.Sqrt(sum)
			if w <= 1e-12 {
				w = 1e-6
			}
			weights[i] = w
			for j := 0; j < cols; j++ {
				working.Set(i, j, vMat.At(i, j)/w)
			}
		}

		target := mat.NewDense(rows, cols, nil)
		for i := 0; i < rows; i++ {
			for j := 0; j < cols; j++ {
				val := working.At(i, j)
				sign := 1.0
				if val < 0 {
					sign = -1.0
				} else if val == 0 {
					sign = 0
				}
				target.Set(i, j, sign*math.Pow(math.Abs(val), opt.Rotation.Kappa))
			}
		}

		var ft mat.Dense
		ft.CloneFrom(working.T())
		var ftf mat.Dense
		ftf.Mul(&ft, working)
		var ftfInv mat.Dense
		if err := ftfInv.Inverse(&ftf); err != nil {
			var ftfReg mat.Dense
			ftfReg.CloneFrom(&ftf)
			for i := 0; i < cols; i++ {
				ftfReg.Set(i, i, ftfReg.At(i, i)+1e-6)
			}
			_ = ftfInv.Inverse(&ftfReg)
		}

		var ftTarget mat.Dense
		ftTarget.Mul(&ft, target)
		transManualReady = true
		transManual.Mul(&ftfInv, &ftTarget)

		var transTtrans mat.Dense
		transTtrans.Mul(transManual.T(), &transManual)
		for j := 0; j < cols; j++ {
			diag := transTtrans.At(j, j)
			if diag <= 1e-12 {
				continue
			}
			scale := math.Sqrt(diag)
			if scale == 0 {
				continue
			}
			for i := 0; i < cols; i++ {
				transManual.Set(i, j, transManual.At(i, j)/scale)
			}
		}

		var rotatedNorm mat.Dense
		rotatedNorm.Mul(working, &transManual)
		fmt.Printf("Manual trans:\n%v\n", mat.Formatted(&transManual, mat.Prefix(" "), mat.Squeeze()))
		fmt.Printf("Manual rotated (norm):\n%v\n", mat.Formatted(&rotatedNorm, mat.Prefix(" "), mat.Squeeze()))
		manual := mat.NewDense(rows, cols, nil)
		for i := 0; i < rows; i++ {
			w := weights[i]
			for j := 0; j < cols; j++ {
				manual.Set(i, j, rotatedNorm.At(i, j)*w)
			}
		}
		fmt.Printf("Manual rotated (denorm):\n%v\n", mat.Formatted(manual, mat.Prefix(" "), mat.Squeeze()))
		var transManualInv mat.Dense
		if err := transManualInv.Inverse(&transManual); err == nil {
			var transManualInvT mat.Dense
			transManualInvT.CloneFrom(transManualInv.T())
			var phiManual mat.Dense
			phiManual.Mul(&transManualInv, &transManualInvT)
			phiManualNorm := normalizeDense(&phiManual)
			fmt.Printf("Manual phi norm:\n%v\n", mat.Formatted(phiManualNorm, mat.Prefix(" "), mat.Squeeze()))
			var phiInv mat.Dense
			if err := phiInv.Inverse(phiManualNorm); err == nil {
				var loadingsPhiInv mat.Dense
				loadingsPhiInv.Mul(manual, &phiInv)
				fmt.Printf("Manual loadings * phi^{-1}:\n%v\n", mat.Formatted(&loadingsPhiInv, mat.Prefix(" "), mat.Squeeze()))
			}
		}

		var ftNoNorm mat.Dense
		ftNoNorm.CloneFrom(vMat.T())
		var ftfNoNorm mat.Dense
		ftfNoNorm.Mul(&ftNoNorm, vMat)
		var ftfNoNormInv mat.Dense
		if err := ftfNoNormInv.Inverse(&ftfNoNorm); err != nil {
			var ftfReg mat.Dense
			ftfReg.CloneFrom(&ftfNoNorm)
			for i := 0; i < cols; i++ {
				ftfReg.Set(i, i, ftfReg.At(i, i)+1e-6)
			}
			_ = ftfNoNormInv.Inverse(&ftfReg)
		}

		targetNoNorm := mat.NewDense(rows, cols, nil)
		for i := 0; i < rows; i++ {
			for j := 0; j < cols; j++ {
				val := vMat.At(i, j)
				sign := 1.0
				if val < 0 {
					sign = -1.0
				} else if val == 0 {
					sign = 0
				}
				targetNoNorm.Set(i, j, sign*math.Pow(math.Abs(val), opt.Rotation.Kappa))
			}
		}

		var ftTargetNoNorm mat.Dense
		ftTargetNoNorm.Mul(&ftNoNorm, targetNoNorm)
		var transNoNorm mat.Dense
		transNoNorm.Mul(&ftfNoNormInv, &ftTargetNoNorm)

		var transNoNormTtrans mat.Dense
		transNoNormTtrans.Mul(transNoNorm.T(), &transNoNorm)
		for j := 0; j < cols; j++ {
			diag := transNoNormTtrans.At(j, j)
			if diag <= 1e-12 {
				continue
			}
			scale := math.Sqrt(diag)
			if scale == 0 {
				continue
			}
			for i := 0; i < cols; i++ {
				transNoNorm.Set(i, j, transNoNorm.At(i, j)/scale)
			}
		}

		var rotatedNoNorm mat.Dense
		rotatedNoNorm.Mul(vMat, &transNoNorm)
		fmt.Printf("Manual trans (no norm):\n%v\n", mat.Formatted(&transNoNorm, mat.Prefix(" "), mat.Squeeze()))
		fmt.Printf("Manual rotated (no norm):\n%v\n", mat.Formatted(&rotatedNoNorm, mat.Prefix(" "), mat.Squeeze()))
		var transNoNormInv mat.Dense
		if err := transNoNormInv.Inverse(&transNoNorm); err == nil {
			var transNoNormInvT mat.Dense
			transNoNormInvT.CloneFrom(transNoNormInv.T())
			var phiNoNorm mat.Dense
			phiNoNorm.Mul(&transNoNormInv, &transNoNormInvT)
			phiNoNormNorm := normalizeDense(&phiNoNorm)
			fmt.Printf("Manual phi norm (no norm):\n%v\n", mat.Formatted(phiNoNormNorm, mat.Prefix(" "), mat.Squeeze()))
		}
	}

	rTarget := mat.NewDense(8, 2, []float64{
		0.07021078, 0.81111038,
		0.03065589, 0.77780405,
		-0.03441815, 0.69749410,
		0.01556561, 0.66120823,
		0.80073134, 0.03028619,
		0.76721134, 0.05655458,
		0.73119284, -0.01733010,
		0.65019591, -0.01219889,
	})

	lMat := mat.NewDense(8, 2, nil)
	for i := 0; i < 8; i++ {
		for j := 0; j < 2; j++ {
			lMat.Set(i, j, loadings[i][j])
		}
	}

	var lt mat.Dense
	lt.CloneFrom(lMat.T())
	var ltL mat.Dense
	ltL.Mul(&lt, lMat)
	var ltLInv mat.Dense
	if err := ltLInv.Inverse(&ltL); err != nil {
		t.Fatalf("invert ltL: %v", err)
	}

	var ltR mat.Dense
	ltR.Mul(&lt, rTarget)

	var trans mat.Dense
	trans.Mul(&ltLInv, &ltR)

	fmt.Printf("Transformation T =\n%v\n", mat.Formatted(&trans, mat.Prefix(" "), mat.Squeeze()))
	if transManualReady {
		var transManualInvForDiff mat.Dense
		if err := transManualInvForDiff.Inverse(&transManual); err == nil {
			var diff mat.Dense
			diff.Mul(&transManualInvForDiff, &trans)
			fmt.Printf("Manual trans^{-1} * T =\n%v\n", mat.Formatted(&diff, mat.Prefix(" "), mat.Squeeze()))
		}
	}

	var phi mat.Dense
	var transInv mat.Dense
	if err := transInv.Inverse(&trans); err == nil {
		var transInvT mat.Dense
		transInvT.CloneFrom(transInv.T())
		phi.Mul(&transInv, &transInvT)
		fmt.Printf("Derived phi =\n%v\n", mat.Formatted(&phi, mat.Prefix(" "), mat.Squeeze()))

		diagScales := []float64{math.Sqrt(phi.At(0, 0)), math.Sqrt(phi.At(1, 1))}
		fmt.Printf("Diag scales: %v\n", diagScales)

		var loadingsTransformed mat.Dense
		loadingsTransformed.Mul(lMat, &trans)
		fmt.Printf("L * T =\n%v\n", mat.Formatted(&loadingsTransformed, mat.Prefix(" "), mat.Squeeze()))

		for j := 0; j < 2; j++ {
			scale := diagScales[j]
			if scale == 0 {
				scale = 1
			}
			for i := 0; i < 8; i++ {
				loadingsTransformed.Set(i, j, loadingsTransformed.At(i, j)*scale)
			}
		}
		fmt.Printf("Scaled (L*T)*diag =\n%v\n", mat.Formatted(&loadingsTransformed, mat.Prefix(" "), mat.Squeeze()))
	}
}

func toMatrix(dt insyra.IDataTable) [][]float64 {
	var out [][]float64
	dt.AtomicDo(func(table *insyra.DataTable) {
		rows, cols := table.Size()
		out = make([][]float64, rows)
		for i := 0; i < rows; i++ {
			row := table.GetRow(i)
			out[i] = make([]float64, cols)
			for j := 0; j < cols; j++ {
				if v, ok := row.Get(j).(float64); ok {
					out[i][j] = v
				}
			}
		}
	})
	return out
}

func normalizeDense(m *mat.Dense) *mat.Dense {
	r, c := m.Dims()
	if r != c {
		return m
	}
	out := mat.NewDense(r, c, nil)
	d := make([]float64, r)
	for i := 0; i < r; i++ {
		v := m.At(i, i)
		if v <= 1e-12 {
			d[i] = 1
		} else {
			d[i] = math.Sqrt(v)
		}
	}
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			denom := d[i] * d[j]
			if denom <= 1e-12 {
				if i == j {
					out.Set(i, j, 1)
				} else {
					out.Set(i, j, 0)
				}
				continue
			}
			out.Set(i, j, m.At(i, j)/denom)
		}
	}
	return out
}
