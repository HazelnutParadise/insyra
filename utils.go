package insyra

import (
	"math/big"

	"github.com/HazelnutParadise/Go-Utils/conv"
)

// ToFloat64 converts any numeric value to float64.
func ToFloat64(v interface{}) float64 {
	switch v := v.(type) {
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		return 0
	}
}

// ToFloat64Safe tries to convert any numeric value to float64 and returns a boolean indicating success.
func ToFloat64Safe(v interface{}) (float64, bool) {
	switch v := v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return ToFloat64(v), true
	default:
		return 0, false
	}
}

func SliceToF64(data []interface{}) []float64 {
	defer func() {
		if r := recover(); r != nil {
			LogWarning("SliceToF64(): Failed to convert data to float64")
		}
	}()
	var floatSlice []float64
	for _, v := range data {
		f64v := conv.ParseF64(v)              // 將 interface{} 轉換為 float64
		floatSlice = append(floatSlice, f64v) // 將 interface{} 轉換為 float64
	}

	return floatSlice
}

// ProcessData processes the input data and returns the data and the length of the data.
// Returns nil and 0 if the data type is unsupported.
// Supported data types are []interface{} and IDataList.
func ProcessData(input interface{}) ([]interface{}, int) {
	var data []interface{}

	// 根據類型判斷如何獲取資料
	switch v := input.(type) {
	case IDataList: // 使用介面來進行斷言
		data = v.Data()
	case []interface{}:
		data = v
	default:
		LogWarning("ProcessData(): Unsupported data type %T, returning nil.", input)
		return nil, 0
	}

	return data, len(data)
}

func SqrtRat(x *big.Rat) *big.Rat {
	// 將 *big.Rat 轉換為 *big.Float
	floatValue := new(big.Float).SetRat(x)

	// 計算平方根
	sqrtValue := new(big.Float).Sqrt(floatValue)

	// 將 *big.Float 轉換為 *big.Rat
	result := new(big.Rat)
	sqrtXRat, _ := sqrtValue.Rat(result)
	return sqrtXRat
}

// PowRat 計算 big.Rat 的次方 (v^n)
func PowRat(base *big.Rat, exponent int) *big.Rat {
	result := new(big.Rat).SetInt64(1) // 初始化為 1
	for i := 0; i < exponent; i++ {
		result.Mul(result, base) // result = result * base
	}
	return result
}
