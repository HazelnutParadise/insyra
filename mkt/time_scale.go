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
// 先將時間截斷到對應的時間尺度單位，再計算差異
func calculateTimeDifference(now, prevTime time.Time, timeScale TimeScale) int {
	// 將時間截斷到對應的時間尺度單位
	nowTruncated := truncateTime(now, timeScale)
	prevTruncated := truncateTime(prevTime, timeScale)

	// 計算差異
	switch timeScale {
	case TimeScaleHourly:
		return int(nowTruncated.Sub(prevTruncated).Hours())
	case TimeScaleDaily:
		return int(nowTruncated.Sub(prevTruncated).Hours() / 24)
	case TimeScaleWeekly:
		return int(nowTruncated.Sub(prevTruncated).Hours() / (24 * 7))
	case TimeScaleMonthly:
		// 計算月數差異
		years := nowTruncated.Year() - prevTruncated.Year()
		months := int(nowTruncated.Month()) - int(prevTruncated.Month())
		return years*12 + months
	case TimeScaleYearly:
		return nowTruncated.Year() - prevTruncated.Year()
	default:
		// 默認使用天數
		return int(nowTruncated.Sub(prevTruncated).Hours() / 24)
	}
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
