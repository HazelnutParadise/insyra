package stats

import (
	"math"

	"gonum.org/v1/gonum/stat/distuv"
)

func tTwoTailedPValue(t, df float64) float64 {
	if df <= 0 {
		return 1.0
	}
	dist := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}
	return 2 * (1 - dist.CDF(math.Abs(t)))
}

func tCDF(t, df float64) float64 {
	if df <= 0 {
		return math.NaN()
	}
	return distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}.CDF(t)
}

func tQuantile(p, df float64) float64 {
	if df <= 0 {
		return math.NaN()
	}
	return distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}.Quantile(p)
}

func fOneTailedPValue(f, df1, df2 float64) float64 {
	return 1 - distuv.F{D1: df1, D2: df2}.CDF(f)
}

func fTwoTailedPValue(f, df1, df2 float64) float64 {
	dist := distuv.F{D1: df1, D2: df2}
	return 2 * math.Min(dist.CDF(f), 1-dist.CDF(f))
}

func chiSquaredPValue(chi2, df float64) float64 {
	return 1 - distuv.ChiSquared{K: df}.CDF(chi2)
}

func zCDF(z float64) float64 {
	return norm.CDF(z)
}

func zPValue(z float64, alt AlternativeHypothesis) float64 {
	switch alt {
	case TwoSided:
		return 2 * (1 - zCDF(math.Abs(z)))
	case Greater:
		return 1 - zCDF(z)
	case Less:
		return zCDF(z)
	default:
		return math.NaN()
	}
}
