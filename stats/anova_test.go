package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestOneWayANOVA(t *testing.T) {
	group1 := insyra.NewDataList([]float64{10, 12, 9, 11})
	group2 := insyra.NewDataList([]float64{20, 19, 21, 22})
	group3 := insyra.NewDataList([]float64{30, 29, 28, 32})

	result := stats.OneWayANOVA(group1, group2, group3)
	if result == nil {
		t.Fatal("OneWayANOVA returned nil")
	}

	expectF := 177.96
	expectP := 5.81e-08

	if !floatAlmostEqual(result.Factor.F, expectF, 0.01) {
		t.Errorf("Unexpected F value: got %.4f, want %.4f", result.Factor.F, expectF)
	}
	if !floatAlmostEqual(result.Factor.P, expectP, 1e-8) {
		t.Errorf("Unexpected P value: got %.10f, want %.10f", result.Factor.P, expectP)
	}
}

func TestTwoWayANOVA(t *testing.T) {
	A1B1 := insyra.NewDataList([]float64{5, 6, 5})
	A1B2 := insyra.NewDataList([]float64{7, 8, 9})
	A2B1 := insyra.NewDataList([]float64{4, 3, 4})
	A2B2 := insyra.NewDataList([]float64{10, 11, 9})

	result := stats.TwoWayANOVA(2, 2, A1B1, A1B2, A2B1, A2B2)
	if result == nil {
		t.Fatal("TwoWayANOVA returned nil")
	}

	expectF_A := 0.125
	expectP_A := 0.7328

	if !floatAlmostEqual(result.FactorA.F, expectF_A, 0.001) {
		t.Errorf("Unexpected F(A): got %.4f, want %.4f", result.FactorA.F, expectF_A)
	}
	if !floatAlmostEqual(result.FactorA.P, expectP_A, 0.001) {
		t.Errorf("Unexpected P(A): got %.4f, want %.4f", result.FactorA.P, expectP_A)
	}
}

func TestRepeatedMeasuresANOVA(t *testing.T) {
	s1 := insyra.NewDataList([]float64{10, 15, 14})
	s2 := insyra.NewDataList([]float64{12, 14, 13})
	s3 := insyra.NewDataList([]float64{11, 13, 13})
	s4 := insyra.NewDataList([]float64{13, 15, 14})
	s5 := insyra.NewDataList([]float64{12, 13, 15})

	result := stats.RepeatedMeasuresANOVA(s1, s2, s3, s4, s5)
	if result == nil {
		t.Fatal("RepeatedMeasuresANOVA returned nil")
	}

	expectF := 9.33
	expectP := 0.0081

	if !floatAlmostEqual(result.Factor.F, expectF, 0.1) {
		t.Errorf("Unexpected F value: got %.4f, want %.4f", result.Factor.F, expectF)
	}
	if !floatAlmostEqual(result.Factor.P, expectP, 0.001) {
		t.Errorf("Unexpected P value: got %.6f, want %.6f", result.Factor.P, expectP)
	}
}
