package stats

import (
	"math"
	"math/big"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/parallel"
)

// SkewnessMode is an enum type for skewness calculation mode.
type SkewnessMode int

const (
	// PearsonFirst represents Type 1 skewness calculation mode.
	Skew_PearsonFirst SkewnessMode = iota

	// FisherPearson represents Type 2 skewness calculation mode.
	Skew_FisherPearson

	// AdjustedFisherPearson represents Type 3 skewness calculation mode.
	Skew_AdjustedFisherPearson
)

// Skew calculates the skewness(sample) of the DataList.
// Returns the skewness.
// Returns nil if the DataList is empty or the skewness cannot be calculated.
func Skew(sample interface{}, method ...SkewnessMode) interface{} {
	d, dLen := insyra.ProcessData(sample)
	d64 := insyra.SliceToF64(d)
	insyra.LogDebug("stats.Skew(): d64: ", d64)
	dl := insyra.NewDataList(d64)
	insyra.LogDebug("stats.Skew(): dl: ", dl)

	usemethod := Skew_PearsonFirst
	if len(method) > 0 {
		usemethod = method[0]
	}
	if len(method) > 1 {
		insyra.LogWarning("stats.Skew(): More than one method specified, returning nil.")
		return nil
	}
	if dLen == 0 {
		insyra.LogWarning("stats.Skew(): DataList is empty, returning nil.")
		return nil
	}

	var result interface{}
	switch usemethod {
	case Skew_PearsonFirst:
		result = calculateSkewPearsonFirst(dl)
	case Skew_FisherPearson:
		result = calculateSkewFisherPearson(dl)
	case Skew_AdjustedFisherPearson:
		result = calculateSkewAdjustedFisherPearson(dl)
	default:
		insyra.LogWarning("stats.Skew(): Invalid method, returning nil.")
		return nil
	}

	if result == nil {
		insyra.LogWarning("stats.Skew(): Skewness is nil, returning nil.")
		return nil
	}
	resultFloat, ok := result.(float64)
	if !ok {
		insyra.LogWarning("stats.Skew(): Skewness is not a float64, returning nil.")
		return nil
	}
	return resultFloat
}

// ======================== calculation functions ========================
func calculateSkewPearsonFirst(dl *insyra.DataList, highPrecision ...bool) interface{} {
	n := new(big.Rat).SetFloat64(conv.ParseF64(dl.Len()))
	nReciprocal := new(big.Rat).Inv(n)
	m1 := dl.Mean(true).(*big.Rat)
	toM2Fn := func() *big.Rat {
		var m2Cal = new(big.Rat)
		for _, v := range dl.Data() {
			vRat := new(big.Rat).SetFloat64(v.(float64))
			vRat.Sub(vRat, m1)
			vRat.Mul(vRat, vRat)
			m2Cal.Add(m2Cal, vRat)
		}
		return m2Cal
	}
	toM3Fn := func() *big.Rat {
		var m3Cal = new(big.Rat)
		for _, v := range dl.Data() {
			vRat := new(big.Rat).SetFloat64(v.(float64))
			vRat.Sub(vRat, m1)
			v2 := new(big.Rat).Mul(vRat, vRat)
			vRat.Mul(vRat, v2)
			m3Cal.Add(m3Cal, vRat)
		}
		return m3Cal
	}

	results := parallel.GroupUp(toM2Fn, toM3Fn).Run().AwaitResult()

	m2 := new(big.Rat).Mul(nReciprocal, results[0][0].(*big.Rat))
	m3 := new(big.Rat).Mul(nReciprocal, results[1][0].(*big.Rat))

	m2Powed := new(big.Rat).Mul(m2, m2)
	m2Powed = new(big.Rat).Mul(m2Powed, m2)
	m2Sqrted := insyra.SqrtRat(m2Powed)

	g1 := new(big.Rat).Quo(m3, m2Sqrted)
	if len(highPrecision) > 0 && highPrecision[0] {
		return g1
	}

	f64g1, _ := g1.Float64()

	return f64g1
}

// 錯誤
func calculateSkewFisherPearson(dl *insyra.DataList) interface{} {
	n := float64(dl.Len())

	// 檢查樣本大小是否小於 3
	if n < 3 {
		insyra.LogWarning("calculateSkewFisherPearson: Sample size too small to calculate skewness, returning nil.")
		return nil
	}

	// 計算均值
	m1 := dl.Mean(false).(float64)

	// 計算二階矩 (m2) 和 三階矩 (m3)
	m2, m3 := 0.0, 0.0
	for _, v := range dl.Data() {
		delta := v.(float64) - m1
		deltaSquared := delta * delta
		m2 += deltaSquared
		m3 += deltaSquared * delta
	}

	// 除以樣本數 n 得到 m2 和 m3
	m2 /= n
	m3 /= n

	// 計算 Type 1 偏度 (g1)
	g1 := m3 / math.Pow(m2, 1.5)

	// 計算 Fisher-Pearson 修正項
	correctionFactor := math.Sqrt(n * (n - 1) / (n - 2))

	// 最終的 Fisher-Pearson 偏度
	G1 := g1 * correctionFactor

	return G1
}

func calculateSkewAdjustedFisherPearson(dl *insyra.DataList) interface{} {
	g1 := calculateSkewPearsonFirst(dl, true).(*big.Rat)
	n := new(big.Rat).SetFloat64(conv.ParseF64(dl.Len()))

	// (n-1)/n
	y := new(big.Rat).Quo(new(big.Rat).Sub(n, new(big.Rat).SetFloat64(1)), n)

	yPowed := new(big.Rat).Mul(new(big.Rat).Mul(y, y), y)
	ySqrted := insyra.SqrtRat(yPowed)

	result := new(big.Rat).Mul(g1, ySqrted)

	f64Result, _ := result.Float64()

	return f64Result
}
