package internal

import (
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/go-echarts/go-echarts/v2/charts"
)

// The package had no test. Its two decisions that a caller cannot see from the
// outside are the colour palette's wrap-around and ApplyYAxis's category
// detection — every call site in package plot throws the return values away,
// so they can only be checked here.

func TestDefaultColors(t *testing.T) {
	colors := DefaultColors()

	if len(colors) != 64 {
		t.Errorf("the palette has %d colours, want 64", len(colors))
	}
	seen := make(map[string]bool, len(colors))
	for i, c := range colors {
		if !strings.HasPrefix(c, "#") || len(c) != 7 {
			t.Errorf("colour %d is %q, want a #rrggbb value", i, c)
		}
		if seen[c] {
			t.Errorf("colour %q appears twice", c)
		}
		seen[c] = true
	}

	// The caller gets its own slice: writing to it must not change the palette.
	colors[0] = "#000000"
	if DefaultColors()[0] == "#000000" {
		t.Error("DefaultColors hands out the same slice every time")
	}
}

// A chart with more series than the palette has colours wraps around rather
// than running off the end.
func TestGetColor_Wraps(t *testing.T) {
	palette := DefaultColors()
	n := len(palette)

	if got, want := GetColor(0), palette[0]; got != want {
		t.Errorf("GetColor(0) = %q, want %q", got, want)
	}
	if got, want := GetColor(n), palette[0]; got != want {
		t.Errorf("GetColor(%d) = %q, want the first colour %q", n, got, want)
	}
	if got, want := GetColor(n+3), palette[3]; got != want {
		t.Errorf("GetColor(%d) = %q, want %q", n+3, got, want)
	}
}

func TestGetColors(t *testing.T) {
	palette := DefaultColors()
	n := len(palette)

	for _, count := range []int{0, -1, -100} {
		if got := GetColors(count); len(got) != 0 {
			t.Errorf("GetColors(%d) returned %d colours, want none", count, len(got))
		}
	}

	got := GetColors(3)
	if len(got) != 3 {
		t.Fatalf("GetColors(3) returned %d colours", len(got))
	}
	for i := range got {
		if got[i] != palette[i] {
			t.Errorf("colour %d: got %q, want %q", i, got[i], palette[i])
		}
	}

	// More than the palette holds: it cycles.
	got = GetColors(n + 2)
	if len(got) != n+2 {
		t.Fatalf("GetColors(%d) returned %d colours", n+2, len(got))
	}
	if got[n] != palette[0] || got[n+1] != palette[1] {
		t.Errorf("the palette did not cycle: got %q and %q", got[n], got[n+1])
	}
}

func TestApplyYAxis_NumericDataIsAValueAxis(t *testing.T) {
	chart := charts.NewLine()
	data := insyra.NewDataList(1, 2, 3).SetName("s")

	isCategory, mapping, toLineData, toFloats := ApplyYAxis(chart, "y", nil, nil, nil, nil, "", data)

	if isCategory {
		t.Error("a numeric column was read as a category axis")
	}
	if len(mapping) != 0 {
		t.Errorf("a value axis produced a label mapping: %v", mapping)
	}
	if got := toFloats(data); len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Errorf("the float converter gave %v", got)
	}
	if got := toLineData(data); len(got) != 3 {
		t.Errorf("the line-data converter gave %d points, want 3", len(got))
	}
}

func TestApplyYAxis_TextDataIsACategoryAxis(t *testing.T) {
	chart := charts.NewLine()
	data := insyra.NewDataList("low", "high", "low").SetName("s")

	isCategory, mapping, _, toFloats := ApplyYAxis(chart, "y", nil, nil, nil, nil, "", data)

	if !isCategory {
		t.Fatal("a text column was read as a value axis")
	}
	// Labels are derived in first-seen order, and each maps to its index.
	if len(mapping) != 2 {
		t.Fatalf("mapping: got %v, want two entries", mapping)
	}
	if mapping["low"] != 0 || mapping["high"] != 1 {
		t.Errorf("mapping: got %v, want low at 0 and high at 1", mapping)
	}
	got := toFloats(data)
	if len(got) != 3 || got[0] != 0 || got[1] != 1 || got[2] != 0 {
		t.Errorf("the float converter gave %v, want the label indices", got)
	}
}

// A caller that passes a labels pointer gets the derived labels written back,
// which is how the chart's axis ends up with the right tick names.
func TestApplyYAxis_WritesDerivedLabelsBack(t *testing.T) {
	chart := charts.NewLine()
	data := insyra.NewDataList("a", "b").SetName("s")
	var labels []string

	ApplyYAxis(chart, "y", &labels, nil, nil, nil, "", data)

	if len(labels) != 2 || labels[0] != "a" || labels[1] != "b" {
		t.Errorf("labels: got %v, want [a b]", labels)
	}
}

// Labels supplied by the caller are used as they are, and make the axis a
// category axis even when the data is numeric.
func TestApplyYAxis_SuppliedLabelsWin(t *testing.T) {
	chart := charts.NewLine()
	data := insyra.NewDataList(1, 2).SetName("s")
	labels := []string{"one", "two"}

	isCategory, mapping, _, _ := ApplyYAxis(chart, "y", &labels, nil, nil, nil, "", data)

	if !isCategory {
		t.Error("supplied labels did not make the axis a category axis")
	}
	if mapping["one"] != 0 || mapping["two"] != 1 {
		t.Errorf("mapping: got %v", mapping)
	}
}

func TestGenerateHTMLColorPreview(t *testing.T) {
	html := GenerateHTMLColorPreview()

	if !strings.Contains(html, "<html") {
		t.Error("the preview is not an HTML document")
	}
	for _, c := range DefaultColors() {
		if !strings.Contains(html, c) {
			t.Errorf("the preview is missing colour %q", c)
			break
		}
	}
}
