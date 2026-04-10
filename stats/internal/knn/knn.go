package knn

import (
	"errors"
	"math"
	"sort"
)

type Weighting string

const (
	UniformWeighting  Weighting = "uniform"
	DistanceWeighting Weighting = "distance"
)

type Options struct {
	Weighting Weighting
}

type ClassificationResult struct {
	Predictions   []string
	Classes       []string
	Probabilities [][]float64
}

type RegressionResult struct {
	Predictions []float64
}

type neighbor struct {
	index    int
	distance float64
}

func Classify(train, test [][]float64, labels []string, k int, opts Options) (*ClassificationResult, error) {
	weighting, err := normalizeWeighting(opts.Weighting)
	if err != nil {
		return nil, err
	}
	if err := validateInputs(train, test, k); err != nil {
		return nil, err
	}
	if len(labels) != len(train) {
		return nil, errors.New("labels length must match training row count")
	}

	classes, classIndex := orderedClasses(labels)
	predictions := make([]string, len(test))
	probabilities := make([][]float64, len(test))
	for i, row := range test {
		neighbors := kNearest(train, row, k)
		probabilities[i] = classifyProbabilities(neighbors, labels, classIndex, len(classes), weighting)
		best := 0
		bestMeanDistance := classMeanDistance(neighbors, labels, classIndex, 0)
		for c := 1; c < len(classes); c++ {
			if probabilities[i][c] > probabilities[i][best] && !almostEqual(probabilities[i][c], probabilities[i][best]) {
				best = c
				bestMeanDistance = classMeanDistance(neighbors, labels, classIndex, c)
				continue
			}
			if almostEqual(probabilities[i][c], probabilities[i][best]) {
				currentMeanDistance := classMeanDistance(neighbors, labels, classIndex, c)
				if currentMeanDistance < bestMeanDistance && !almostEqual(currentMeanDistance, bestMeanDistance) {
					best = c
					bestMeanDistance = currentMeanDistance
				}
			}
		}
		predictions[i] = classes[best]
	}
	return &ClassificationResult{
		Predictions:   predictions,
		Classes:       classes,
		Probabilities: probabilities,
	}, nil
}

func Regress(train, test [][]float64, targets []float64, k int, opts Options) (*RegressionResult, error) {
	weighting, err := normalizeWeighting(opts.Weighting)
	if err != nil {
		return nil, err
	}
	if err := validateInputs(train, test, k); err != nil {
		return nil, err
	}
	if len(targets) != len(train) {
		return nil, errors.New("targets length must match training row count")
	}

	predictions := make([]float64, len(test))
	for i, row := range test {
		neighbors := kNearest(train, row, k)
		predictions[i] = regressPrediction(neighbors, targets, weighting)
	}
	return &RegressionResult{Predictions: predictions}, nil
}

func normalizeWeighting(weighting Weighting) (Weighting, error) {
	if weighting == "" {
		return UniformWeighting, nil
	}
	switch weighting {
	case UniformWeighting, DistanceWeighting:
		return weighting, nil
	default:
		return "", errors.New("unsupported KNN weighting")
	}
}

func validateInputs(train, test [][]float64, k int) error {
	if len(train) == 0 || len(train[0]) == 0 {
		return errors.New("training data must have at least 1 row and 1 column")
	}
	if len(test) == 0 || len(test[0]) == 0 {
		return errors.New("test data must have at least 1 row and 1 column")
	}
	if k <= 0 {
		return errors.New("k must be greater than 0")
	}
	if k > len(train) {
		return errors.New("k must not exceed training row count")
	}
	p := len(train[0])
	for i, row := range train {
		if len(row) != p {
			return errors.New("training rows must all have the same dimension")
		}
		if hasInvalidFloat(row) {
			return errors.New("training data contains invalid numeric values")
		}
		if len(row) != p {
			return errors.New("training rows must all have the same dimension")
		}
		_ = i
	}
	for _, row := range test {
		if len(row) != p {
			return errors.New("training and test data must have the same column count")
		}
		if hasInvalidFloat(row) {
			return errors.New("test data contains invalid numeric values")
		}
	}
	return nil
}

func hasInvalidFloat(row []float64) bool {
	for _, v := range row {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return true
		}
	}
	return false
}

func orderedClasses(labels []string) ([]string, map[string]int) {
	classes := make([]string, 0)
	classIndex := make(map[string]int, len(labels))
	for _, label := range labels {
		if _, ok := classIndex[label]; ok {
			continue
		}
		classIndex[label] = len(classes)
		classes = append(classes, label)
	}
	return classes, classIndex
}

func kNearest(train [][]float64, query []float64, k int) []neighbor {
	neighbors := make([]neighbor, len(train))
	for i, row := range train {
		neighbors[i] = neighbor{
			index:    i,
			distance: euclidean(row, query),
		}
	}
	sort.Slice(neighbors, func(i, j int) bool {
		if almostEqual(neighbors[i].distance, neighbors[j].distance) {
			return neighbors[i].index < neighbors[j].index
		}
		return neighbors[i].distance < neighbors[j].distance
	})
	return neighbors[:k]
}

func classifyProbabilities(neighbors []neighbor, labels []string, classIndex map[string]int, nClasses int, weighting Weighting) []float64 {
	weights := make([]float64, nClasses)
	hasZeroDistance := false
	for _, nb := range neighbors {
		if almostEqual(nb.distance, 0) {
			hasZeroDistance = true
			break
		}
	}
	for _, nb := range neighbors {
		if hasZeroDistance && !almostEqual(nb.distance, 0) {
			continue
		}
		idx := classIndex[labels[nb.index]]
		weights[idx] += neighborWeight(nb.distance, weighting)
	}
	total := 0.0
	for _, weight := range weights {
		total += weight
	}
	if total == 0 {
		return weights
	}
	for i := range weights {
		weights[i] /= total
	}
	return weights
}

func classMeanDistance(neighbors []neighbor, labels []string, classIndex map[string]int, class int) float64 {
	sum := 0.0
	count := 0.0
	for _, nb := range neighbors {
		if classIndex[labels[nb.index]] != class {
			continue
		}
		sum += nb.distance
		count++
	}
	if count == 0 {
		return math.Inf(1)
	}
	return sum / count
}

func regressPrediction(neighbors []neighbor, targets []float64, weighting Weighting) float64 {
	hasZeroDistance := false
	for _, nb := range neighbors {
		if almostEqual(nb.distance, 0) {
			hasZeroDistance = true
			break
		}
	}
	sumWeight := 0.0
	sumTarget := 0.0
	for _, nb := range neighbors {
		if hasZeroDistance && !almostEqual(nb.distance, 0) {
			continue
		}
		weight := neighborWeight(nb.distance, weighting)
		sumWeight += weight
		sumTarget += weight * targets[nb.index]
	}
	if sumWeight == 0 {
		return math.NaN()
	}
	return sumTarget / sumWeight
}

func neighborWeight(distance float64, weighting Weighting) float64 {
	switch weighting {
	case DistanceWeighting:
		if almostEqual(distance, 0) {
			return 1
		}
		return 1 / distance
	default:
		return 1
	}
}

func euclidean(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum)
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-12
}
