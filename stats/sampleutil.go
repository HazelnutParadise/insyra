package stats

import "math"

func sampleSE(s, n float64) float64 {
	return s / math.Sqrt(n)
}

func pooledVariance(var1, var2, n1, n2 float64) float64 {
	return ((n1-1)*var1 + (n2-1)*var2) / (n1 + n2 - 2)
}

func pooledSE(var1, var2, n1, n2 float64) (se, pVar float64) {
	pVar = pooledVariance(var1, var2, n1, n2)
	se = math.Sqrt(pVar * (1/n1 + 1/n2))
	return
}

func welchDF(var1, var2, n1, n2 float64) float64 {
	se1 := var1 / n1
	se2 := var2 / n2
	seSum := se1 + se2
	return (seSum * seSum) / (se1*se1/(n1-1) + se2*se2/(n2-1))
}

func twoSampleSE(var1, var2, n1, n2 float64) float64 {
	return math.Sqrt(var1/n1 + var2/n2)
}
