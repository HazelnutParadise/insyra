package stats

import (
	"errors"
	"fmt"

	"github.com/HazelnutParadise/insyra"
)

type PredictType string

const (
	PredictLinear   PredictType = "linear"
	PredictResponse PredictType = "response"
	PredictClass    PredictType = "class"
)

func (r *LogisticRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("logistic regression result is nil")
	}
	return predictFromCoefficients(r.Coefficients, r.link, typ, r.ClassLabels, newXs...)
}

func (r *PoissonRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("poisson regression result is nil")
	}
	if typ == PredictClass {
		return nil, errors.New("class prediction is only available for logistic regression")
	}
	return predictFromCoefficients(r.Coefficients, r.link, typ, nil, newXs...)
}

func (r *GLMResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("GLM result is nil")
	}
	if typ == PredictClass {
		return nil, errors.New("class prediction is only available for logistic regression")
	}
	return predictFromCoefficients(r.Coefficients, r.link, typ, nil, newXs...)
}

func predictFromCoefficients(beta []float64, link glmLink, typ PredictType, classLabels []any, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if typ == "" {
		typ = PredictResponse
	}
	if link == nil {
		return nil, errors.New("result link is not available")
	}
	if len(beta) == 0 {
		return nil, errors.New("no coefficients available")
	}
	if len(newXs) != len(beta)-1 {
		return nil, fmt.Errorf("expected %d predictors, got %d", len(beta)-1, len(newXs))
	}
	xs, n, err := gatherPredictorInputs(newXs)
	if err != nil {
		return nil, err
	}
	X := buildDesignMatrix(xs, n)
	out := make([]any, n)
	for i := range n {
		eta := 0.0
		for j := range beta {
			eta += X.At(i, j) * beta[j]
		}
		switch typ {
		case PredictLinear:
			out[i] = eta
		case PredictResponse:
			out[i] = link.mu(eta)
		case PredictClass:
			if len(classLabels) != 2 {
				return nil, errors.New("class labels are not available")
			}
			if link.mu(eta) >= 0.5 {
				out[i] = classLabels[1]
			} else {
				out[i] = classLabels[0]
			}
		default:
			return nil, fmt.Errorf("unsupported predict type %q", typ)
		}
	}
	return insyra.NewDataList(out), nil
}
