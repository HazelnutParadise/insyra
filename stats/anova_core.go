package stats

import "errors"

type oneWayANOVAStats struct {
	SSB float64
	SSW float64
	DFB int
	DFW int
	F   float64
	P   float64
	Eta float64
}

func oneWayANOVAFromSlices(values []float64, labels []int, k int) (*oneWayANOVAStats, error) {
	if k < 2 || len(values) == 0 || len(values) != len(labels) {
		return nil, errors.New("invalid group count or input lengths")
	}

	n := len(values)
	groupSums := make([]float64, k)
	groupCounts := make([]int, k)
	totalSum := 0.0

	for i, v := range values {
		group := labels[i]
		if group < 0 || group >= k {
			return nil, errors.New("group label out of range")
		}
		groupSums[group] += v
		groupCounts[group]++
		totalSum += v
	}

	for _, count := range groupCounts {
		if count == 0 {
			return nil, errors.New("at least one group is empty")
		}
	}

	totalMean := totalSum / float64(n)

	ssb := 0.0
	for i := 0; i < k; i++ {
		mean := groupSums[i] / float64(groupCounts[i])
		ssb += float64(groupCounts[i]) * (mean - totalMean) * (mean - totalMean)
	}

	ssw := 0.0
	for i, v := range values {
		group := labels[i]
		mean := groupSums[group] / float64(groupCounts[group])
		ssw += (v - mean) * (v - mean)
	}

	dfb := k - 1
	dfw := n - k
	if dfb <= 0 || dfw <= 0 {
		return nil, errors.New("invalid degrees of freedom")
	}

	f := fRatio(ssb, dfb, ssw, dfw)
	return &oneWayANOVAStats{
		SSB: ssb,
		SSW: ssw,
		DFB: dfb,
		DFW: dfw,
		F:   f,
		P:   fOneTailedPValue(f, float64(dfb), float64(dfw)),
		Eta: etaSquared(ssb, ssw),
	}, nil
}
