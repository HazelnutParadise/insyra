package utils

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

type F64orRat interface {
	float64 | *big.Rat
}

// ToFloat64 converts any numeric value to float64.
func ToFloat64(v any) float64 {
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
func ToFloat64Safe(v any) (float64, bool) {
	switch v := v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return ToFloat64(v), true
	default:
		return 0, false
	}
}

func ParseColIndex(colIndex string) (colNumber int, ok bool) {
	result := 0
	for _, char := range colIndex {
		if char < 'A' || char > 'Z' {
			return -1, false
		}
		result = result*26 + int(char-'A') + 1
	}
	return result - 1, true
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
			return "'" + lines[0] + "...'"
		}
		return "'" + v + "'"

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
			// 對於長度 <= 3 的陣列，使用逗號分隔的格式
			var elements []string
			for i := range length {
				elements = append(elements, fmt.Sprintf("%v", rv.Index(i).Interface()))
			}
			return "[" + strings.Join(elements, ", ") + "]"
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

// convertToDateString 將日期值或時間戳轉換為指定格式的字串（使用Go時間格式）
func ConvertToDateString(value any, goDateFormat string) string {
	switch v := value.(type) {
	case time.Time:
		return v.Format(goDateFormat)
	case int:
		int64value := int64(v)
		return convertTimestampToString(int64value, goDateFormat)
	case int32:
		int64value := int64(v)
		return convertTimestampToString(int64value, goDateFormat)
	case int64:
		return convertTimestampToString(v, goDateFormat)
	case float64:
		if v > 0 && v < 100000 {
			// Excel date serial number (with time fraction)
			days := int(v)
			frac := v - float64(days)
			base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
			t := base.AddDate(0, 0, days)
			totalSeconds := frac * 86400
			hours := int(totalSeconds / 3600)
			minutes := int((totalSeconds - float64(hours)*3600) / 60)
			seconds := int(totalSeconds - float64(hours)*3600 - float64(minutes)*60)
			t = time.Date(t.Year(), t.Month(), t.Day(), hours, minutes, seconds, 0, time.UTC)
			return t.Format(goDateFormat)
		} else {
			// Not supported, convert to string
			return fmt.Sprintf("%v", v)
		}
	case string:
		return v
	default:
		// 嘗試轉為字串
		return fmt.Sprintf("%v", v)
	}
}

// convertTimestampToString converts various timestamp formats to date string
func convertTimestampToString(ts int64, goDateFormat string) string {
	if ts > 0 && ts < 100000 {
		// Excel date serial number
		base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
		t := base.AddDate(0, 0, int(ts))
		return t.Format(goDateFormat)
	} else if ts >= 1000000000000 && ts < 100000000000000 { // 13 digits, milliseconds
		// Unix timestamp in milliseconds
		t := time.Unix(0, ts*int64(time.Millisecond)).UTC()
		return t.Format(goDateFormat)
	} else if ts >= 1000000000000000000 { // 19 digits, nanoseconds
		// Unix timestamp in nanoseconds
		t := time.Unix(0, ts).UTC()
		return t.Format(goDateFormat)
	} else {
		// Unix timestamp in seconds
		t := time.Unix(ts, 0).UTC()
		return t.Format(goDateFormat)
	}
}

// IsColorSupported 檢測當前終端是否支持 ANSI 顏色代碼
func IsColorSupported() bool {
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

// ConvertDateFormat 將常見的日期格式模式轉換為 Go 語言的時間格式
func ConvertDateFormat(pattern string) string {
	// 常見的日期格式映射（支援大小寫）
	formatMap := map[string]string{
		"YYYY": "2006", // 四位年份（大寫）
		"yyyy": "2006", // 四位年份（小寫）
		"YY":   "06",   // 兩位年份（大寫）
		"yy":   "06",   // 兩位年份（小寫）

		// 月份（使用大寫 M 表示月份）
		"MM": "01", // 兩位月份（大寫）
		"M":  "1",  // 一位月份（大寫）

		// 分鐘（使用小寫 m 表示分鐘）
		"mm": "04", // 兩位分鐘（小寫）
		"m":  "4",  // 一位分鐘（小寫）

		// 日期
		"DD": "02", // 兩位日期（大寫）
		"dd": "02", // 兩位日期（小寫）
		"D":  "2",  // 一位日期（大寫）
		"d":  "2",  // 一位日期（小寫）

		// 小時（24小時制）
		"HH": "15", // 24小時制小時（大寫）
		"hh": "15", // 24小時制小時（小寫）
		"H":  "15", // 24小時制小時（大寫）
		"h":  "15", // 24小時制小時（小寫）

		// 秒
		"SS": "05", // 秒（大寫）
		"ss": "05", // 秒（小寫）
		"S":  "5",  // 秒（大寫）
		"s":  "5",  // 秒（小寫）
	}

	result := pattern
	// 注意：要先替換長的模式，避免部分替換
	orderedKeys := []string{"YYYY", "yyyy", "YY", "yy", "MM", "mm", "DD", "dd", "HH", "hh", "SS", "ss", "M", "m", "D", "d", "H", "h", "S", "s"}

	for _, key := range orderedKeys {
		if val, exists := formatMap[key]; exists {
			result = strings.ReplaceAll(result, key, val)
		}
	}

	return result
}

// GetTypeSortingRank returns the type rank for sorting mixed types.
// Lower rank means higher priority (comes first in ascending order).
func GetTypeSortingRank(v any) int {
	if v == nil {
		return 0
	}
	switch v.(type) {
	case bool:
		return 1
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return 2
	case string:
		return 3
	case time.Time:
		return 4
	default:
		return 5
	}
}

// CompareAny compares two values of any type and returns:
// -1 if a < b
//
//	0 if a == b
//	1 if a > b
//
// It uses type ranking and type-specific comparison logic.
func CompareAny(a, b any) int {
	typeRankA := GetTypeSortingRank(a)
	typeRankB := GetTypeSortingRank(b)
	if typeRankA != typeRankB {
		return typeRankA - typeRankB
	}
	// Same type rank, compare values
	var cmp int
	switch va := a.(type) {
	case string:
		if vb, ok := b.(string); ok {
			cmp = strings.Compare(va, vb)
		} else {
			cmp = strings.Compare(va, fmt.Sprint(b))
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		fa := ToFloat64(a)
		if fb, ok := ToFloat64Safe(b); ok {
			if fa < fb {
				cmp = -1
			} else if fa > fb {
				cmp = 1
			} else {
				cmp = 0
			}
		} else {
			cmp = strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
		}
	case time.Time:
		if vb, ok := b.(time.Time); ok {
			if va.Before(vb) {
				cmp = -1
			} else if va.After(vb) {
				cmp = 1
			} else {
				cmp = 0
			}
		} else {
			cmp = strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
		}
	default:
		cmp = strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
	}
	return cmp
}

// ParallelSortStableFunc sorts the slice x in ascending order as determined by the cmp function.
// It is a parallelized version of slices.SortStableFunc, using goroutines to improve performance on large datasets.
// The function maintains stability: equal elements preserve their original order.
// This optimized version uses adaptive goroutines scaling and improved chunking strategy.
func ParallelSortStableFunc[S ~[]E, E any](x S, cmp func(E, E) int) {
	n := len(x)
	if n <= 1 {
		return
	}

	// Use sequential sort for small arrays
	if n < 4910 {
		slices.SortStableFunc(x, cmp)
		return
	}

	// Determine optimal number of goroutines based on data size
	numGoroutines := min(getOptimalGoroutines(n), runtime.NumCPU())

	// Sort chunks in parallel using the same logic as the default version
	sortChunksOptimized(x, cmp, numGoroutines)

	// Merge chunks using the original stable merge
	ParallelMergeStable(x, cmp, numGoroutines)
}

// sortChunksOptimized sorts data chunks in parallel with consistent chunking
func sortChunksOptimized[S ~[]E, E any](x S, cmp func(E, E) int, numChunks int) {
	n := len(x)
	chunkSize := n / numChunks
	if chunkSize == 0 {
		chunkSize = 1
	}

	var wg sync.WaitGroup
	for i := range numChunks {
		start := i * chunkSize
		end := start + chunkSize
		if i == numChunks-1 {
			end = n
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			slices.SortStableFunc(x[start:end], cmp)
		}(start, end)
	}
	wg.Wait()
}

// ParallelMergeStable merges the sorted chunks in the slice x.
// It assumes x is divided into numChunks sorted sub-slices.
func ParallelMergeStable[S ~[]E, E any](x S, cmp func(E, E) int, numChunks int) {
	n := len(x)
	if numChunks <= 1 {
		return
	}

	chunkSize := n / numChunks
	temp := make(S, n)
	copy(temp, x)

	// Merge pairs of chunks
	for size := 1; size < numChunks; size *= 2 {
		for left := 0; left < numChunks-size; left += 2 * size {
			mid := left + size
			right := min(left+2*size, numChunks)

			leftStart := left * chunkSize
			midStart := mid * chunkSize
			rightEnd := right * chunkSize
			if right == numChunks {
				rightEnd = n
			}

			mergeStable(temp[leftStart:midStart], temp[midStart:rightEnd], x[leftStart:rightEnd], cmp)
		}
		copy(temp, x)
	}
}

// mergeStable merges two sorted slices a and b into dst, maintaining stability.
func mergeStable[S ~[]E, E any](a, b, dst S, cmp func(E, E) int) {
	i, j, k := 0, 0, 0
	for i < len(a) && j < len(b) {
		if cmp(a[i], b[j]) <= 0 {
			dst[k] = a[i]
			i++
		} else {
			dst[k] = b[j]
			j++
		}
		k++
	}
	for i < len(a) {
		dst[k] = a[i]
		i++
		k++
	}
	for j < len(b) {
		dst[k] = b[j]
		j++
		k++
	}
}

// getOptimalGoroutines returns the optimal number of goroutines for a given data size
func getOptimalGoroutines(n int) int {
	// Adaptive growth strategy: slow growth for small datasets, faster growth for large datasets
	if n < 10000 {
		// For small datasets: use logarithmic-like growth
		if n < 5500 {
			return 2
		} else if n < 6500 {
			return 3
		} else if n < 7500 {
			return 4
		} else if n < 8500 {
			return 5
		} else {
			return 6
		}
	} else if n < 50000 {
		// For medium datasets: moderate linear growth
		return 6 + (n-10000)/5000 // Increases by 1 every 5000 elements
	} else if n < 200000 {
		// For large datasets: accelerated growth
		return 12 + (n-50000)/15000 // Increases by 1 every 15000 elements
	} else {
		// For very large datasets: use square root scaling
		goroutines := int(math.Sqrt(float64(n)) / 50) // sqrt(n)/50 gives good scaling
		if goroutines > runtime.NumCPU() {
			goroutines = runtime.NumCPU()
		}
		if goroutines < 16 {
			goroutines = 16 // Minimum for very large datasets
		}
		return goroutines
	}
}

// Get terminal window width
func GetTerminalWidth() int {
	width := 80 // Default width

	// Try to get terminal window size
	fd := int(os.Stdout.Fd())
	if w, _, err := term.GetSize(fd); err == nil && w > 0 {
		width = w
	}

	return width
}
