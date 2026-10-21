package ta

import (
	"math"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestPivotPointsByHand(t *testing.T) {
	// Row 0: H 12, L 8, C 10, so row 1 sees P 10, R1 12, R2 14, R3 16, S1 8, S2 6, S3 4.
	got, err := PivotPoints(prices(12, 13), prices(8, 9), prices(10, 11))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{"P": 10, "R1": 12, "R2": 14, "R3": 16, "S1": 8, "S2": 6, "S3": 4}
	for name, l := range map[string][]float64{
		"P": floats(t, got.P), "R1": floats(t, got.R1), "R2": floats(t, got.R2), "R3": floats(t, got.R3),
		"S1": floats(t, got.S1), "S2": floats(t, got.S2), "S3": floats(t, got.S3),
	} {
		if !math.IsNaN(l[0]) || math.Abs(l[1]-want[name]) > 1e-12 {
			t.Errorf("%s = %v, want NaN then %v", name, l, want[name])
		}
	}
}

func TestHeikinAshiByHand(t *testing.T) {
	dt, err := HeikinAshi(prices(10, 11), prices(12, 13), prices(9, 10), prices(11, 12))
	if err != nil {
		t.Fatal(err)
	}
	col := func(name string) []float64 { return floats(t, dt.GetColByName(name)) }
	o, h, l, c := col("Open"), col("High"), col("Low"), col("Close")
	// Row 0: close (10+12+9+11)/4 = 10.5, open (10+11)/2 = 10.5. Row 1: open (10.5+10.5)/2, close (11+13+10+12)/4 = 11.5.
	if o[0] != 10.5 || c[0] != 10.5 || h[0] != 12 || l[0] != 9 || o[1] != 10.5 || c[1] != 11.5 || h[1] != 13 || l[1] != 10 {
		t.Fatalf("o %v h %v l %v c %v", o, h, l, c)
	}
}

func TestHeikinAshiValidatesAllFourSeries(t *testing.T) {
	p := prices(10, 11, 12)
	for name, args := range map[string][4]insyra.IDataList{
		"zero open":      {prices(10, 0, 12), p, p, p},
		"zero high":      {p, prices(10, 0, 12), p, p},
		"negative low":   {p, p, prices(10, -1, 12), p},
		"zero close":     {p, p, p, prices(10, 0, 12)},
		"high below low": {p, prices(9, 9, 9), p, p},
		"nan close":      {p, p, p, prices(10, math.NaN(), 12)},
		"short high":     {p, prices(10, 11), p, p},
		"short low":      {p, p, prices(10, 11), p},
		"short open":     {prices(10, 11), p, p, p},
	} {
		if _, err := HeikinAshi(args[0], args[1], args[2], args[3]); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestPivotPointsResultTypeName(t *testing.T) {
	r, err := PivotPoints(prices(1, 2), prices(1, 2), prices(1, 2))
	if err != nil || r == nil {
		t.Fatal(err)
	}
}

func TestTDSequentialCountsASetupAndItsCountdown(t *testing.T) {
	c := []float64{10, 10, 10, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	for v := 0.5; len(c) < 30; v -= 0.01 {
		c = append(c, v)
	}
	got, err := TDSequential(prices(c...))
	if err != nil {
		t.Fatal(err)
	}
	bs, bc, ss := floats(t, got.BuySetup), floats(t, got.BuyCountdown), floats(t, got.SellSetup)
	for i, want := range []float64{0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9} {
		if bs[i] != want {
			t.Fatalf("buy setup %v", bs[:13])
		}
	}
	if bs[13] != 1 {
		t.Errorf("a run that goes on past 9 starts again from 1: %v", bs[13])
	}
	if bc[12] != 0 || bc[13] != 1 || bc[14] != 2 || bc[25] != 13 || bc[26] != 13 {
		t.Fatalf("buy countdown %v", bc[10:])
	}
	for _, v := range ss {
		if v != 0 {
			t.Fatalf("no sell setup on a fall: %v", ss)
		}
	}
	// A completed sell setup cancels an unfinished buy countdown.
	d := append(append([]float64(nil), c[:20]...), 5, 5, 5, 5)
	for v := 6.0; len(d) < 50; v++ {
		d = append(d, v)
	}
	res, _ := TDSequential(prices(d...))
	cd := floats(t, res.BuyCountdown)
	if cd[len(cd)-1] != 0 {
		t.Errorf("a sell setup completed, so the buy countdown is cancelled: %v", cd[len(cd)-1])
	}
}

func TestPointAndFigureByHand(t *testing.T) {
	got, err := PointAndFigure(prices(10.5, 11.2, 12.4, 13.1, 12.9, 11.5, 10.2), 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []PFColumn{
		{PFRising, 1, 3, 13, 10, 4},
		{PFFalling, 6, 6, 12, 10, 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
	// Moves inside a box change nothing.
	none, _ := PointAndFigure(prices(10.1, 10.9, 10.4), 1, 3)
	if len(none) != 0 {
		t.Errorf("no box was left: %+v", none)
	}
	// A falling start and a reversal back up.
	down, _ := PointAndFigure(prices(20.5, 19.5, 18.2, 17.5, 20.1), 1, 3)
	if len(down) != 2 || down[0].Kind != PFFalling || down[0].Bottom != 17 || down[1].Kind != PFRising || down[1].Boxes != 3 {
		t.Errorf("down then up: %+v", down)
	}
}

func TestStructureRefusesBadInput(t *testing.T) {
	p := prices(10, 11, 12)
	if _, err := PivotPoints(p, prices(1, 2), p); err == nil {
		t.Error("different lengths")
	}
	if _, err := HeikinAshi(prices(10, 0, 12), p, p, p); err == nil {
		t.Error("a zero open")
	}
	if _, err := HeikinAshi(p, p, p, prices(1)); err == nil {
		t.Error("different lengths")
	}
	if _, err := TDSequential(nil); err == nil {
		t.Error("nil close")
	}
	if _, err := PointAndFigure(p, 0, 3); err == nil {
		t.Error("box size 0")
	}
	if _, err := PointAndFigure(p, 1, 0); err == nil {
		t.Error("reversal 0")
	}
	if _, err := PointAndFigure(prices(1, -1), 1, 3); err == nil {
		t.Error("a negative close")
	}
	if got, err := PointAndFigure(prices(), 1, 3); err != nil || got != nil {
		t.Errorf("empty input: %v, %v", got, err)
	}
}

func TestStructureListsAreNamed(t *testing.T) {
	p := prices(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	pv, _ := PivotPoints(p, p, p)
	td, _ := TDSequential(p)
	for want, got := range map[string]string{
		"P": pv.P.GetName(), "R1": pv.R1.GetName(), "R2": pv.R2.GetName(), "R3": pv.R3.GetName(),
		"S1": pv.S1.GetName(), "S2": pv.S2.GetName(), "S3": pv.S3.GetName(),
		"BuySetup": td.BuySetup.GetName(), "SellSetup": td.SellSetup.GetName(),
		"BuyCountdown": td.BuyCountdown.GetName(), "SellCountdown": td.SellCountdown.GetName(),
	} {
		if got != want {
			t.Errorf("a list is named %q, want %q", got, want)
		}
	}
	dt, _ := HeikinAshi(p, p, p, p)
	for _, name := range []string{"Open", "High", "Low", "Close"} {
		if dt.GetColByName(name) == nil {
			t.Errorf("Heikin-Ashi has no %s column", name)
		}
	}
}

func TestTDSequentialRestartsASetupPastNine(t *testing.T) {
	c := []float64{20, 20, 20, 20}
	for v := 19.0; v >= 9; v-- { // ten falling closes after four flat ones
		c = append(c, v)
	}
	got, err := TDSequential(prices(c...))
	if err != nil {
		t.Fatal(err)
	}
	bs := floats(t, got.BuySetup)
	if bs[12] != 9 || bs[13] != 1 {
		t.Fatalf("buy setup %v", bs)
	}
}
