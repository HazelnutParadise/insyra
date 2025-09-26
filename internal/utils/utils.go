package utils

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

func ParseColIndex(colIndex string) int {
	result := 0
	for _, char := range colIndex {
		result = result*26 + int(char-'A') + 1
	}
	return result - 1
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

// ConvertDateFormat 將常見的日期格式模式轉換為 Go 語言的時間格式
func ConvertDateFormat(pattern string) string {
	// 常見的日期格式映射（支援大小寫）
	formatMap := map[string]string{
		"YYYY": "2006", // 四位年份（大寫）
		"yyyy": "2006", // 四位年份（小寫）
		"YY":   "06",   // 兩位年份（大寫）
		"yy":   "06",   // 兩位年份（小寫）
		"MM":   "01",   // 兩位月份（大寫）
		"mm":   "01",   // 兩位月份（小寫，在日期格式中通常指月份）
		"M":    "1",    // 一位月份（大寫）
		"m":    "1",    // 一位月份（小寫）
		"DD":   "02",   // 兩位日期（大寫）
		"dd":   "02",   // 兩位日期（小寫）
		"D":    "2",    // 一位日期（大寫）
		"d":    "2",    // 一位日期（小寫）
		"HH":   "15",   // 24小時制小時（大寫）
		"hh":   "15",   // 24小時制小時（小寫）
		"H":    "15",   // 24小時制小時（大寫）
		"h":    "15",   // 24小時制小時（小寫）
		"SS":   "05",   // 秒（大寫）
		"ss":   "05",   // 秒（小寫）
		"S":    "5",    // 秒（大寫）
		"s":    "5",    // 秒（小寫）
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
