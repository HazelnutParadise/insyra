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
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

type F64orRat interface {
	float64 | *big.Rat
}

// The bounds of int64 as float64 values. 2^63 is the first value past the top
// of the range and is exactly representable, so a float is convertible when it
// is at least minInt64AsFloat and strictly below twoToThe63.
const (
	minInt64AsFloat = -9223372036854775808.0
	twoToThe63      = 9223372036854775808.0
)

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
		// A named type over a numeric kind — `type Celsius float64` — does not
		// match any case above, but IsNumeric (which goes through reflection)
		// calls it a number. One of the two had to be wrong about every
		// user-defined numeric type; the reflect fallback is the same shape
		// accel's projection settled on, and it only runs for values the type
		// switch already missed.
		f, _ := reflectToFloat64(v)
		return f
	}
}

// reflectToFloat64 converts a named type over a numeric kind. It reports false
// for anything else, including nil.
func reflectToFloat64(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	}
	return 0, false
}

// ToFloat64Safe tries to convert any numeric value to float64 and returns a boolean indicating success.
func ToFloat64Safe(v any) (float64, bool) {
	switch v := v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return ToFloat64(v), true
	default:
		return reflectToFloat64(v)
	}
}

func ParseColIndex(colIndex string) (colNumber int, ok bool) {
	// Reject empty input
	if len(colIndex) == 0 {
		return -1, false
	}
	// Accumulate the 0-based index directly rather than the 1-based value and
	// subtracting at the end: the largest index CalcColIndex can produce needs
	// a 1-based value of maxInt+1, so the old form rejected its own output.
	// Going straight to 0-based, z_next = z*26 + (v + 25).
	const maxInt = int(^uint(0) >> 1)
	z := 0
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
		if i == 0 {
			z = v - 1
			continue
		}
		step := v + 25
		if z > (maxInt-step)/26 {
			return -1, false
		}
		z = z*26 + step
	}
	return z, true
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
	// 負的寬度沒有意義，而且會讓下面的 rs[:maxLength] 以負邊界切片而 panic。
	// 當成 0 處理，回傳空字串。
	if maxLength < 0 {
		maxLength = 0
	}
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

		// 針對整數值的浮點數使用整數格式。int(v) 對超出範圍的浮點數在 Go 裡
		// 沒有定義結果（amd64 得到 MinInt64，arm64 飽和到 MaxInt64），過去
		// 2^63 在 Mac 上會印成 9223372036854775807，比實際值少 1，而在
		// Linux／Windows 上印成指數形式。轉換前先確認落在 int64 範圍內：
		// 上界寫成嚴格小於 2^63，因為 float64(math.MaxInt64) 會進位成 2^63。
		if v >= minInt64AsFloat && v < twoToThe63 && v == math.Trunc(v) {
			return fmt.Sprintf("%d", int64(v))
		}

		// 根據大小動態調整小數位數
		if math.Abs(v) < 0.0001 || math.Abs(v) >= 10000 {
			return fmt.Sprintf("%.4e", v) // 科學計數法
		}

		// 顯示數字，但不顯示尾部的零
		s := fmt.Sprintf("%.4f", v)
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
		// 四捨五入到小數第四位後，9999.99999 會變成 "10000"，把一個不是整數的
		// 值顯示成整數。近似值帶著小數點還看得出是近似，整數不會，所以這種情況
		// 改用完整表示法。
		if !strings.ContainsAny(s, ".eE") {
			return FloatText(v, 64)
		}
		return s

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
		// 不是合法的 UTF-8 就不是文字。Parquet 的 Binary 欄位會以字串形式進到
		// 格子裡（DataList 的格子放不了 slice），把那些位元組加引號交給終端機，
		// 除了看不懂之外還會讓整列歪掉：runewidth 把 NUL 算 0 欄寬、把非法位元組
		// 算成一個 U+FFFD，而終端機實際畫幾欄由它自己決定，實測差一欄。改走
		// []byte 的顯示方式，同一條「這是位元組不是文字」的規則，而且十六進位
		// 是 ASCII，欄寬算得準。這個檢查要在多行判斷之前，因為位元組裡剛好有
		// 0x0a 不代表它是多行字串。
		if !utf8.ValidString(v) {
			return FormatValue([]byte(v))
		}
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

		// 值自己知道怎麼轉成文字就用它的文字。放在 struct 分支之前，因為
		// 型別名稱對讀表格的人沒有用：一個從 Parquet 讀進來的十進位欄位
		// 整欄印成 <decimal.Decimal>，值就看不見了。上面有專屬 case 的型別
		// （time.Time 等）走不到這裡，行為不變。
		if s, ok := value.(fmt.Stringer); ok {
			return s.String()
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
	} else if ts >= 1000000000000 && ts < 1000000000000000 { // 13-15 digits, milliseconds
		// Unix timestamp in milliseconds. time.UnixMilli rather than
		// time.Unix(0, ts*int64(time.Millisecond)): that multiplication
		// overflows int64 above 9223372036854 ms (about 2262-04-11), which is
		// well inside the range this branch accepts, and silently produced a
		// different date.
		t := time.UnixMilli(ts).UTC()
		return t.Format(goDateFormat)
	} else if ts >= 1000000000000000 && ts < 1000000000000000000 { // 16-18 digits, microseconds
		// Without this window a 16-digit stamp was read as seconds, which put a
		// 2023 date in the year 53872.
		t := time.UnixMicro(ts).UTC()
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
//
// Both "-" and "/" date separators are accepted, with or without a time part
// and with or without a zone; a layout that carries no zone is read as UTC.
// Strings that are not dates (plain numbers, words) never match, so callers
// such as CCL can use it to probe a value.
func TryParseTime(str string) (time.Time, bool) {
	// Longest first, so "2006-01-02 15:04:05" is not truncated by a shorter
	// layout. Layouts without a zone are read as UTC (time.Parse's rule).
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006/01/02 15:04",
		"2006-01-02",
		"2006/01/02",
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

// ConvertDateFormat turns a common date-format pattern into a Go time layout.
//
// It scans left to right, taking a maximal run of one letter as a single token,
// so "MMM" is a three-letter month rather than three separate months. Text in
// square brackets is copied out verbatim, which is the only way to put a letter
// in the output without it being read as a token. Everything else — separators,
// digits, spaces — is copied as it is.
//
// It used to run a sequence of ReplaceAll over the whole string, which rewrote
// letters wherever they appeared: "MMM" became "011" and "Date:" became "2ate:".
//
// Recognised tokens:
//
//	YYYY yyyy  four-digit year      YY yy  two-digit year
//	MMMM       full month name      MMM    short month name
//	MM M       month number         DD dd D d  day of month
//	HH H       hour, 24-hour        hh h   hour (see below)
//	mm m       minute               SS ss S s  second
//	A a        AM/PM                [text]     literal text
//
// `hh` and `h` map to the 24-hour layout rather than the 12-hour one most
// conventions give them. That predates this rewrite and callers pass their own
// patterns, so it is left alone rather than silently changing what their charts
// print.
func ConvertDateFormat(pattern string) string {
	// Keyed by the letter and how many times it repeats.
	runs := map[byte]map[int]string{
		'Y': {4: "2006", 2: "06"},
		'y': {4: "2006", 2: "06"},
		'M': {4: "January", 3: "Jan", 2: "01", 1: "1"},
		'D': {2: "02", 1: "2"},
		'd': {2: "02", 1: "2"},
		'H': {2: "15", 1: "15"},
		'h': {2: "15", 1: "15"},
		'm': {2: "04", 1: "4"},
		'S': {2: "05", 1: "5"},
		's': {2: "05", 1: "5"},
		'A': {1: "PM"},
		'a': {1: "pm"},
	}

	isLetter := func(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

	var b strings.Builder
	for i := 0; i < len(pattern); {
		c := pattern[i]

		// Literal text.
		if c == '[' {
			if end := strings.IndexByte(pattern[i:], ']'); end >= 0 {
				b.WriteString(pattern[i+1 : i+end])
				i += end + 1
				continue
			}
			// No closing bracket: nothing to escape, copy it out.
			b.WriteByte(c)
			i++
			continue
		}

		if !isLetter(c) {
			b.WriteByte(c)
			i++
			continue
		}

		// A maximal run of the same letter.
		n := 1
		for i+n < len(pattern) && pattern[i+n] == c {
			n++
		}
		lengths, known := runs[c]
		if !known {
			b.WriteString(pattern[i : i+n])
			i += n
			continue
		}
		// Consume the run greedily, longest known length first, so an
		// unrecognised length such as "MMMMM" still produces something rather
		// than passing the whole run through.
		for n > 0 {
			used := 0
			for length := n; length >= 1; length-- {
				if out, ok := lengths[length]; ok {
					b.WriteString(out)
					used = length
					break
				}
			}
			if used == 0 {
				b.WriteString(strings.Repeat(string(c), n))
				break
			}
			i += used
			n -= used
		}
	}
	return b.String()
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
