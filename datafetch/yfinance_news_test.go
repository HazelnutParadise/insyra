package datafetch

import (
	"strings"
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestYFNewsTabConvertsToGoYFinance(t *testing.T) {
	cases := []struct {
		name      string
		tab       YFNewsTab
		want      models.NewsTab
		wantErr   bool
		errSubstr string
	}{
		{"empty string means news", "", models.NewsTabNews, false, ""},
		{"YFNewsTabNews", YFNewsTabNews, models.NewsTabNews, false, ""},
		{"YFNewsTabAll", YFNewsTabAll, models.NewsTabAll, false, ""},
		{"YFNewsTabPressReleases", YFNewsTabPressReleases, models.NewsTabPressReleases, false, ""},
		{"unknown tab", YFNewsTab("videos"), "", true, "videos"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.tab.toModel()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.tab)
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNewsRefusesAnUnknownTabBeforeAnyRequest(t *testing.T) {
	y, err := YFinance(YFinanceConfig{})
	if err != nil {
		t.Fatal(err)
	}
	defer y.close()

	_, err = y.Ticker("AAPL").News(5, YFNewsTab("videos"))
	if err == nil {
		t.Fatal("expected error for unknown tab")
	}
	if !strings.Contains(err.Error(), "videos") {
		t.Errorf("error %q does not contain %q", err.Error(), "videos")
	}
}
