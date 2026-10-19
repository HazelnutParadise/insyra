package quant

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/HazelnutParadise/insyra"
)

// ToneMode chooses how text is split into words for ToneCounts.
type ToneMode int

const (
	// ToneWords splits text into words at spaces and punctuation, as English is
	// written, and matches terms as whole words.
	ToneWords ToneMode = iota
	// ToneSubstring matches terms as substrings, longest first, for text with
	// no spaces such as Chinese. Each matched term is consumed, so with both 大漲
	// and 漲 in the lexicon, 大漲 is counted once and 漲 is not counted again
	// inside it.
	ToneSubstring
)

// String returns the constant name of m, or ToneMode(n) for a value that is not
// one of the constants.
func (m ToneMode) String() string {
	switch m {
	case ToneWords:
		return "ToneWords"
	case ToneSubstring:
		return "ToneSubstring"
	}
	return fmt.Sprintf("ToneMode(%d)", int(m))
}

// Lexicon is the word lists a tone score counts. It is not supplied: the
// Loughran-McDonald finance lists for English are available from their authors
// under their own terms, and a Chinese list has to come from the caller.
type Lexicon struct {
	Positive []string
	Negative []string
	// Negation lists the words that cancel a positive word following them.
	Negation []string
}

// ToneOptions are the optional settings of ToneCounts.
type ToneOptions struct {
	Mode ToneMode
	// NegationWindow is how far before a positive word a negation cancels it: in
	// words for ToneWords and in characters for ToneSubstring. 0 means no
	// negation handling. Only positive words are cancelled; a negated negative
	// word is still counted as negative.
	NegationWindow int
	// CaseSensitive matches the lexicon exactly. Otherwise case is ignored.
	CaseSensitive bool
}

// ToneCounts counts positive and negative words in each text. The result has
// the columns Positive and Negative (int64, words matched) and Words (int64,
// the length of the text in words for ToneWords and in letters, digits and
// characters for ToneSubstring, with spaces and punctuation left out).
// A nil or empty text counts zero. Terms that appear in more than one list make
// an error, since one word cannot be both.
func ToneCounts(texts insyra.IDataList, lex Lexicon, opts ...ToneOptions) (*insyra.DataTable, error) {
	if len(opts) > 1 {
		return nil, errors.New("ToneCounts accepts at most one ToneOptions")
	}
	var o ToneOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.Mode != ToneWords && o.Mode != ToneSubstring {
		return nil, fmt.Errorf("unsupported tone mode %v", o.Mode)
	}
	if o.NegationWindow < 0 {
		return nil, errors.New("the negation window must not be negative")
	}
	if insyra.IsNilList(texts) {
		return nil, errors.New("texts is nil")
	}
	terms, err := newToneTerms(lex, o)
	if err != nil {
		return nil, err
	}
	var raw []any
	texts.AtomicDo(func(l *insyra.DataList) { raw = l.Data() })

	pos, neg, words := make([]any, len(raw)), make([]any, len(raw)), make([]any, len(raw))
	for i, v := range raw {
		text := ""
		switch x := v.(type) {
		case nil:
		case string:
			text = x
		default:
			return nil, fmt.Errorf("texts holds %v at row index %d; it must be text", v, i)
		}
		if !o.CaseSensitive {
			text = strings.ToLower(text)
		}
		var p, n, w int64
		if o.Mode == ToneWords {
			p, n, w = terms.countWords(text, o.NegationWindow)
		} else {
			p, n, w = terms.countSubstrings(text, o.NegationWindow)
		}
		pos[i], neg[i], words[i] = p, n, w
	}
	return insyra.NewDataTable(
		insyra.NewDataList(pos...).SetName("Positive"),
		insyra.NewDataList(neg...).SetName("Negative"),
		insyra.NewDataList(words...).SetName("Words"),
	), nil
}

type toneKind int

const (
	tonePositive toneKind = iota + 1
	toneNegative
	toneNegation
)

type toneTerms struct {
	kind   map[string]toneKind
	byLen  []string // every term, longest first, for substring matching
	maxLen int
}

func newToneTerms(lex Lexicon, o ToneOptions) (*toneTerms, error) {
	t := &toneTerms{kind: map[string]toneKind{}}
	add := func(list []string, kind toneKind, label string) error {
		for _, term := range list {
			term = strings.TrimSpace(term)
			if term == "" {
				return fmt.Errorf("the %s list holds an empty term", label)
			}
			if !o.CaseSensitive {
				term = strings.ToLower(term)
			}
			if prev, dup := t.kind[term]; dup && prev != kind {
				return fmt.Errorf("the term %q is in more than one list", term)
			}
			t.kind[term] = kind
		}
		return nil
	}
	if err := add(lex.Positive, tonePositive, "Positive"); err != nil {
		return nil, err
	}
	if err := add(lex.Negative, toneNegative, "Negative"); err != nil {
		return nil, err
	}
	if err := add(lex.Negation, toneNegation, "Negation"); err != nil {
		return nil, err
	}
	for term := range t.kind {
		t.byLen = append(t.byLen, term)
	}
	sort.Slice(t.byLen, func(i, j int) bool {
		if len(t.byLen[i]) != len(t.byLen[j]) {
			return len(t.byLen[i]) > len(t.byLen[j])
		}
		return t.byLen[i] < t.byLen[j]
	})
	if len(t.byLen) > 0 {
		t.maxLen = len(t.byLen[0])
	}
	return t, nil
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '\'' }

func (t *toneTerms) countWords(text string, window int) (pos, neg, words int64) {
	tokens := strings.FieldsFunc(text, func(r rune) bool { return !isWordRune(r) })
	for i, tok := range tokens {
		tok = strings.Trim(tok, "'")
		words++
		switch t.kind[tok] {
		case toneNegative:
			neg++
		case tonePositive:
			negated := false
			for j := max(0, i-window); j < i; j++ {
				if t.kind[strings.Trim(tokens[j], "'")] == toneNegation {
					negated = true
					break
				}
			}
			if !negated {
				pos++
			}
		}
	}
	return pos, neg, words
}

func (t *toneTerms) countSubstrings(text string, window int) (pos, neg, words int64) {
	runes := []rune(text)
	lastNegation := -1 << 30 // the rune index where the latest negation ended
	for i := 0; i < len(runes); {
		matched := ""
		if t.maxLen > 0 {
			rest := string(runes[i:min(len(runes), i+t.maxLen)])
			for _, term := range t.byLen {
				if strings.HasPrefix(rest, term) {
					matched = term
					break
				}
			}
		}
		if matched == "" {
			if isWordRune(runes[i]) {
				words++
			}
			i++
			continue
		}
		size := len([]rune(matched))
		words += int64(size)
		switch t.kind[matched] {
		case toneNegative:
			neg++
		case tonePositive:
			if window == 0 || i-lastNegation > window {
				pos++
			}
		case toneNegation:
			lastNegation = i + size
		}
		i += size
	}
	return pos, neg, words
}

// DailyToneOptions are the settings of DailyTone.
type DailyToneOptions struct {
	// CloseHour and CloseMinute are the market close in Location. An article
	// published at or after it, or on a day that is not a trading day, counts
	// for the next trading day. The default 0:00 means every article published
	// on a trading day counts for that day.
	CloseHour, CloseMinute int
	// Location is the time zone the close is read in. Nil keeps each timestamp's
	// own zone.
	Location *time.Location
	// Window is the number of trading days of the rolling net tone. It must be
	// at least 1.
	Window int
	// ZWindow is the number of earlier days of the trailing z-score, at least 2.
	// 0 leaves the z-score NaN.
	ZWindow int
}

// DailyTone adds tone counts up by trading day without looking ahead.
//
// published holds each article's time (time.Time), counts the table from
// ToneCounts (columns Positive, Negative and Words, one row per article), and
// tradingDays the trading dates in ascending order. An article is assigned to
// the first trading day that is on or after its date, and to the next trading
// day when it was published at or after the close or on a day with no trading, so
// weekend and after-close news is seen on the next session, never the one
// already over. Articles after the last trading day are dropped, and articles
// before the first trading day count for it.
//
// The result has one row per trading day: Date, Positive, Negative and Words for
// that day; NetTone, the rolling (positive - negative) / words over the last
// Window trading days including the day; and Z, the trailing z-score of
// NetTone against the ZWindow days before the day, never including it. A day or
// window with no words is NaN.
func DailyTone(published insyra.IDataList, counts *insyra.DataTable, tradingDays insyra.IDataList, opts DailyToneOptions) (*insyra.DataTable, error) {
	if opts.Window < 1 {
		return nil, errors.New("the window must be at least 1")
	}
	if opts.ZWindow < 0 || opts.ZWindow == 1 {
		return nil, errors.New("the z-score window must be 0 or at least 2")
	}
	if opts.CloseHour < 0 || opts.CloseHour > 23 || opts.CloseMinute < 0 || opts.CloseMinute > 59 {
		return nil, errors.New("the close must be a time of day")
	}
	if insyra.IsNilList(published) || insyra.IsNilList(tradingDays) || insyra.IsNilTable(counts) {
		return nil, errors.New("published, counts and tradingDays are all needed")
	}
	var times, days []any
	var pos, neg, words []any
	published.AtomicDo(func(l *insyra.DataList) { times = l.Data() })
	tradingDays.AtomicDo(func(l *insyra.DataList) { days = l.Data() })
	counts.AtomicDo(func(dt *insyra.DataTable) {
		for name, dst := range map[string]*[]any{"Positive": &pos, "Negative": &neg, "Words": &words} {
			if col := dt.GetColByName(name); col != nil {
				*dst = col.Data()
			}
		}
	})
	if pos == nil || neg == nil || words == nil {
		return nil, errors.New("counts needs the columns Positive, Negative and Words")
	}
	if len(times) != len(pos) {
		return nil, fmt.Errorf("published has %d rows and counts has %d", len(times), len(pos))
	}

	dates := make([]time.Time, len(days))
	for i, d := range days {
		t, ok := d.(time.Time)
		if !ok {
			return nil, fmt.Errorf("tradingDays holds %v at row index %d; it must be a time.Time", d, i)
		}
		dates[i] = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		if i > 0 && !dates[i].After(dates[i-1]) {
			return nil, fmt.Errorf("tradingDays is not in ascending order at row index %d", i)
		}
	}
	dayPos := make([]float64, len(dates))
	dayNeg := make([]float64, len(dates))
	dayWords := make([]float64, len(dates))
	for i, v := range times {
		t, ok := v.(time.Time)
		if !ok {
			return nil, fmt.Errorf("published holds %v at row index %d; it must be a time.Time", v, i)
		}
		if opts.Location != nil {
			t = t.In(opts.Location)
		}
		p, errP := countCell(pos[i], "Positive", i)
		n, errN := countCell(neg[i], "Negative", i)
		w, errW := countCell(words[i], "Words", i)
		if err := errors.Join(errP, errN, errW); err != nil {
			return nil, err
		}
		date := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		afterClose := t.Hour() > opts.CloseHour || t.Hour() == opts.CloseHour && t.Minute() >= opts.CloseMinute
		// The first trading day on or after the article's date, or after it
		// when the article came at or after the close.
		idx := sort.Search(len(dates), func(k int) bool { return !dates[k].Before(date) })
		if idx < len(dates) && dates[idx].Equal(date) && afterClose && (opts.CloseHour != 0 || opts.CloseMinute != 0) {
			idx++
		}
		if idx >= len(dates) {
			continue
		}
		dayPos[idx] += p
		dayNeg[idx] += n
		dayWords[idx] += w
	}

	n := len(dates)
	net := make([]float64, n)
	for t := 0; t < n; t++ {
		var sp, sn, sw float64
		for k := max(0, t-opts.Window+1); k <= t; k++ {
			sp, sn, sw = sp+dayPos[k], sn+dayNeg[k], sw+dayWords[k]
		}
		net[t] = math.NaN()
		if sw > 0 {
			net[t] = (sp - sn) / sw
		}
	}
	outDate, outPos, outNeg, outWords, outNet, outZ := make([]any, n), make([]any, n), make([]any, n), make([]any, n), make([]any, n), make([]any, n)
	for t := 0; t < n; t++ {
		outDate[t], outPos[t], outNeg[t], outWords[t], outNet[t] = dates[t], int64(dayPos[t]), int64(dayNeg[t]), int64(dayWords[t]), net[t]
		outZ[t] = math.NaN()
		if opts.ZWindow >= 2 && t >= opts.ZWindow {
			mean, count := 0.0, 0
			for k := t - opts.ZWindow; k < t; k++ {
				if !math.IsNaN(net[k]) {
					mean += net[k]
					count++
				}
			}
			if count >= 2 && !math.IsNaN(net[t]) {
				mean /= float64(count)
				ss := 0.0
				for k := t - opts.ZWindow; k < t; k++ {
					if !math.IsNaN(net[k]) {
						ss += (net[k] - mean) * (net[k] - mean)
					}
				}
				if sd := math.Sqrt(ss / float64(count-1)); sd > 0 {
					outZ[t] = (net[t] - mean) / sd
				}
			}
		}
	}
	return insyra.NewDataTable(
		insyra.NewDataList(outDate...).SetName("Date"),
		insyra.NewDataList(outPos...).SetName("Positive"),
		insyra.NewDataList(outNeg...).SetName("Negative"),
		insyra.NewDataList(outWords...).SetName("Words"),
		insyra.NewDataList(outNet...).SetName("NetTone"),
		insyra.NewDataList(outZ...).SetName("Z"),
	), nil
}

func countCell(v any, name string, row int) (float64, error) {
	f, ok := insyra.ToFloat64Safe(v)
	if !ok || math.IsNaN(f) || math.IsInf(f, 0) || f < 0 {
		return 0, fmt.Errorf("counts column %s holds %v at row index %d; it must be a count of at least 0", name, v, row)
	}
	return f, nil
}
