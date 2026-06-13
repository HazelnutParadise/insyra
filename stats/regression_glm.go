package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

type GLMOptions struct {
	Family          GLMFamily
	Link            GLMLink
	ConfidenceLevel float64
	MaxIter         int
	Tolerance       float64
	Offset          insyra.IDataList
	Weights         insyra.IDataList
}

type GLMResult struct {
	Family              GLMFamily
	Link                GLMLink
	Coefficients        []float64
	StandardErrors      []float64
	ZValues             []float64
	PValues             []float64
	ConfidenceIntervals [][2]float64
	LinearPredictors    []float64
	FittedValues        []float64
	Residuals           []float64
	PearsonResiduals    []float64
	DevianceResiduals   []float64
	Deviance            float64
	NullDeviance        float64
	LogLikelihood       float64
	NullLogLikelihood   float64
	AIC                 float64
	BIC                 float64
	PearsonChi2         float64
	Dispersion          float64
	DFResidual          int
	Iterations          int
	Converged           bool
	ConfidenceLevel     float64
	family              glmFamily
	link                glmLink
}

func GLM(opts GLMOptions, dlY insyra.IDataList, dlXs ...insyra.IDataList) (*GLMResult, error) {
	if len(dlXs) == 0 {
		return nil, errors.New("no independent variables provided")
	}
	fam, err := resolveGLMFamily(opts.Family)
	if err != nil {
		return nil, err
	}
	link, linkName, err := resolveGLMLink(fam, opts.Link)
	if err != nil {
		return nil, err
	}
	extras := []insyra.IDataList(nil)
	offsetIdx, weightsIdx := -1, -1
	if opts.Offset != nil {
		offsetIdx = len(extras)
		extras = append(extras, opts.Offset)
	}
	if opts.Weights != nil {
		weightsIdx = len(extras)
		extras = append(extras, opts.Weights)
	}
	y, xs, extraVals, n, err := gatherRegressionInputs(dlY, dlXs, extras...)
	if err != nil {
		return nil, err
	}
	if n <= len(dlXs)+1 {
		return nil, errors.New("need at least p+2 observations for GLM")
	}
	if err := validateGLMResponse(y, fam); err != nil {
		return nil, err
	}
	var offset []float64
	if offsetIdx >= 0 {
		offset = extraVals[offsetIdx]
	}
	var priorWeights []float64
	if weightsIdx >= 0 {
		priorWeights = extraVals[weightsIdx]
	}

	X := buildDesignMatrix(xs, n)
	fitOpts := irlsOptions{
		maxIter:   opts.MaxIter,
		tolerance: opts.Tolerance,
		offset:    offset,
		weights:   priorWeights,
	}
	fit, err := fitIRLS(X, y, fam, link, fitOpts)
	if err != nil {
		return nil, err
	}
	weights := priorWeightsOrOnes(priorWeights, n)
	nullFit, err := fitNullIRLSWithOptions(y, fam, link, offset, weights, fitOpts)
	if err != nil {
		return nil, err
	}
	dfResidual := n - len(fit.beta)
	pearson := pearsonChiSq(y, fit.mu, weights, fam)
	dispersion, fixed := fam.dispersionFixed()
	if !fixed {
		dispersion = pearsonDispersion(pearson, dfResidual)
	}
	se, z, p := computeGLMInference(fit.beta, fit.covUnscaled, dispersion)
	if fam.name() == string(Gaussian) {
		for i := range z {
			if !math.IsNaN(z[i]) && dfResidual > 0 {
				p[i] = tTwoTailedPValue(z[i], float64(dfResidual))
			}
		}
	}
	cl := resolveConfidenceLevel(opts.ConfidenceLevel)
	cis := buildGLMCoeffCIs(fit.beta, se, cl)
	logLikDispersion := dispersion
	if fam.name() == string(Gaussian) && n > 0 {
		logLikDispersion = fit.deviance / float64(n)
	}
	logLik := glmLogLik(y, fit.mu, weights, fam, logLikDispersion)
	nullLogLik := glmLogLik(y, nullFit.mu, weights, fam, logLikDispersion)
	k := len(fit.beta)
	aicK := k
	if fam.name() == string(Gaussian) {
		aicK++
	}

	return &GLMResult{
		Family:              opts.Family,
		Link:                linkName,
		Coefficients:        append([]float64(nil), fit.beta...),
		StandardErrors:      se,
		ZValues:             z,
		PValues:             p,
		ConfidenceIntervals: cis,
		LinearPredictors:    append([]float64(nil), fit.eta...),
		FittedValues:        append([]float64(nil), fit.mu...),
		Residuals:           responseResiduals(y, fit.mu),
		PearsonResiduals:    pearsonResiduals(y, fit.mu, weights, fam),
		DevianceResiduals:   devianceResiduals(y, fit.mu, weights, fam),
		Deviance:            fit.deviance,
		NullDeviance:        nullFit.deviance,
		LogLikelihood:       logLik,
		NullLogLikelihood:   nullLogLik,
		AIC:                 glmAIC(logLik, aicK),
		BIC:                 glmBIC(logLik, aicK, n),
		PearsonChi2:         pearson,
		Dispersion:          dispersion,
		DFResidual:          dfResidual,
		Iterations:          fit.iterations,
		Converged:           fit.converged,
		ConfidenceLevel:     cl,
		family:              fam,
		link:                link,
	}, nil
}

func resolveGLMFamily(family GLMFamily) (glmFamily, error) {
	switch family {
	case Binomial:
		return binomialFamily{}, nil
	case Poisson:
		return poissonFamily{}, nil
	case Gaussian:
		return gaussianFamily{}, nil
	default:
		return nil, fmt.Errorf("unsupported GLM family %q", family)
	}
}

func resolveGLMLink(fam glmFamily, linkName GLMLink) (glmLink, GLMLink, error) {
	if linkName == "" {
		link := fam.canonicalLink()
		return link, GLMLink(link.name()), nil
	}
	switch linkName {
	case Logit:
		if fam.name() != string(Binomial) {
			return nil, "", fmt.Errorf("link %q is not supported for family %q", linkName, fam.name())
		}
		return logitLink{}, linkName, nil
	case Log:
		if fam.name() != string(Poisson) {
			return nil, "", fmt.Errorf("link %q is not supported for family %q", linkName, fam.name())
		}
		return logLink{}, linkName, nil
	case Identity:
		if fam.name() != string(Gaussian) {
			return nil, "", fmt.Errorf("link %q is not supported for family %q", linkName, fam.name())
		}
		return identityLink{}, linkName, nil
	case Probit, Cloglog:
		return nil, "", fmt.Errorf("link %q is reserved but not implemented", linkName)
	default:
		return nil, "", fmt.Errorf("unsupported GLM link %q", linkName)
	}
}

func validateGLMResponse(y []float64, fam glmFamily) error {
	for i, v := range y {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("response must be finite (index %d)", i)
		}
		switch fam.name() {
		case string(Binomial):
			if v < 0 || v > 1 {
				return fmt.Errorf("binomial response must be in [0,1] (index %d)", i)
			}
		case string(Poisson):
			if v < 0 {
				return fmt.Errorf("poisson response must be non-negative (index %d)", i)
			}
		}
	}
	return nil
}
