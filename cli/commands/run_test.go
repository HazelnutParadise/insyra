package commands

import "testing"

// A Windows absolute path lost every separator because `\` was treated as a
// universal escape, so `load C:\data\bars.csv` opened `C:databars.csv`.
func TestSplitScriptTokensKeepsWindowsPaths(t *testing.T) {
	cases := map[string][]string{
		`load C:\Users\me\bars.csv as dt`:   {"load", `C:\Users\me\bars.csv`, "as", "dt"},
		`load "C:\Users\me\bars.csv" as dt`: {"load", `C:\Users\me\bars.csv`, "as", "dt"},
		`load /tmp/a b.csv`:                 {"load", "/tmp/a", "b.csv"},
		`load "/tmp/a b.csv"`:               {"load", "/tmp/a b.csv"},
		// A backslash still escapes a quote and itself.
		`filter dt 'O\'Brien'`: {"filter", "dt", "O'Brien"},
		`echo a\\b`:            {"echo", `a\b`},
	}
	for line, want := range cases {
		got := splitScriptTokens(line)
		if len(got) != len(want) {
			t.Errorf("%s\n  got  %q\n  want %q", line, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s\n  got  %q\n  want %q", line, got, want)
				break
			}
		}
	}
}
