package datafetch

import (
	"testing"
	"time"
)

func TestYFinanceTimeoutSeconds(t *testing.T) {
	cases := []struct {
		name string
		in   time.Duration
		want int
	}{
		{"explicit 30s", 30 * time.Second, 30},
		{"fractional 1500ms -> 1s", 1500 * time.Millisecond, 1},
		{"default", 0, int(defaultYFTimeout / time.Second)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := YFinanceConfig{Timeout: tc.in}
			y, err := YFinance(cfg)
			if err != nil {
				t.Fatalf("YFinance returned error: %v", err)
			}
			defer y.Close()

			if y.timeoutSeconds != tc.want {
				t.Fatalf("timeoutSeconds = %d; want %d", y.timeoutSeconds, tc.want)
			}
		})
	}
}
