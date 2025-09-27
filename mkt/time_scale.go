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
func calculateTimeDifference(laterTime, prevTime time.Time, timeScale TimeScale) int64 {
	// 將時間截斷到對應的時間尺度單位
	laterTruncated := truncateTime(laterTime, timeScale)
	prevTruncated := truncateTime(prevTime, timeScale)

	// 計算差異
	switch timeScale {
	case TimeScaleHourly:
		return int64(laterTruncated.Sub(prevTruncated).Hours())
	case TimeScaleDaily:
		return int64(laterTruncated.Sub(prevTruncated).Hours() / 24)
	case TimeScaleWeekly:
		return int64(laterTruncated.Sub(prevTruncated).Hours() / (24 * 7))
	case TimeScaleMonthly:
		// 計算月數差異
		years := laterTruncated.Year() - prevTruncated.Year()
		months := int(laterTruncated.Month()) - int(prevTruncated.Month())
		return int64(years*12 + months)
	case TimeScaleYearly:
		return int64(laterTruncated.Year() - prevTruncated.Year())
	default:
		// 默認使用天數
		return int64(laterTruncated.Sub(prevTruncated).Hours() / 24)
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
