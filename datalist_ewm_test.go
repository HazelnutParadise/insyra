package insyra

import (
	"strings"
	"testing"
)

func TestDataList_EWM_InvalidDecayParameters(t *testing.T) {
	cases := []EWMOptions{
		{},
		{Alpha: 0.5, Span: 3},
	}
	for _, opts := range cases {
		dl := NewDataList(1.0, 2.0, 3.0)
		got := dl.EWM(opts).Mean()
		if got == nil || got.Len() != 0 {
			t.Errorf("invalid options returned %v, want empty DataList", got)
		}
		if err := dl.Err(); err == nil || !strings.Contains(err.Message, "EWM") {
			t.Errorf("invalid options Err = %v, want EWM warning", err)
		}
	}
}

func TestDataList_EWM_AdjustFalseRecursiveMean(t *testing.T) {
	got := NewDataList(1.0, 2.0, 3.0).EWM(EWMOptions{Alpha: 0.5}).Mean().Data()
	want := []any{1.0, 1.5, 2.25}
	sliceEqualApprox(t, got, want, 1e-12)

	got = NewDataList(1.0, 2.0, 3.0).EWM(EWMOptions{Alpha: 0.5, Adjust: false}).Mean().Data()
	sliceEqualApprox(t, got, want, 1e-12)
}

func TestDataList_EWM_MinObs(t *testing.T) {
	got := NewDataList(1.0, 2.0, 3.0, 4.0, 5.0).EWM(EWMOptions{Alpha: 0.5, MinObs: 3}).Mean().Data()
	want := []any{nil, nil, 2.25, 3.125, 4.0625}
	sliceEqualApprox(t, got, want, 1e-12)
}

func TestDataTable_EWMCol_ResolvesNameAndIndex(t *testing.T) {
	dt := buildPriceTable()
	want := []any{100.0, 105.0, 112.5, 113.75, 121.875}
	sliceEqualApprox(t, dt.EWMCol("price", EWMOptions{Alpha: 0.5, Adjust: false}).Mean().Data(), want, 1e-12)
	sliceEqualApprox(t, dt.EWMCol("B", EWMOptions{Alpha: 0.5, Adjust: false}).Mean().Data(), want, 1e-12)
}
