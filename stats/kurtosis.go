// kurtosis.go - Calculate the kurtosis of the DataList.

package stats

import (
	"math/big"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/parallel"
)

// Kurtosis calculates the kurtosis(sample) of the DataList.
// Returns the kurtosis.
// Returns nil if the DataList is empty or the kurtosis cannot be calculated.
func Kurtosis(data interface{}, method ...int) interface{} {
	d, _ := insyra.ProcessData(data)
	d64 := insyra.SliceToF64(d)
	dl := insyra.NewDataList(d64)
	usemethod := 1
	if len(method) > 1 {
		insyra.LogWarning("stats.Kurtosis(): More than one method specified, returning nil.")
		return nil
	}
	if len(method) == 1 {
		usemethod = method[0]
	}

	var result *big.Rat
	var ok bool
	switch usemethod {
	case 1:
		result, ok = calculateKurtType1(dl)
	case 2:
		result, ok = calculateKurtType2(dl)
	case 3:
		result, ok = calculateKurtType3(dl)
	default:
		insyra.LogWarning("stats.Kurtosis(): Invalid method, returning nil.")
		return nil
	}

	if !ok {
		insyra.LogWarning("stats.Kurtosis(): Kurtosis is nil, returning nil.")
		return nil
	}

	f64Result, _ := result.Float64()
	return f64Result
}

// ======================== calculation functions ========================
func calculateKurtType1(dl insyra.IDataList) (*big.Rat, bool) {
	// 初始化 m2 和 m4 的計算
	var m2, m4 *big.Rat
	parallel.GroupUp(func() {
		m2, _ = CalculateMoment(dl, 2, true)
	}, func() {
		m4, _ = CalculateMoment(dl, 4, true)
	}).Run().AwaitResult()

	// 計算峰態
	m2Pow2 := new(big.Rat).Mul(m2, m2) // m2^2
	if m2Pow2.Sign() == 0 {
		return nil, false // 如果二階矩為0，返回錯誤，避免除以0
	}

	// g2 = m4 / m2^2 - 3
	result := new(big.Rat).Sub(new(big.Rat).Quo(m4, m2Pow2), new(big.Rat).SetInt64(3))

	return result, true
}

func calculateKurtType2(dl insyra.IDataList) (*big.Rat, bool) {
	n := new(big.Rat).SetFloat64(float64(dl.Len()))
	g2, ok := calculateKurtType1(dl)
	if !ok {
		return nil, false
	}

	nPlus1 := new(big.Rat).Add(n, new(big.Rat).SetInt64(1))
	nMinus1 := new(big.Rat).Sub(n, new(big.Rat).SetInt64(1))
	nMinus2 := new(big.Rat).Sub(n, new(big.Rat).SetInt64(2))
	nMinus3 := new(big.Rat).Sub(n, new(big.Rat).SetInt64(3))

	// g2*(n+1)+6
	x1 := new(big.Rat).Add(new(big.Rat).Mul(g2, nPlus1), new(big.Rat).SetInt64(6))

	numerator := new(big.Rat).Mul(x1, nMinus1)

	denominator := new(big.Rat).Mul(nMinus2, nMinus3)

	result := new(big.Rat).Quo(numerator, denominator)

	return result, true
}

func calculateKurtType3(dl insyra.IDataList) (*big.Rat, bool) {
	g2, ok := calculateKurtType1(dl)
	if !ok {
		return nil, false
	}

	g2Plus3 := new(big.Rat).Add(g2, new(big.Rat).SetInt64(3))

	nReciprocal := new(big.Rat).SetFloat64(1.0 / float64(dl.Len()))
	oneMinusNReciprocal := new(big.Rat).Sub(new(big.Rat).SetInt64(1), nReciprocal)
	oneMinusNReciprocalPow2 := new(big.Rat).Mul(oneMinusNReciprocal, oneMinusNReciprocal)

	result := new(big.Rat).Sub(new(big.Rat).Mul(g2Plus3, oneMinusNReciprocalPow2), new(big.Rat).SetInt64(3))
	return result, true
}