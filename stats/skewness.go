package stats

import (
	"math/big"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/parallel"
)

// Skewness calculates the skewness(sample) of the DataList.
// Returns the skewness.
// Returns nil if the DataList is empty or the skewness cannot be calculated.
func Skewness(sample interface{}, method ...int) interface{} {
	d, dLen := insyra.ProcessData(sample)
	d64 := insyra.SliceToF64(d)
	insyra.LogDebug("stats.Skew(): d64: ", d64)
	dl := insyra.NewDataList(d64)
	insyra.LogDebug("stats.Skew(): dl: ", dl)

	usemethod := 1
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
	case 1:
		result = calculateSkewType1(dl)
	case 2:
		result = calculateSkewTyep2(dl)
	case 3:
		result = calculateSkewType3(dl)
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
func calculateSkewType1(dl *insyra.DataList, highPrecision ...bool) interface{} {
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

func calculateSkewTyep2(dl *insyra.DataList) interface{} {
	n := new(big.Rat).SetFloat64(conv.ParseF64(dl.Len()))
	g1 := calculateSkewType1(dl, true).(*big.Rat)

	// 计算 n(n-1)
	nMinus1 := new(big.Rat).Sub(n, new(big.Rat).SetInt64(1))
	numerator := new(big.Rat).Mul(n, nMinus1)

	// sqrt(n(n-1))
	numeratorSqrt := insyra.SqrtRat(numerator)

	// 计算 (n-2)
	nMinus2 := new(big.Rat).Sub(n, new(big.Rat).SetInt64(2))

	y := new(big.Rat).Quo(numeratorSqrt, nMinus2)

	// 计算最终的 Fisher-Pearson 偏度
	G1 := new(big.Rat).Mul(g1, y)
	f64G1, _ := G1.Float64()

	return f64G1
}

func calculateSkewType3(dl *insyra.DataList) interface{} {
	g1 := calculateSkewType1(dl, true).(*big.Rat)
	n := new(big.Rat).SetFloat64(conv.ParseF64(dl.Len()))

	// (n-1)/n
	y := new(big.Rat).Quo(new(big.Rat).Sub(n, new(big.Rat).SetFloat64(1)), n)

	yPowed := new(big.Rat).Mul(new(big.Rat).Mul(y, y), y)
	ySqrted := insyra.SqrtRat(yPowed)

	result := new(big.Rat).Mul(g1, ySqrted)

	f64Result, _ := result.Float64()

	return f64Result
}
