package mkt

import (
	"testing"
	"time"
)

func TestCalculateTimeDifference(t *testing.T) {
	now := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	tests := []struct {
		name           string
		lastTradingDay time.Time
		timeScale      TimeScale
		expected       int64
	}{
		{
			name:           "Hourly difference",
			lastTradingDay: time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC),
			timeScale:      TimeScaleHourly,
			expected:       2,
		},
		{
			name:           "Daily difference",
			lastTradingDay: time.Date(2024, 6, 14, 23, 59, 0, 0, time.UTC),
			timeScale:      TimeScaleDaily,
			expected:       1,
		},
		{
			name:           "Weekly difference",
			lastTradingDay: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			timeScale:      TimeScaleWeekly,
			expected:       2,
		},
		{
			name:           "Monthly difference",
			lastTradingDay: time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC),
			timeScale:      TimeScaleMonthly,
			expected:       2,
		},
		{
			name:           "Yearly difference",
			lastTradingDay: time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC),
			timeScale:      TimeScaleYearly,
			expected:       1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTimeDifference(now, tt.lastTradingDay, tt.timeScale)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestCalculateTimeDifferenceIntegerUnits 測試確保回傳整數個時間尺度單位
// 這與 formatTradingPeriod 的邏輯相呼應
func TestCalculateTimeDifferenceIntegerUnits(t *testing.T) {
	tests := []struct {
		name      string
		later     time.Time
		prev      time.Time
		timeScale TimeScale
		expected  int64
	}{
		// Hourly：同一小時內應為 0
		{
			name:      "Hourly_same_hour",
			later:     time.Date(2024, 6, 15, 10, 59, 59, 0, time.UTC),
			prev:      time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
			timeScale: TimeScaleHourly,
			expected:  0,
		},
		// Daily：同一天內應為 0
		{
			name:      "Daily_same_day",
			later:     time.Date(2024, 6, 15, 23, 59, 59, 0, time.UTC),
			prev:      time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			timeScale: TimeScaleDaily,
			expected:  0,
		},
		// Weekly：同一週內應為 0（2024-06-10 是週一，2024-06-16 是週日）
		{
			name:      "Weekly_same_week",
			later:     time.Date(2024, 6, 16, 23, 59, 59, 0, time.UTC), // 週日
			prev:      time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC),    // 週一
			timeScale: TimeScaleWeekly,
			expected:  0,
		},
		// Weekly：跨週應為 1
		{
			name:      "Weekly_cross_week",
			later:     time.Date(2024, 6, 17, 0, 0, 0, 0, time.UTC),    // 下週一
			prev:      time.Date(2024, 6, 16, 23, 59, 59, 0, time.UTC), // 週日
			timeScale: TimeScaleWeekly,
			expected:  1,
		},
		// Monthly：同一月內應為 0
		{
			name:      "Monthly_same_month",
			later:     time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC),
			prev:      time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			timeScale: TimeScaleMonthly,
			expected:  0,
		},
		// Yearly：同一年內應為 0
		{
			name:      "Yearly_same_year",
			later:     time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			prev:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			timeScale: TimeScaleYearly,
			expected:  0,
		},
		// Yearly：跨年應為 1
		{
			name:      "Yearly_cross_year",
			later:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			prev:      time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			timeScale: TimeScaleYearly,
			expected:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTimeDifference(tt.later, tt.prev, tt.timeScale)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
