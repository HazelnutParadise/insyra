package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

type PoissonRegressionOptions struct {
	ConfidenceLevel float64
	MaxIter         int
	Tolerance       float64
	Offset          insyra.IDataList
	DispersionCheck bool
}

type PoissonRegressionResult struct {
	Coefficients           []float64
	StandardErrors         []float64
	ZValues                []float64
	PValues                []float64
	ConfidenceIntervals    [][2]float64
	IncidenceRateRatios    []float64
	IRR                    []float64
	IRRConfidenceIntervals [][2]float64
	IRRConfidenceInterval  [][2]float64
	LinearPredictors       []float64
	FittedRates            []float64
	FittedValues           []float64
	Residuals              []float64
	PearsonResiduals       []float64
	DevianceResiduals      []float64
	Deviance               float64
	NullDeviance           float64
	LogLikelihood          float64
	NullLogLikelihood      float64
	AIC                    float64
	BIC                    float64
	PearsonChi2            float64
	DispersionStatistic    float64
	OverDispersed          bool
	DFResidual             int
	Iterations             int
	Converged              bool
	ConfidenceLevel        float64
	family                 glmFamily
	link                   glmLink
}

func PoissonRegression(dlY insyra.IDataList, dlXs ...insyra.IDataList) (*PoissonRegressionResult, error) {
	return PoissonRegressionWithOptions(PoissonRegressionOptions{}, dlY, dlXs...)
}

func PoissonRegressionWithOptions(opts PoissonRegressionOptions, dlY insyra.IDataList, dlXs ...insyra.IDataList) (*PoissonRegressionResult, error) {
	if len(dlXs) == 0 {
		return nil, errors.New("no independent variables provided")
	}
	extras := []insyra.IDataList(nil)
	if opts.Offset != nil {
		extras = append(extras, opts.Offset)
	}
	y, xs, extraVals, n, err := gatherRegressionInputs(dlY, dlXs, extras...)
	if err != nil {
		return nil, err
	}
	if n <= len(dlXs)+1 {
		return nil, errors.New("need at least p+2 observations for poisson regression")
	}
	for i, v := range y {
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("poisson response must be non-negative finite values (index %d)", i)
		}
	}
	var offset []float64
	if len(extraVals) > 0 {
		offset = extraVals[0]
	}

	fam := poissonFamily{}
	link := logLink{}
	X := buildDesignMatrix(xs, n)
	fit, err := fitIRLS(X, y, fam, link, irlsOptions{
		maxIter:   opts.MaxIter,
		tolerance: opts.Tolerance,
		offset:    offset,
	})
	if err != nil {
		return nil, err
	}

	weights := priorWeightsOrOnes(nil, n)
	nullFit, err := fitNullIRLS(y, fam, link, offset, weights)
	if err != nil {
		return nil, err
	}
	cl := resolveConfidenceLevel(opts.ConfidenceLevel)
	se, z, p := computeGLMInference(fit.beta, fit.covUnscaled, 1)
	cis := buildGLMCoeffCIs(fit.beta, se, cl)
	logLik := glmLogLik(y, fit.mu, weights, fam, 1)
	nullLogLik := glmLogLik(y, nullFit.mu, weights, fam, 1)
	k := len(fit.beta)
	dfResidual := n - k
	pearson := pearsonChiSq(y, fit.mu, weights, fam)
	dispersion := pearsonDispersion(pearson, dfResidual)
	overDispersed := opts.DispersionCheck && dispersion > 1.5
	if overDispersed {
		insyra.LogWarning("stats", "PoissonRegression", "possible over-dispersion detected: Pearson chi-square / df = %.6g", dispersion)
	}
	irr := expSlice(fit.beta)
	irrCI := expCIs(cis)

	return &PoissonRegressionResult{
		Coefficients:           append([]float64(nil), fit.beta...),
		StandardErrors:         se,
		ZValues:                z,
		PValues:                p,
		ConfidenceIntervals:    cis,
		IncidenceRateRatios:    irr,
		IRR:                    irr,
		IRRConfidenceIntervals: irrCI,
		IRRConfidenceInterval:  irrCI,
		LinearPredictors:       append([]float64(nil), fit.eta...),
		FittedRates:            append([]float64(nil), fit.mu...),
		FittedValues:           append([]float64(nil), fit.mu...),
		Residuals:              responseResiduals(y, fit.mu),
		PearsonResiduals:       pearsonResiduals(y, fit.mu, weights, fam),
		DevianceResiduals:      devianceResiduals(y, fit.mu, weights, fam),
		Deviance:               fit.deviance,
		NullDeviance:           nullFit.deviance,
		LogLikelihood:          logLik,
		NullLogLikelihood:      nullLogLik,
		AIC:                    glmAIC(logLik, k),
		BIC:                    glmBIC(logLik, k, n),
		PearsonChi2:            pearson,
		DispersionStatistic:    dispersion,
		OverDispersed:          overDispersed,
		DFResidual:             dfResidual,
		Iterations:             fit.iterations,
		Converged:              fit.converged,
		ConfidenceLevel:        cl,
		family:                 fam,
		link:                   link,
	}, nil
}
