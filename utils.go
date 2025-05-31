package insyra

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/mattn/go-runewidth"
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

	wideTable.SetColToRowNames("A")
	for i, _ := range independents {
		wideTable.columns[i].name = fmt.Sprintf("%v", independentName[i])
	}
	wideTable.columns[len(independents)].name = data.GetName()

	return wideTable
}

// TruncateString 截斷字符串到指定寬度，太長的字符串末尾加上省略號，使用 runewidth 計算字元寬度
func TruncateString(s string, maxLength int) string {
	// 總寬度小於等於限制，直接返回
	if runewidth.StringWidth(s) <= maxLength {
		return s
	}
	// 若限制過小，按 rune 長度裁剪
	if maxLength <= 3 {
		rs := []rune(s)
		if len(rs) <= maxLength {
			return s
		}
		return string(rs[:maxLength])
	}
	// 留出 3 單位寬度給省略號
	width := 0
	var b strings.Builder
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if width+rw > maxLength-3 {
			break
		}
		b.WriteRune(r)
		width += rw
	}
	return b.String() + "..."
}

// FormatValue 根據值的類型格式化輸出，改善顯示效果
func FormatValue(value any) string {
	if value == nil {
		return "nil"
	}

	switch v := value.(type) {
	case float64:
		// 處理特殊浮點數
		if math.IsNaN(v) {
			return "NaN"
		}
		if math.IsInf(v, 1) {
			return "+Inf"
		}
		if math.IsInf(v, -1) {
			return "-Inf"
		}

		// 針對整數值的浮點數使用整數格式
		if v == float64(int(v)) {
			return fmt.Sprintf("%d", int(v))
		}

		// 根據大小動態調整小數位數
		if math.Abs(v) < 0.0001 || math.Abs(v) >= 10000 {
			return fmt.Sprintf("%.4e", v) // 科學計數法
		}

		// 顯示數字，但不顯示尾部的零
		s := fmt.Sprintf("%.4f", v)
		s = strings.TrimRight(s, "0")
		return strings.TrimRight(s, ".")

	case float32:
		return FormatValue(float64(v))

	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)

	case bool:
		if v {
			return "true"
		}
		return "false"

	case string:
		// 如果是多行字符串，只顯示第一行
		if strings.Contains(v, "\n") {
			lines := strings.Split(v, "\n")
			return lines[0] + "..."
		}
		return v

	case []byte:
		if len(v) > 20 {
			return fmt.Sprintf("%x... (%d bytes)", v[:10], len(v))
		}
		return fmt.Sprintf("%x", v)

	case time.Time:
		return v.Format("2006-01-02 15:04:05")

	default:
		// 檢測是否是數組或切片類型
		rv := reflect.ValueOf(value)
		kind := rv.Kind()

		if kind == reflect.Slice || kind == reflect.Array {
			length := rv.Len()
			if length == 0 {
				return "[]"
			}
			if length > 3 {
				return fmt.Sprintf("[%v, %v, ... +%d]",
					rv.Index(0).Interface(),
					rv.Index(1).Interface(),
					length-2)
			}
			return fmt.Sprintf("%v", value)
		}

		// 檢測是否是 map 類型
		if kind == reflect.Map {
			size := rv.Len()
			if size == 0 {
				return "{}"
			}
			return fmt.Sprintf("{...%d keys}", size)
		}

		// 檢測是否是結構體
		if kind == reflect.Struct {
			typeName := reflect.TypeOf(value).String()
			return fmt.Sprintf("<%s>", typeName)
		}
		// 其他類型使用默認格式化
		return fmt.Sprintf("%v", value)
	}
}

// isNumeric 檢查一個值是否為數值類型
func isNumeric(v interface{}) bool {
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

// ColorText 根據環境支持自動決定是否添加顏色到文本
// code 是 ANSI 顏色代碼，text 是要設置顏色的文本
func ColorText(code string, text interface{}) string {
	if isColorSupported() {
		return fmt.Sprintf("\033[%sm%v\033[0m", code, text)
	}
	return fmt.Sprintf("%v", text)
}

// isColorSupported 檢測當前終端是否支持 ANSI 顏色代碼
func isColorSupported() bool {
	// 檢測 NO_COLOR 環境變量
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// 檢測 TERM 環境變量
	term := os.Getenv("TERM")
	if term == "dumb" {
		return false
	}

	// 檢測是否是 Windows 並判斷控制台類型
	if runtime.GOOS == "windows" {
		// Windows 10 1909 之後的版本支持 ANSI 顏色
		// 這裡使用簡單的判斷方式，實際上可能需要更複雜的檢測
		return true
	}

	// 大多數 Unix-like 系統默認支持 ANSI 顏色
	return true
}

// parseColIndex converts an Excel-like column name (e.g., "A", "Z", "AA") to its 0-based integer index.
func parseColIndex(colName string) int {
	result := 0
	for _, char := range colName {
		result = result*26 + int(char-'A') + 1
	}
	return result - 1
}
