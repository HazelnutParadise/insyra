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
