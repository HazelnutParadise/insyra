package insyra

import "math"

// EWMOptions configures an exponentially weighted computation. Exactly one of
// Alpha, Span, or HalfLife must be set. Adjust and Bias follow pandas Series.ewm
// and its var/std reducers respectively. MinObs defaults to one.
type EWMOptions struct {
	Alpha    float64
	Span     float64
	HalfLife float64
	Adjust   bool
	Bias     bool
	MinObs   int
}

// EWMDataList is the intermediate produced by DataList.EWM. It contains a
// snapshot of the source, so later source mutations do not affect reducers.
type EWMDataList struct {
	srcData []any
	srcName string
	opts    EWMOptions
	alpha   float64
	parent  *DataList
	err     string
}

// EWM builds an exponentially weighted view over dl. Invalid decay options
// emit a warning and make every reducer return an empty DataList.
func (dl *DataList) EWM(opts EWMOptions) *EWMDataList {
	e := &EWMDataList{opts: opts, parent: dl}
	if opts.MinObs <= 0 {
		e.opts.MinObs = 1
	}

	decayCount := 0
	if opts.Alpha != 0 {
		decayCount++
	}
	if opts.Span != 0 {
		decayCount++
	}
	if opts.HalfLife != 0 {
		decayCount++
	}
	if decayCount != 1 {
		e.err = "EWM: exactly one of Alpha, Span, or HalfLife must be specified"
		dl.warn("EWM", "%s", e.err)
		return e
	}

	switch {
	case opts.Alpha != 0:
		if math.IsNaN(opts.Alpha) || math.IsInf(opts.Alpha, 0) || opts.Alpha <= 0 || opts.Alpha > 1 {
			e.err = "EWM: Alpha must be in (0, 1]"
		}
		e.alpha = opts.Alpha
	case opts.Span != 0:
		if math.IsNaN(opts.Span) || math.IsInf(opts.Span, 0) || opts.Span < 1 {
			e.err = "EWM: Span must be >= 1"
		} else {
			e.alpha = 2 / (opts.Span + 1)
		}
	case opts.HalfLife != 0:
		if math.IsNaN(opts.HalfLife) || math.IsInf(opts.HalfLife, 0) || opts.HalfLife <= 0 {
			e.err = "EWM: HalfLife must be > 0"
		} else {
			e.alpha = 1 - math.Exp(math.Log(0.5)/opts.HalfLife)
		}
	}
	if e.err != "" {
		dl.warn("EWM", "%s", e.err)
		return e
	}

	dl.AtomicDo(func(dl *DataList) {
		e.srcData = make([]any, len(dl.data))
		copy(e.srcData, dl.data)
		e.srcName = dl.name
	})
	return e
}

// Mean returns the exponentially weighted mean.
func (e *EWMDataList) Mean() *DataList {
	means, _ := e.compute()
	return e.result(means)
}

// Var returns the exponentially weighted variance. Bias=false applies pandas'
// effective-weight correction, while Bias=true returns the weighted population
// variance.
func (e *EWMDataList) Var() *DataList {
	_, vars := e.compute()
	return e.result(vars)
}

// Std returns the square root of Var.
func (e *EWMDataList) Std() *DataList {
	_, vars := e.compute()
	for i, value := range vars {
		if value == nil {
			continue
		}
		vars[i] = math.Sqrt(value.(float64))
	}
	return e.result(vars)
}

func (e *EWMDataList) result(values []any) *DataList {
	if e.err != "" {
		out := NewDataList()
		out.name = e.srcName
		return out
	}
	out := NewDataList(values...)
	out.name = e.srcName
	return out
}

// compute follows pandas' one-pass EWM update. Missing cells leave the last
// emitted value in place but still decay the accumulated weights, matching
// pandas' default ignore_na=false behavior.
func (e *EWMDataList) compute() ([]any, []any) {
	if e.err != "" {
		return []any{}, []any{}
	}
	means := make([]any, len(e.srcData))
	vars := make([]any, len(e.srcData))

	alpha := e.alpha
	oldWeightFactor := 1 - alpha
	newWeight := alpha
	if e.opts.Adjust {
		newWeight = 1
	}
	mean := 0.0
	variance := 0.0
	varianceOutput := any(nil)
	hasMean := false
	nobs := 0
	weightSum := 0.0
	weightSquareSum := 0.0

	for i, raw := range e.srcData {
		x, ok := ToFloat64Safe(raw)
		valid := ok && !math.IsNaN(x)

		// The public API follows pandas' default ignore_na=false semantics:
		// every gap advances the decay even though it does not add an observation.
		if hasMean {
			weightSum *= oldWeightFactor
			weightSquareSum *= oldWeightFactor * oldWeightFactor
		}

		if valid {
			nobs++
			if !hasMean {
				mean = x
				variance = 0
				weightSum = 1
				weightSquareSum = 1
				hasMean = true
			} else {
				oldMean := mean
				oldWeight := weightSum
				totalWeight := oldWeight + newWeight
				mean = (oldWeight*oldMean + newWeight*x) / totalWeight
				oldDelta := oldMean - mean
				newDelta := x - mean
				variance = (oldWeight*(variance+oldDelta*oldDelta) + newWeight*newDelta*newDelta) / totalWeight
				weightSquareSum += newWeight * newWeight
				weightSum = totalWeight
				if !e.opts.Adjust {
					// The recursive form renormalizes after every observation.
					// Keep the squared-weight state normalized as well, while
					// preserving the effect of gaps before this observation.
					weightSquareSum /= totalWeight * totalWeight
					weightSum = 1
				}
			}
		}

		if !hasMean || nobs < e.opts.MinObs {
			means[i] = nil
			vars[i] = nil
			continue
		}
		means[i] = mean
		if e.opts.Bias {
			varianceOutput = variance
		} else {
			denominator := weightSum*weightSum - weightSquareSum
			if nobs < 2 || denominator <= 0 {
				varianceOutput = nil
			} else {
				varianceOutput = variance * weightSum * weightSum / denominator
			}
		}
		vars[i] = varianceOutput
	}
	return means, vars
}
