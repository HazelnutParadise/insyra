package stats

import (
	"fmt"
	"math/big"

	"github.com/HazelnutParadise/insyra"
)

// calculateMoment 函數用於計算 n 階矩，使用 big.Rat 進行高精度計算
// 如果 central 為 true，則計算中心矩（基於均值）；如果為 false，則計算非中心矩（基於 0）。
func calculateMoment(dl *insyra.DataList, n int, central bool) (*big.Rat, error) {
	// 檢查數據長度
	if dl.Len() == 0 {
		return nil, fmt.Errorf("數據集不能為空")
	}

	// 初始化均值
	mean := new(big.Rat)
	if central {
		mean = dl.Mean(true).(*big.Rat) // 計算均值
	}

	// 初始化 n 階矩
	moment := new(big.Rat)

	// 遍歷每個數據點
	for _, v := range dl.Data() {
		val := new(big.Rat).SetFloat64(v.(float64))  // 轉換為 big.Rat
		deviation := new(big.Rat).Sub(val, mean)     // (v - 均值)
		deviationPowN := insyra.PowRat(deviation, n) // 計算 (v - 均值)^n
		moment.Add(moment, deviationPowN)            // 累加到 moment
	}

	// 計算 n 階矩的平均值
	length := new(big.Rat).SetInt64(int64(dl.Len()))
	moment.Quo(moment, length) // moment / len(data)

	return moment, nil
}
