package utils

import "strings"

func ParseColIndex(colIndex string) int {
	result := 0
	for _, char := range colIndex {
		result = result*26 + int(char-'A') + 1
	}
	return result - 1
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
