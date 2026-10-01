package ccl

import "strings"

// SequenceStream computes a built-in sequence function over a column fed in
// order, a batch at a time, holding only the values its shift or window reaches
// across a batch boundary. The outputs of every Push and the final Flush, in
// order, are what the function returns for the whole column, bit for bit.
type SequenceStream interface {
	// Push feeds the next values and returns the outputs for the rows it can
	// answer now, continuing where the previous outputs ended. A look-ahead
	// holds the last rows back until the values they need arrive, so it may
	// return fewer outputs than values.
	Push(values []any) ([]any, error)
	// Flush returns the outputs for every row still held, once the column has
	// ended.
	Flush() ([]any, error)
}

// streamableSequences names every built-in sequence function that has a
// streaming form. The four cumulative ones are here as well, and stream through
// their accumulator rather than through the windows below.
var streamableSequences = map[string]bool{
	"LAG": true, "LEAD": true, "DIFF": true, "PCT_CHANGE": true,
	"CUMSUM": true, "CUMPROD": true, "CUMMAX": true, "CUMMIN": true,
	"ROLLING_SUM": true, "ROLLING_MEAN": true, "ROLLING_MIN": true,
	"ROLLING_MAX": true, "ROLLING_STD": true,
}

// NewSequenceStream returns the streaming form of the built-in sequence
// function name, in any letter case, called with params: the arguments after
// the column, each as the one-element column the evaluator hands a constant. It
// returns false for any other name, and for a built-in name a caller
// re-registered with RegisterSequenceFunction, whose arithmetic is then the
// caller's. A param the function would refuse is an error, the function's own.
func NewSequenceStream(name string, params [][]any) (SequenceStream, bool, error) {
	kind := strings.ToUpper(name)
	if isUserSequence(name) {
		return nil, false, nil
	}
	if !streamableSequences[kind] {
		return nil, false, nil
	}
	if _, cumulative := cumulativeSequences[kind]; cumulative {
		return newCumStream(kind, params)
	}
	return newWindowStream(kind, params)
}

// isUserSequence reports whether a caller has registered name through
// RegisterSequenceFunction, in which case its arithmetic is the caller's rather
// than a built-in a stream can reproduce.
func isUserSequence(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return userSequences[strings.ToUpper(name)]
}

// checkSequenceParams asks the registered function whether it accepts these
// params, by calling it on an empty column. Every such call fails before it reads
// a value, so the caller sees the function's own message rather than a second
// copy of it written here.
func checkSequenceParams(name string, params [][]any) error {
	_, err := callSequenceFunction(name, append([][]any{{}}, params...))
	return err
}

// cumStream streams one of the four cumulative functions. The state that crosses
// a batch boundary is the running accumulator, so every value is answered as soon
// as it arrives and nothing is ever held back.
type cumStream struct {
	acc *cumAccumulator
}

func newCumStream(name string, params [][]any) (SequenceStream, bool, error) {
	if err := checkSequenceParams(name, params); err != nil {
		return nil, true, err
	}
	spec := cumulativeSequences[name]
	acc := &cumAccumulator{acc: spec.initial, seeded: !spec.seedFromFirst, combine: spec.combine}
	return &cumStream{acc: acc}, true, nil
}

func (c *cumStream) Push(values []any) ([]any, error) {
	return c.acc.feed(values), nil
}

// Flush answers nothing: the accumulator answers each row as it arrives.
func (c *cumStream) Flush() ([]any, error) {
	return nil, nil
}

// windowStream streams a shift, a difference or a window. For row i the function
// reads rows i-L to i+R, and L and R follow from the name and the argument, so
// the stream keeps one run of values: the last L inputs of the rows it has
// answered, which are the rows it may still be asked to look back at, and the
// inputs of the rows it has not. An output is released once its window is
// complete, which costs the last R rows of a batch until the next one arrives.
// Only the rows being released are computed, by the same code the function runs
// on the whole column, so each output is the one the function gives that row.
type windowStream struct {
	name       string
	params     [][]any
	lookBehind int
	lookAhead  int

	// vals holds the history and the held rows in order, and answered is the
	// position in vals of the first row not yet answered.
	vals     []any
	answered int
}

func newWindowStream(name string, params [][]any) (SequenceStream, bool, error) {
	if err := checkSequenceParams(name, params); err != nil {
		return nil, true, err
	}
	lookBehind, lookAhead, err := sequenceWindow(name, params)
	if err != nil {
		return nil, true, err
	}
	// Copied, so that a caller reusing the slice it passed cannot change what
	// this stream calls the function with.
	kept := append([][]any(nil), params...)
	return &windowStream{name: name, params: kept, lookBehind: lookBehind, lookAhead: lookAhead}, true, nil
}

// sequenceWindow reads how many rows before and after a row the function reads:
// LAG(x, p) has L = p for p >= 0 and R = -p otherwise, LEAD the reverse,
// DIFF and PCT_CHANGE have L = p and R = 0, and ROLLING_*(x, w) has L = w - 1
// and R = 0. The params are read through the same helper the functions use, so
// one that is not a whole number is refused with the message the function gives.
func sequenceWindow(name string, params [][]any) (lookBehind, lookAhead int, err error) {
	switch name {
	case "LAG", "LEAD":
		periods, err := scalarInt(params[0], name, "periods")
		if err != nil {
			return 0, 0, err
		}
		// LAG reads the row p before it and LEAD the row p after it, so a
		// negative count sends each of them the other way.
		behind, ahead := periods, -periods
		if name == "LEAD" {
			behind, ahead = -periods, periods
		}
		return max(behind, 0), max(ahead, 0), nil
	case "DIFF", "PCT_CHANGE":
		periods := 1
		if len(params) > 0 {
			p, err := scalarInt(params[0], name, "periods")
			if err != nil {
				return 0, 0, err
			}
			periods = p
		}
		return periods, 0, nil
	}
	// The window functions reduce the w rows ending at the row, so they look w-1
	// behind it and nothing ahead. The windowed function has already refused a
	// window that is not positive.
	window, err := scalarInt(params[0], name, "window")
	if err != nil {
		return 0, 0, err
	}
	return window - 1, 0, nil
}

func (w *windowStream) Push(values []any) ([]any, error) {
	// Appended, so the stream owns every value it holds and never aliases the
	// slice the caller passed in.
	w.vals = append(w.vals, values...)

	// The rows before answered are answered already, and the last lookAhead rows
	// are still waiting for the values after them. The rest have their whole
	// window in vals.
	ready := len(w.vals) - w.lookAhead - w.answered
	if ready <= 0 {
		return nil, nil
	}
	out, err := windowedSequences[w.name](w.vals, w.params, w.answered, w.answered+ready)
	if err != nil {
		return nil, err
	}
	w.answered += ready
	w.dropAnswered()
	return out, nil
}

// dropAnswered lets go of the values no later row reads: every answered row more
// than lookBehind before the first one not yet answered. It copies the rest into a
// new slice once the dropped part is over half of what is held, so a value is
// copied a constant number of times however long the column is, and the memory
// of the dropped part goes back with the old slice.
func (w *windowStream) dropAnswered() {
	drop := w.answered - w.lookBehind
	if drop <= len(w.vals)/2 {
		return
	}
	w.vals = append([]any(nil), w.vals[drop:]...)
	w.answered -= drop
}

// Flush answers the rows still held, which no further value could have settled,
// so a shift or a window reaching past the end of the column gives the same
// answers it gives on the whole column.
func (w *windowStream) Flush() ([]any, error) {
	if w.answered == len(w.vals) {
		return nil, nil
	}
	out, err := windowedSequences[w.name](w.vals, w.params, w.answered, len(w.vals))
	if err != nil {
		return nil, err
	}
	w.answered = len(w.vals)
	return out, nil
}
