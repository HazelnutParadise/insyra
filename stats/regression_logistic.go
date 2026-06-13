package stats

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"

	"github.com/HazelnutParadise/insyra"
)

type LogisticRegressionOptions struct {
	ConfidenceLevel  float64
	MaxIter          int
	Tolerance        float64
	PositiveClass    any
	SeparationPolicy SeparationPolicy
	Ridge            float64
}

type LogisticRegressionResult struct {
	Coefficients        []float64
	StandardErrors      []float64
	ZValues             []float64
	PValues             []float64
	ConfidenceIntervals [][2]float64
	OddsRatios          []float64
	OddsRatioCIs        [][2]float64
	LinearPredictors    []float64
	FittedProbabilities []float64
	Residuals           []float64
	PearsonResiduals    []float64
	DevianceResiduals   []float64
	Deviance            float64
	NullDeviance        float64
	LogLikelihood       float64
	NullLogLikelihood   float64
	AIC                 float64
	BIC                 float64
	McFaddenR2          float64
	CoxSnellR2          float64
	NagelkerkeR2        float64
	DFResidual          int
	Iterations          int
	Converged           bool
	SeparationDetected  bool
	Penalized           bool
	Ridge               float64
	PositiveClass       any
	ClassLabels         []any
	ConfidenceLevel     float64
	family              glmFamily
	link                glmLink
}

func LogisticRegression(dlY insyra.IDataList, dlXs ...insyra.IDataList) (*LogisticRegressionResult, error) {
	return LogisticRegressionWithOptions(LogisticRegressionOptions{}, dlY, dlXs...)
}

func LogisticRegressionWithOptions(opts LogisticRegressionOptions, dlY insyra.IDataList, dlXs ...insyra.IDataList) (*LogisticRegressionResult, error) {
	if len(dlXs) == 0 {
		return nil, errors.New("no independent variables provided")
	}
	if opts.SeparationPolicy == "" {
		opts.SeparationPolicy = SepWarn
	}
	if opts.SeparationPolicy != SepWarn && opts.SeparationPolicy != SepError && opts.SeparationPolicy != SepRidge {
		return nil, fmt.Errorf("unsupported separation policy %q", opts.SeparationPolicy)
	}
	ridge := 0.0
	if opts.SeparationPolicy == SepRidge {
		ridge = opts.Ridge
		if ridge <= 0 {
			ridge = 1e-4
		}
	}

	if dlY == nil {
		return nil, errors.New("y data list is nil")
	}
	xs, n, err := gatherPredictorInputs(dlXs)
	if err != nil {
		return nil, err
	}
	if n <= len(dlXs)+1 {
		return nil, errors.New("need at least p+2 observations for logistic regression")
	}
	var rawY []any
	dlY.AtomicDo(func(l *insyra.DataList) {
		rawY = l.Data()
	})
	if len(rawY) != n {
		return nil, errors.New("x and y must have the same length")
	}
	y, positiveClass, classLabels, err := encodeBinaryResponse(rawY, opts.PositiveClass)
	if err != nil {
		return nil, err
	}
	if len(y) != n {
		return nil, errors.New("encoded response length mismatch")
	}

	fam := binomialFamily{}
	link := logitLink{}
	X := buildDesignMatrix(xs, n)
	fit, err := fitIRLS(X, y, fam, link, irlsOptions{
		maxIter:   opts.MaxIter,
		tolerance: opts.Tolerance,
		ridge:     ridge,
	})
	if err != nil {
		return nil, err
	}
	if fit.sepFlag && opts.SeparationPolicy == SepError {
		return nil, errors.New("possible separation detected in logistic regression")
	}

	weights := priorWeightsOrOnes(nil, n)
	nullFit, err := fitNullIRLS(y, fam, link, nil, weights)
	if err != nil {
		return nil, err
	}
	cl := resolveConfidenceLevel(opts.ConfidenceLevel)
	se, z, p := computeGLMInference(fit.beta, fit.covUnscaled, 1)
	cis := buildGLMCoeffCIs(fit.beta, se, cl)
	logLik := glmLogLik(y, fit.mu, weights, fam, 1)
	nullLogLik := glmLogLik(y, nullFit.mu, weights, fam, 1)
	k := len(fit.beta)

	return &LogisticRegressionResult{
		Coefficients:        append([]float64(nil), fit.beta...),
		StandardErrors:      se,
		ZValues:             z,
		PValues:             p,
		ConfidenceIntervals: cis,
		OddsRatios:          expSlice(fit.beta),
		OddsRatioCIs:        expCIs(cis),
		LinearPredictors:    append([]float64(nil), fit.eta...),
		FittedProbabilities: append([]float64(nil), fit.mu...),
		Residuals:           responseResiduals(y, fit.mu),
		PearsonResiduals:    pearsonResiduals(y, fit.mu, weights, fam),
		DevianceResiduals:   devianceResiduals(y, fit.mu, weights, fam),
		Deviance:            fit.deviance,
		NullDeviance:        nullFit.deviance,
		LogLikelihood:       logLik,
		NullLogLikelihood:   nullLogLik,
		AIC:                 glmAIC(logLik, k),
		BIC:                 glmBIC(logLik, k, n),
		McFaddenR2:          mcFaddenR2(logLik, nullLogLik),
		CoxSnellR2:          coxSnellR2(logLik, nullLogLik, n),
		NagelkerkeR2:        nagelkerkeR2(logLik, nullLogLik, n),
		DFResidual:          n - k,
		Iterations:          fit.iterations,
		Converged:           fit.converged,
		SeparationDetected:  fit.sepFlag,
		Penalized:           ridge > 0,
		Ridge:               ridge,
		PositiveClass:       positiveClass,
		ClassLabels:         classLabels,
		ConfidenceLevel:     cl,
		family:              fam,
		link:                link,
	}, nil
}

func encodeBinaryResponse(raw []any, positiveClass any) ([]float64, any, []any, error) {
	if len(raw) == 0 {
		return nil, nil, nil, errors.New("response is empty")
	}
	unique := make(map[string]any)
	for _, v := range raw {
		unique[classKey(v)] = v
	}
	if len(unique) != 2 {
		return nil, nil, nil, fmt.Errorf("logistic regression requires exactly two response classes, got %d", len(unique))
	}

	classes := make([]any, 0, len(unique))
	for _, v := range unique {
		classes = append(classes, v)
	}
	sort.Slice(classes, func(i, j int) bool {
		return classSortKey(classes[i]) < classSortKey(classes[j])
	})

	if positiveClass == nil {
		positiveClass = defaultPositiveClass(classes)
	} else if !classEqual(positiveClass, classes[0]) && !classEqual(positiveClass, classes[1]) {
		return nil, nil, nil, fmt.Errorf("positive class %v is not present in response", positiveClass)
	}

	negativeClass := classes[0]
	if classEqual(negativeClass, positiveClass) {
		negativeClass = classes[1]
	}
	classLabels := []any{negativeClass, positiveClass}
	y := make([]float64, len(raw))
	for i, v := range raw {
		if classEqual(v, positiveClass) {
			y[i] = 1
		}
	}
	return y, positiveClass, classLabels, nil
}

func defaultPositiveClass(classes []any) any {
	if len(classes) != 2 {
		return nil
	}
	for _, v := range classes {
		if b, ok := v.(bool); ok && b {
			return v
		}
		if f, ok := numericClass(v); ok && f == 1 {
			return v
		}
	}
	return classes[1]
}

func classKey(v any) string {
	if f, ok := numericClass(v); ok {
		return fmt.Sprintf("num:%g", f)
	}
	return fmt.Sprintf("%T:%v", v, v)
}

func classSortKey(v any) string {
	if f, ok := numericClass(v); ok {
		return fmt.Sprintf("num:%020.12f", f)
	}
	return fmt.Sprintf("%T:%v", v, v)
}

func classEqual(a, b any) bool {
	if fa, okA := numericClass(a); okA {
		if fb, okB := numericClass(b); okB {
			return fa == fb
		}
	}
	return reflect.DeepEqual(a, b)
}

func numericClass(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case float32:
		return float64(x), true
	case float64:
		if math.IsNaN(x) {
			return 0, false
		}
		return x, true
	default:
		return 0, false
	}
}
