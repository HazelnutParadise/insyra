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
	return predictFromCoefficients(r.Coefficients, r.link, typ, r.ClassLabels, nil, newXs...)
}

func (r *PoissonRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("poisson regression result is nil")
	}
	if typ == PredictClass {
		return nil, errors.New("class prediction is only available for logistic regression")
	}
	if r.hasOffset {
		return nil, errors.New("model was fit with an offset; use PredictWithOffset to supply the new-data offset")
	}
	return predictFromCoefficients(r.Coefficients, r.link, typ, nil, nil, newXs...)
}

// PredictWithOffset predicts on new data while applying a per-row offset, as
// required when the model was fit with an offset (e.g. log-exposure rate models).
func (r *PoissonRegressionResult) PredictWithOffset(typ PredictType, offset insyra.IDataList, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("poisson regression result is nil")
	}
	if typ == PredictClass {
		return nil, errors.New("class prediction is only available for logistic regression")
	}
	if offset == nil {
		return nil, errors.New("offset data list is nil")
	}
	return predictFromCoefficients(r.Coefficients, r.link, typ, nil, offset, newXs...)
}

func (r *GLMResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("GLM result is nil")
	}
	if typ == PredictClass {
		return nil, errors.New("class prediction is only available for logistic regression")
	}
	if r.hasOffset {
		return nil, errors.New("model was fit with an offset; use PredictWithOffset to supply the new-data offset")
	}
	return predictFromCoefficients(r.Coefficients, r.link, typ, nil, nil, newXs...)
}

// PredictWithOffset predicts on new data while applying a per-row offset, as
// required when the model was fit with an offset.
func (r *GLMResult) PredictWithOffset(typ PredictType, offset insyra.IDataList, newXs ...insyra.IDataList) (*insyra.DataList, error) {
	if r == nil {
		return nil, errors.New("GLM result is nil")
	}
	if typ == PredictClass {
		return nil, errors.New("class prediction is only available for logistic regression")
	}
	if offset == nil {
		return nil, errors.New("offset data list is nil")
	}
	return predictFromCoefficients(r.Coefficients, r.link, typ, nil, offset, newXs...)
}

func predictFromCoefficients(beta []float64, link glmLink, typ PredictType, classLabels []any, offsetDL insyra.IDataList, newXs ...insyra.IDataList) (*insyra.DataList, error) {
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
	var offset []float64
	if offsetDL != nil {
		offsetDL.AtomicDo(func(l *insyra.DataList) {
			offset = l.ToF64Slice()
		})
		if len(offset) != n {
			return nil, fmt.Errorf("offset length %d does not match %d predictor rows", len(offset), n)
		}
	}
	X := buildDesignMatrix(xs, n)
	out := make([]any, n)
	for i := range n {
		eta := 0.0
		for j := range beta {
			eta += X.At(i, j) * beta[j]
		}
		if offset != nil {
			eta += offset[i]
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
