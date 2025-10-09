package insyra

import (
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra/internal/utils"
)

type F64orRat = utils.F64orRat

var ToFloat64 = utils.ToFloat64
var ToFloat64Safe = utils.ToFloat64Safe

// SliceToF64 converts a []any to a []float64.
func SliceToF64(input []any) []float64 {
	var out []float64
	for _, v := range input {
		switch val := v.(type) {
		case float64:
			out = append(out, val)
		case int:
			out = append(out, float64(val))
		default:
			out = append(out, 0) // Substitute 0 for non-numeric values
		}
	}
	return out
}

// ProcessData processes the input data and returns the data and the length of the data.
// Returns nil and 0 if the data type is unsupported.
// Supported data types are slices, IDataList, and pointers to these types.
func ProcessData(input any) ([]any, int) {
	var data []any

	// 使用反射来处理数据类型
	value := reflect.ValueOf(input)

	// 处理指针类型，获取指针指向的元素
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Slice:
		// 遍历切片中的每一个元素
		for i := range value.Len() {
			element := value.Index(i).Interface()
			data = append(data, element)
		}
	case reflect.Interface:
		// 支持 IDataList 的断言
		if dl, ok := input.(IDataList); ok {
			data = dl.Data()
		} else {
			LogWarning("core", "ProcessData", "Unsupported data type %T, returning nil.", input)
			return nil, 0
		}
	case reflect.Array:
		// 如果需要支持数组类型，可以添加对 reflect.Array 的处理
		for i := range value.Len() {
			element := value.Index(i).Interface()
			data = append(data, element)
		}
	default:
		// 尝试类型断言为 IDataList
		if dl, ok := input.(IDataList); ok {
			data = dl.Data()
		} else {
			LogWarning("core", "ProcessData", "Unsupported data type %T, returning nil.", input)
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
	for range exponent {
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
	factorMap := make(map[any][][]float64)

	// 迭代資料，根據因子將觀測值和自變數分組
	for i := range factor.Len() {
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

	wideTable.SetColToRowNames("A")
	for i := range independents {
		wideTable.columns[i].name = fmt.Sprintf("%v", independentName[i])
	}
	wideTable.columns[len(independents)].name = data.GetName()

	return wideTable
}

// isNumeric checks if the value is numeric.
func IsNumeric(v any) bool {
	if v == nil {
		return false
	}

	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	}

	// 處理反射類型
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}

	return false
}

// ParseColIndex converts an Excel-like column name (e.g., "A", "Z", "AA") to its 0-based integer index.
func ParseColIndex(colName string) int {
	return utils.ParseColIndex(colName)
}

// SortTimes sorts a slice of time.Time in ascending order.
// It sorts the times directly in the provided slice.
func SortTimes(times []time.Time) {
	utils.ParallelSortStableFunc(times, func(a, b time.Time) int {
		if a.Before(b) {
			return -1
		} else if a.After(b) {
			return 1
		}
		return 0
	})
}
