package palette

import (
	"fmt"
	"testing"
)

func TestColorMatchesHexAndWraps(t *testing.T) {
	hex := Hex()
	if len(hex) != 9 || len(Names()) != len(hex) {
		t.Fatalf("%d colours and %d names; want 9 of each", len(hex), len(Names()))
	}
	if hex[0] != "#2550B0" || hex[1] != "#EE9A2C" || hex[2] != "#D73E3E" {
		t.Fatalf("the palette opens with %v; want Insyra's blue, orange and red", hex[:3])
	}
	for _, i := range []int{0, 1, 2, len(hex) - 1, len(hex), len(hex) + 1} {
		r, g, b, _ := Color(i).RGBA()
		got := fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
		if want := hex[i%len(hex)]; got != want {
			t.Errorf("Color(%d) = %s; want %s", i, got, want)
		}
	}
	hex[0] = "#000000"
	if Hex()[0] != "#2550B0" {
		t.Error("Hex returned the palette itself, so a caller could change it")
	}
}
