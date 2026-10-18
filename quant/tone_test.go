package quant

import (
	"math"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

func counts(t *testing.T, dt *insyra.DataTable, name string) []int64 {
	t.Helper()
	var out []int64
	for _, v := range dt.GetColByName(name).Data() {
		out = append(out, v.(int64))
	}
	return out
}

func TestToneCountsWords(t *testing.T) {
	lex := Lexicon{Positive: []string{"good", "gain"}, Negative: []string{"loss", "weak"}, Negation: []string{"not", "no"}}
	texts := insyra.NewDataList("Not good, but gain; weak loss", "GOOD gain", nil, "no gain")
	got, err := ToneCounts(texts, lex, ToneOptions{NegationWindow: 1})
	if err != nil {
		t.Fatal(err)
	}
	// Text 1: "not good" is negated, "but gain" is not (window 1); weak and loss are negative; 6 words.
	if p, n, w := counts(t, got, "Positive"), counts(t, got, "Negative"), counts(t, got, "Words"); p[0] != 1 || n[0] != 2 || w[0] != 6 ||
		p[1] != 2 || w[1] != 2 || p[2] != 0 || w[2] != 0 || p[3] != 0 || w[3] != 2 {
		t.Errorf("positive %v negative %v words %v", p, n, w)
	}
	// With a window of 3 the negation reaches back to "gain" as well.
	wide, _ := ToneCounts(texts, lex, ToneOptions{NegationWindow: 3})
	if p := counts(t, wide, "Positive"); p[0] != 0 {
		t.Errorf("window 3: positive %v", p)
	}
	// Without negation handling every positive word counts.
	plain, _ := ToneCounts(texts, lex)
	if p := counts(t, plain, "Positive"); p[0] != 2 || p[3] != 1 {
		t.Errorf("no negation: positive %v", p)
	}
	exact, _ := ToneCounts(insyra.NewDataList("GOOD good"), lex, ToneOptions{CaseSensitive: true})
	if p := counts(t, exact, "Positive"); p[0] != 1 {
		t.Errorf("case sensitive: positive %v", p)
	}
}

func TestToneCountsSubstringConsumesLongestMatch(t *testing.T) {
	lex := Lexicon{Positive: []string{"大漲", "漲", "好"}, Negative: []string{"跌"}, Negation: []string{"不"}}
	got, err := ToneCounts(insyra.NewDataList("大漲後不好又漲，一跌"), lex, ToneOptions{Mode: ToneSubstring, NegationWindow: 1})
	if err != nil {
		t.Fatal(err)
	}
	// 大漲 is one positive term (not 大漲 and 漲), 不好 is negated (nothing between), 又漲 is two characters
	// after the negation and so outside a window of 1, and 跌 is negative.
	p, n, w := counts(t, got, "Positive"), counts(t, got, "Negative"), counts(t, got, "Words")
	if p[0] != 2 || n[0] != 1 || w[0] != 9 {
		t.Errorf("positive %v negative %v words %v", p, n, w)
	}
}

func TestToneCountsRefusesBadInput(t *testing.T) {
	var nilList *insyra.DataList
	cases := map[string]func() error{
		"nil":    func() error { _, e := ToneCounts(nilList, Lexicon{}); return e },
		"number": func() error { _, e := ToneCounts(insyra.NewDataList("a", 3), Lexicon{}); return e },
		"both": func() error {
			_, e := ToneCounts(insyra.NewDataList("a"), Lexicon{Positive: []string{"a"}, Negative: []string{"A"}})
			return e
		},
		"empty": func() error { _, e := ToneCounts(insyra.NewDataList("a"), Lexicon{Positive: []string{" "}}); return e },
		"window": func() error {
			_, e := ToneCounts(insyra.NewDataList("a"), Lexicon{}, ToneOptions{NegationWindow: -1})
			return e
		},
		"mode": func() error { _, e := ToneCounts(insyra.NewDataList("a"), Lexicon{}, ToneOptions{Mode: 9}); return e },
		"two opts": func() error {
			_, e := ToneCounts(insyra.NewDataList("a"), Lexicon{}, ToneOptions{}, ToneOptions{})
			return e
		},
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestDailyToneAlignsWeekendAndAfterCloseNewsToTheNextSession(t *testing.T) {
	taipei := time.FixedZone("CST", 8*3600)
	at := func(d, h, m int) time.Time { return time.Date(2026, time.October, d, h, m, 0, 0, taipei) }
	trading := insyra.NewDataList(day(2026, 10, 12), day(2026, 10, 13), day(2026, 10, 14), day(2026, 10, 15), day(2026, 10, 16), day(2026, 10, 19))
	published := insyra.NewDataList(
		at(12, 9, 0),   // Monday morning: Monday
		at(13, 13, 29), // just before the close: Tuesday
		at(13, 13, 30), // at the close: Wednesday
		at(16, 10, 0),  // Friday morning: Friday
		at(16, 18, 0),  // Friday evening: Monday
		at(17, 11, 0),  // Saturday: Monday
		at(20, 9, 0),   // after the last trading day: dropped
	)
	counts := insyra.NewDataTable(
		insyra.NewDataList(2, 1, 1, 3, 1, 1, 9).SetName("Positive"),
		insyra.NewDataList(0, 1, 0, 1, 0, 1, 9).SetName("Negative"),
		insyra.NewDataList(10, 10, 10, 10, 10, 10, 10).SetName("Words"),
	)
	got, err := DailyTone(published, counts, trading, DailyToneOptions{CloseHour: 13, CloseMinute: 30, Window: 2})
	if err != nil {
		t.Fatal(err)
	}
	pos, words := counts2(t, got, "Positive"), counts2(t, got, "Words")
	wantPos := []int64{2, 1, 1, 0, 3, 2}
	wantWords := []int64{10, 10, 10, 0, 10, 20}
	for i := range wantPos {
		if pos[i] != wantPos[i] || words[i] != wantWords[i] {
			t.Fatalf("positive %v words %v, want %v and %v", pos, words, wantPos, wantWords)
		}
	}
	// Window 2: day 1 = Mon+Tue = (2+1-0-1)/20; day 3 (no news) = Tue? no: Wed+Thu = (1)/10.
	net := got.GetColByName("NetTone").Data()
	if math.Abs(net[1].(float64)-2.0/20) > 1e-12 || math.Abs(net[3].(float64)-1.0/10) > 1e-12 {
		t.Errorf("net tone = %v", net)
	}
	// Monday 19th: Fri (3-1)/10 and Mon (2-1)/20 over 30 words.
	if math.Abs(net[5].(float64)-3.0/30) > 1e-12 && math.Abs(net[5].(float64)-(2.0+1.0)/30) > 1e-12 {
		t.Errorf("net tone Monday = %v", net[5])
	}
}

func counts2(t *testing.T, dt *insyra.DataTable, name string) []int64 {
	t.Helper()
	return counts(t, dt, name)
}

func TestDailyToneZScoreUsesOnlyEarlierDays(t *testing.T) {
	days := make([]any, 8)
	pub := make([]any, 8)
	pos := make([]any, 8)
	neg := make([]any, 8)
	words := make([]any, 8)
	for i := range days {
		days[i] = day(2026, 10, 12+i)
		pub[i] = day(2026, 10, 12+i).Add(9 * time.Hour)
		pos[i], neg[i], words[i] = i+1, 0, 10
	}
	run := func(lastPositive int) *insyra.DataTable {
		p := append([]any(nil), pos...)
		p[7] = lastPositive
		dt, err := DailyTone(insyra.NewDataList(pub...), insyra.NewDataTable(
			insyra.NewDataList(p...).SetName("Positive"), insyra.NewDataList(neg...).SetName("Negative"), insyra.NewDataList(words...).SetName("Words")),
			insyra.NewDataList(days...), DailyToneOptions{Window: 1, ZWindow: 4})
		if err != nil {
			t.Fatal(err)
		}
		return dt
	}
	a, b := run(8), run(80)
	za, zb := a.GetColByName("Z").Data(), b.GetColByName("Z").Data()
	for i := 0; i < 7; i++ {
		if !(za[i] == zb[i] || (math.IsNaN(za[i].(float64)) && math.IsNaN(zb[i].(float64)))) {
			t.Errorf("row index %d changed when only the last day's news did", i)
		}
	}
	// Day 4's z: net tones 0.1..0.4 earlier, today 0.5.
	earlier := []float64{0.1, 0.2, 0.3, 0.4}
	mean := 0.25
	ss := 0.0
	for _, v := range earlier {
		ss += (v - mean) * (v - mean)
	}
	want := (0.5 - mean) / math.Sqrt(ss/3)
	if math.Abs(za[4].(float64)-want) > 1e-12 || !math.IsNaN(za[3].(float64)) {
		t.Errorf("z at row index 4 = %v, want %v (row index 3 %v)", za[4], want, za[3])
	}
}

func TestDailyToneRefusesBadInput(t *testing.T) {
	trading := insyra.NewDataList(day(2026, 10, 12), day(2026, 10, 13))
	pub := insyra.NewDataList(day(2026, 10, 12))
	good := insyra.NewDataTable(insyra.NewDataList(1).SetName("Positive"), insyra.NewDataList(0).SetName("Negative"), insyra.NewDataList(5).SetName("Words"))
	var nilList *insyra.DataList
	cases := map[string]func() error{
		"window":  func() error { _, e := DailyTone(pub, good, trading, DailyToneOptions{}); return e },
		"zwindow": func() error { _, e := DailyTone(pub, good, trading, DailyToneOptions{Window: 1, ZWindow: 1}); return e },
		"close": func() error {
			_, e := DailyTone(pub, good, trading, DailyToneOptions{Window: 1, CloseHour: 25})
			return e
		},
		"nil": func() error { _, e := DailyTone(nilList, good, trading, DailyToneOptions{Window: 1}); return e },
		"columns": func() error {
			_, e := DailyTone(pub, insyra.NewDataTable(insyra.NewDataList(1)), trading, DailyToneOptions{Window: 1})
			return e
		},
		"length": func() error {
			_, e := DailyTone(insyra.NewDataList(day(2026, 10, 12), day(2026, 10, 13)), good, trading, DailyToneOptions{Window: 1})
			return e
		},
		"not time": func() error {
			_, e := DailyTone(insyra.NewDataList("x"), good, trading, DailyToneOptions{Window: 1})
			return e
		},
		"days": func() error {
			_, e := DailyTone(pub, good, insyra.NewDataList(day(2026, 10, 13), day(2026, 10, 12)), DailyToneOptions{Window: 1})
			return e
		},
		"negative": func() error {
			_, e := DailyTone(pub, insyra.NewDataTable(insyra.NewDataList(-1).SetName("Positive"), insyra.NewDataList(0).SetName("Negative"), insyra.NewDataList(5).SetName("Words")), trading, DailyToneOptions{Window: 1})
			return e
		},
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
