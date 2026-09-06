package mkt

import (
	"sort"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestRFMDoesNotPanicOnTextAmount(t *testing.T) {
	insyra.SetDefaultConfig()
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	dt := insyra.NewDataTable(
		insyra.NewDataList("c1", "c1", "c2").SetName("id"),
		insyra.NewDataList("2024-01-01", "2024-02-01", "2024-03-01").SetName("day"),
		insyra.NewDataList(10, "abc", 30).SetName("amt"),
	)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RFM panicked: %v", r)
		}
	}()
	out := RFM(dt, RFMConfig{CustomerIDColName: "id", TradingDayColName: "day", AmountColName: "amt", NumGroups: 2})
	if out == nil || out.NumRows() != 2 {
		t.Fatalf("expected 2 customers, got %v", out)
	}
}

func TestRFMAndCAIOutputSorted(t *testing.T) {
	insyra.SetDefaultConfig()
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	ids := []any{}
	days := []any{}
	amts := []any{}
	for _, c := range []string{"zeta", "alpha", "mid", "beta", "omega", "gamma"} {
		for i, d := range []string{"2024-01-01", "2024-01-05", "2024-01-09", "2024-01-20"} {
			ids = append(ids, c)
			days = append(days, d)
			amts = append(amts, 10+i)
		}
	}
	dt := insyra.NewDataTable(insyra.NewDataList(ids...).SetName("id"), insyra.NewDataList(days...).SetName("day"), insyra.NewDataList(amts...).SetName("amt"))
	for run := 0; run < 2; run++ {
		rfm := RFM(dt, RFMConfig{CustomerIDColName: "id", TradingDayColName: "day", AmountColName: "amt", NumGroups: 3})
		got := rfm.GetColByName("CustomerID").ToStringSlice()
		if !sort.StringsAreSorted(got) {
			t.Fatalf("RFM rows not sorted: %v", got)
		}
		cai := CustomerActivityIndex(dt, CAIConfig{CustomerIDColName: "id", TradingDayColName: "day"})
		gotC := cai.GetColByName("CustomerID").ToStringSlice()
		if !sort.StringsAreSorted(gotC) {
			t.Fatalf("CAI rows not sorted: %v", gotC)
		}
	}
}
