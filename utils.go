package insyra

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/HazelnutParadise/Go-Utils/conv"
)

type F64orRat interface {
	float64 | *big.Rat
}

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

// SliceToF64 converts a []interface{} to a []float64.
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
// Supported data types are slices, IDataList, and pointers to these types.
func ProcessData(input interface{}) ([]interface{}, int) {
	var data []interface{}

	// 使用反射来处理数据类型
	value := reflect.ValueOf(input)

	// 处理指针类型，获取指针指向的元素
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Slice:
		// 遍历切片中的每一个元素
		for i := 0; i < value.Len(); i++ {
			element := value.Index(i).Interface()
			data = append(data, element)
		}
	case reflect.Interface:
		// 支持 IDataList 的断言
		if dl, ok := input.(IDataList); ok {
			data = dl.Data()
		} else {
			LogWarning("ProcessData(): Unsupported data type %T, returning nil.", input)
			return nil, 0
		}
	case reflect.Array:
		// 如果需要支持数组类型，可以添加对 reflect.Array 的处理
		for i := 0; i < value.Len(); i++ {
			element := value.Index(i).Interface()
			data = append(data, element)
		}
	default:
		// 尝试类型断言为 IDataList
		if dl, ok := input.(IDataList); ok {
			data = dl.Data()
		} else {
			LogWarning("ProcessData(): Unsupported data type %T, returning nil.", input)
			return nil, 0
		}
	}

	return data, len(data)
}

// SqrtRat calculates the square root of a big.Rat.
// 計算 big.Rat 的平方根
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

// PowRat calculates the power of a big.Rat.
// 計算 big.Rat 的次方 (v^n)
func PowRat(base *big.Rat, exponent int) *big.Rat {
	result := new(big.Rat).SetInt64(1) // 初始化為 1
	for i := 0; i < exponent; i++ {
		result.Mul(result, base) // result = result * base
	}
	return result
}

// ConvertLongDataToWide converts long data to wide data.
// data: 觀測值(依變數)
// factor: 因子
// independents: 自變數
// aggFunc: 聚合函數
// Return DataTable.
func ConvertLongDataToWide(data, factor IDataList, independents []IDataList, aggFunc func([]float64) float64) IDataTable {

	// 檢查是否存在空的 factor 或 data
	if factor == nil || data == nil {
		fmt.Println("Factor or data is nil")
		return nil
	}

	var independentName []string
	for _, independent := range independents {
		independentName = append(independentName, independent.GetName())
	}

	wideTable := NewDataTable()

	// 建立一個 map 儲存每個因子對應的觀測值及自變數
	factorMap := make(map[interface{}][][]float64)

	// 迭代資料，根據因子將觀測值和自變數分組
	for i := 0; i < factor.Len(); i++ {
		key := factor.Get(i)

		// 如果該因子尚未在 map 中，初始化空的觀測值和自變數切片
		if _, exists := factorMap[key]; !exists {
			factorMap[key] = make([][]float64, 0)
		}

		// 收集自變數的值
		row := make([]float64, len(independents))
		for j, independent := range independents {
			row[j] = conv.ParseF64(independent.Get(i))
		}

		// 加入觀測值
		value := conv.ParseF64(data.Get(i))
		row = append(row, value)

		// 將該行資料加入對應的因子組
		factorMap[key] = append(factorMap[key], row)
	}

	// 如果提供的聚合函數為 nil，則將資料直接轉換為寬資料
	if aggFunc == nil {
		aggFunc = func(values []float64) float64 {
			if len(values) > 0 {
				return values[0] // 直接返回第一筆資料
			}
			return 0.0
		}
	}

	// 將因子映射到寬資料表
	for key, rows := range factorMap {
		row := NewDataList()

		// 將因子值作為行標題
		row.Append(key)

		// 依次處理每個自變數
		for j := 0; j < len(independents); j++ {
			columnValues := make([]float64, len(rows))
			for i, r := range rows {
				columnValues[i] = r[j]
			}
			// 聚合自變數的數據
			row.Append(aggFunc(columnValues))
		}

		// 聚合觀測值
		observations := make([]float64, len(rows))
		for i, r := range rows {
			observations[i] = r[len(independents)] // 最後一列是觀測值
		}
		row.Append(aggFunc(observations))

		// 將結果加入寬資料表
		wideTable.AppendRowsFromDataList(row)
	}

	wideTable.SetColumnToRowNames("A")
	for i, _ := range independents {
		wideTable.columns[i].name = fmt.Sprintf("%v", independentName[i])
	}
	wideTable.columns[len(independents)].name = data.GetName()

	return wideTable
}
