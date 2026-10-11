package csvxl

import "testing"

// The guard leaves a number written in decimal digits alone and guards text a
// spreadsheet would not read as one, deciding by spelling rather than by what
// strconv.ParseFloat accepts.
func TestGuardFormulaDecidesByTheNumbersSpelling(t *testing.T) {
	cases := map[string]string{
		"-12":       "-12",
		"+3.5":      "+3.5",
		"-1e309":    "-1e309",
		"-.5":       "-.5",
		"+Inf":      "'+Inf",
		"-Infinity": "'-Infinity",
		"-NaN":      "'-NaN",
		"+0x1p3":    "'+0x1p3",
		"=SUM(A1)":  "'=SUM(A1)",
		"@cmd":      "'@cmd",
		"plain":     "plain",
	}
	for in, want := range cases {
		if got := guardFormula(in); got != want {
			t.Errorf("guardFormula(%q) = %q, want %q", in, got, want)
		}
	}
}
