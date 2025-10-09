package stats

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/mat"
)

// dataTableToDense converts an insyra.IDataTable (returned by FactorAnalysis)
// back into a *mat.Dense. Returns nil if input is nil or conversion fails.
func dataTableToDense(dt insyra.IDataTable) *mat.Dense {
	if dt == nil {
		return nil
	}
	r, c := dt.Size()
	if r == 0 || c == 0 {
		return nil
	}
	m := mat.NewDense(r, c, nil)
	for j := 0; j < c; j++ {
		col := dt.GetColByNumber(j)
		if col == nil {
			return nil
		}
		vals := col.ToF64Slice()
		// If lengths mismatch, fail
		if len(vals) != r {
			return nil
		}
		for i := 0; i < r; i++ {
			m.Set(i, j, vals[i])
		}
	}
	return m
}

// RunPAFObliminForDiagnostics runs FactorAnalysis with the provided options
// (typically PAF extraction + Oblimin rotation) and returns a map containing
// the rotated loadings ("loadings"), rotation matrix ("rotmat"), and
// Phi ("Phi") as *mat.Dense so callers can programmatically compare
// results with external references (e.g., SPSS output).
func RunPAFObliminForDiagnostics(dt insyra.IDataTable, opts FactorAnalysisOptions) map[string]any {
	if dt == nil {
		return nil
	}
	fm := FactorAnalysis(dt, opts)
	if fm == nil {
		return nil
	}

	res := make(map[string]any)

	// Loadings (FactorLoadings)
	if fm.Loadings != nil {
		if ld := dataTableToDense(fm.Loadings); ld != nil {
			res["loadings"] = ld
		}
	}

	// RotationMatrix (this is T as stored in FactorAnalysis)
	if fm.RotationMatrix != nil {
		if rm := dataTableToDense(fm.RotationMatrix); rm != nil {
			res["rotmat"] = rm
		}
	}

	// Phi
	if fm.Phi != nil {
		if ph := dataTableToDense(fm.Phi); ph != nil {
			res["Phi"] = ph
		}
	}

	// Include convergence flags for reference
	res["Converged"] = fm.Converged
	res["RotationConverged"] = fm.RotationConverged
	res["Iterations"] = fm.Iterations

	// Optionally include brief message
	res["msg"] = fmt.Sprintf("diagnostics: converged=%v rotConv=%v it=%d", fm.Converged, fm.RotationConverged, fm.Iterations)

	return res
}
