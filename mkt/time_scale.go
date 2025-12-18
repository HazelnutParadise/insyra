package mkt

import "time"

type TimeScale string

const (
	TimeScaleHourly  TimeScale = "hourly"
	TimeScaleDaily   TimeScale = "daily"
	TimeScaleWeekly  TimeScale = "weekly"
	TimeScaleMonthly TimeScale = "monthly"
	TimeScaleYearly  TimeScale = "yearly"
)

// calculateTimeDifference 根據 TimeScale 計算兩個時間點之間的差異
// 先將時間截斷到對應的時間尺度單位，再計算整數個時間尺度單位的差異
// 這與 formatTradingPeriod 的邏輯相呼應，確保同一單位時間尺度內只算一次
func calculateTimeDifference(laterTime, prevTime time.Time, timeScale TimeScale) int64 {
	// 將時間截斷到對應的時間尺度單位
	laterTruncated := truncateTime(laterTime, timeScale)
	prevTruncated := truncateTime(prevTime, timeScale)

	// 計算差異（確保是整數個時間尺度單位）
	switch timeScale {
	case TimeScaleHourly:
		// 計算小時數差異：天數差 * 24 + 小時差，避免夏令時問題
		daysDiff := dateDiffInDays(laterTruncated, prevTruncated)
		hoursDiff := laterTruncated.Hour() - prevTruncated.Hour()
		return int64(daysDiff*24 + hoursDiff)
	case TimeScaleDaily:
		// 計算天數差異：使用日期組件直接計算
		return int64(dateDiffInDays(laterTruncated, prevTruncated))
	case TimeScaleWeekly:
		// 計算週數差異：天數差除以7（因為已經截斷到週一，所以一定是7的倍數）
		return int64(dateDiffInDays(laterTruncated, prevTruncated) / 7)
	case TimeScaleMonthly:
		// 計算月數差異
		years := laterTruncated.Year() - prevTruncated.Year()
		months := int(laterTruncated.Month()) - int(prevTruncated.Month())
		return int64(years*12 + months)
	case TimeScaleYearly:
		return int64(laterTruncated.Year() - prevTruncated.Year())
	default:
		// 默認使用天數
		return int64(dateDiffInDays(laterTruncated, prevTruncated))
	}
}

// dateDiffInDays 計算兩個時間之間的天數差異（整數）
// 使用 Julian Day Number 的概念來精確計算天數差
func dateDiffInDays(later, prev time.Time) int {
	// 將兩個時間都轉換為同一時區（UTC）的日期，然後計算天數差
	laterDate := time.Date(later.Year(), later.Month(), later.Day(), 0, 0, 0, 0, time.UTC)
	prevDate := time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, time.UTC)
	// 使用 Sub 計算 Duration，然後轉換為天數（UTC 下不會有夏令時問題）
	return int(laterDate.Sub(prevDate).Hours() / 24)
}

// truncateTime 將時間截斷到指定的時間尺度單位
func truncateTime(t time.Time, timeScale TimeScale) time.Time {
	switch timeScale {
	case TimeScaleHourly:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	case TimeScaleDaily:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case TimeScaleWeekly:
		// 截斷到本週的週一
		daysSinceMonday := int(t.Weekday() - time.Monday)
		if daysSinceMonday < 0 {
			daysSinceMonday += 7
		}
		return time.Date(t.Year(), t.Month(), t.Day()-daysSinceMonday, 0, 0, 0, 0, t.Location())
	case TimeScaleMonthly:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case TimeScaleYearly:
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	default:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}
}
