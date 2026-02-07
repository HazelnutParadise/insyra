package utils

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"reflect"
	"runtime"
	"strings"
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
	// Reject empty input
	if len(colIndex) == 0 {
		return -1, false
	}
	result := 0
	// Process bytes directly to avoid allocation from strings.ToUpper
	for i := 0; i < len(colIndex); i++ {
		c := colIndex[i]
		var v int
		if c >= 'A' && c <= 'Z' {
			v = int(c - 'A' + 1)
		} else if c >= 'a' && c <= 'z' {
			v = int(c - 'a' + 1)
		} else {
			return -1, false
		}
		// Overflow check: ensure result*26 + v fits into int
		maxInt := int(^uint(0) >> 1)
		if result > (maxInt-v)/26 {
			return -1, false
		}
		result = result*26 + v
	}
	return result - 1, true
}

func CalcColIndex(colNumber int) (colIndex string, ok bool) {
	if colNumber < 0 {
		return "", false
	}

	// 使用單次迴圈與固定大小暫存陣列（最多 20 字元，足以容納 64-bit int）
	var tmpBuf [20]byte
	i := 0
	for colNumber >= 0 {
		remainder := colNumber % 26
		tmpBuf[i] = byte(remainder) + 'A'
		i++
		colNumber = (colNumber / 26) - 1
	}

	// 反向複製到結果切片（避免多次分配與 rune 轉換）
	res := make([]byte, i)
	for j := 0; j < i; j++ {
		res[j] = tmpBuf[i-1-j]
	}
	return string(res), true
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

// TryParseTime attempts to parse common date/time string formats and returns
// the parsed time and true on success. Exported so other packages can reuse it.
func TryParseTime(str string) (time.Time, bool) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05 -0700 MST",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, str); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
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
