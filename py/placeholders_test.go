package py

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// The adversarial review of py-typed-run found each placeholder replaced with
// strings.ReplaceAll over the whole text in turn, so a later pass rewrote what
// an earlier one had inserted and one argument could turn another into code.
func TestAValueIsNeverSearchedForPlaceholders(t *testing.T) {
	got, err := replacePlaceholders("title = $v1\nlabel = $v2", "$v2", "+__import__('os').system('id')+")
	if err != nil {
		t.Fatal(err)
	}
	want := "title = \"$v2\"\nlabel = \"+__import__('os').system('id')+\""
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// $v1 also matched the start of $v10.
func TestAPlaceholderIsReadByItsWholeNumber(t *testing.T) {
	args := []any{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}
	got, err := replacePlaceholders("x = $v10 + $v1 + $v11 + $v1", args...)
	if err != nil {
		t.Fatal(err)
	}
	if want := `x = "j" + "a" + "k" + "a"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPlaceholdersPastTheArgumentsAreLeftAsWritten(t *testing.T) {
	template := "x = $v2 + $v12 + $v01 + $v0 + $v99999999999999999999"
	got, err := replacePlaceholders(template, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got != template {
		t.Errorf("got %q, want %q", got, template)
	}
}

// A value JSON cannot write was written with fmt's %v, so the text inside it
// went into the script as code.
func TestAValueThatIsNotAPythonLiteralIsAnError(t *testing.T) {
	_, err := replacePlaceholders("x = $v1", []any{"__import__('os').system('id'),", math.NaN()})
	if err == nil || !strings.Contains(err.Error(), "$v1") {
		t.Errorf("got %v, want an error naming $v1", err)
	}
	if _, err := replacePlaceholders("x = $v1", math.Inf(1)); err == nil {
		t.Error("an infinity gave no error")
	}
}

// Only the arguments the template uses are converted, so one it does not use
// cannot fail the call.
func TestAnArgumentTheTemplateDoesNotUseIsNotConverted(t *testing.T) {
	got, err := replacePlaceholders("x = $v2", math.NaN(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if got != "x = 3" {
		t.Errorf("got %q, want x = 3", got)
	}
}

// Every kind of value is written as it was before the one-pass replacement.
func TestEachKindOfValueBecomesTheSamePythonLiteral(t *testing.T) {
	named := insyra.NewDataList(1, nil, "x", true)
	named.SetName(`s "q"`)
	dt := insyra.NewDataTable(insyra.NewDataList(1, 2), insyra.NewDataList("a", nil))
	dt.SetColNames([]string{"c1", "c2"})
	dt.SetRowNames([]string{"r1", "r2"})
	cases := []struct {
		value any
		want  string
	}{
		{"say \"hi\"\n\\", `"say \"hi\"\n\\"`},
		{true, "True"},
		{false, "False"},
		{[]int{1, -2}, "[1, -2]"},
		{[]float64{1.5, -2, 1e21}, "[1.5, -2, 1000000000000000000000]"},
		{[]string{`a"b`, "c"}, `["a\"b", "c"]`},
		{named, `pd.Series(name="s \"q\"",data=[1,None,"x",True])`},
		{insyra.NewDataList(), "pd.Series(data=[])"},
		{dt, `pd.DataFrame(columns=["c1","c2"],index=["r1","r2"],data=[[1,"a"],[2,None]])`},
		{map[string]any{"k": []any{nil, true, "null"}}, `{"k":[None,True,"null"]}`},
		{42, "42"},
		{2.5, "2.5"},
		{nil, "None"},
	}
	for _, c := range cases {
		got, err := replacePlaceholders("$v1", c.value)
		if err != nil {
			t.Errorf("%T: %v", c.value, err)
			continue
		}
		if got != c.want {
			t.Errorf("%T: got %s, want %s", c.value, got, c.want)
		}
	}
}

// A NaN or an infinity in a []float64 was written as NaN or +Inf, names
// Python does not know, so the script failed only once it ran.
func TestANonFiniteFloatInASliceIsAnError(t *testing.T) {
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err := replacePlaceholders("x = $v1", []float64{1, v})
		if err == nil || !strings.Contains(err.Error(), "$v1") {
			t.Errorf("[]float64{1, %v} gave %v, want an error naming $v1", v, err)
		}
	}
}
