package palette

import (
	"fmt"
	"testing"
)

func TestColorMatchesHexAndWraps(t *testing.T) {
	hex := Hex()
	if hex[0] != "#1F449B" || hex[1] != "#F9AC41" || hex[2] != "#D73E3E" {
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
	if Hex()[0] != "#1F449B" {
		t.Error("Hex returned the palette itself, so a caller could change it")
	}
}
