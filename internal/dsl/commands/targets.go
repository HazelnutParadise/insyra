package commands

import (
	"fmt"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

// A command that cannot do what it was asked must say so. These helpers check
// the target BEFORE calling the library, so the message names what was missing
// instead of reporting success and leaving the user to notice later.
//
// They read the table's own name lists rather than the Get* lookups, which
// record an error on the table when they miss; a check should not leave a mark
// on the caller's data.

// How a CLI token picks a column or a row, ruled by the owner on #315
// (2026-09-27). A token carries no type, so a bare one is read every way it
// can be: digits as a 0-based number, negative counting from the end; letters
// as an Excel-style index, for columns only; and as a name, compared exactly.
// When every reading that lands agrees, that is the target. When two land on
// different targets the command refuses instead of guessing, because a wrong
// column is silent and an error costs one retype. number:, index: and name:
// force one reading.
//
// Letters are why neither order is safe on its own: they collide with real
// names (a, x, ID) in a way csvkit's and xsv's numeric positions rarely do.

const (
	numberPrefix = "number:"
	indexPrefix  = "index:"
	namePrefix   = "name:"
)

// resolveColumnToken returns the 0-based position of the column token picks.
func resolveColumnToken(cmd string, table *insyra.DataTable, token string) (int, error) {
	return resolveTarget(cmd, "column", table.ColNames(), true, token)
}

// resolveRowToken returns the 0-based position of the row token picks. Rows
// have no letters, so a row token is a number or a name.
func resolveRowToken(cmd string, table *insyra.DataTable, token string) (int, error) {
	return resolveTarget(cmd, "row", table.RowNames(), false, token)
}

// resolveColumnTokens resolves each token, stopping at the first that fails.
func resolveColumnTokens(cmd string, table *insyra.DataTable, tokens []string) ([]int, error) {
	out := make([]int, len(tokens))
	for i, token := range tokens {
		pos, err := resolveColumnToken(cmd, table, token)
		if err != nil {
			return nil, err
		}
		out[i] = pos
	}
	return out, nil
}

// reading is one way a token lands on a target.
type reading struct {
	pos    int
	label  string // how the token was read, e.g. `index A` or `the column named "a"`
	detail string // what that reading lands on, for an ambiguity message
	force  string // the prefixed spelling that picks this reading alone
}

// resolveTarget applies the rule to one token. names lists every target in
// order, "" for one without a name, so its length is the number of targets.
func resolveTarget(cmd, kind string, names []string, letters bool, token string) (int, error) {
	count := len(names)
	switch {
	case strings.HasPrefix(token, numberPrefix):
		raw := strings.TrimPrefix(token, numberPrefix)
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, fmt.Errorf("%s: %q is not a number; %s takes a 0-based %s number", cmd, raw, numberPrefix, kind)
		}
		pos, ok := numberPosition(n, count)
		if !ok {
			return 0, outOfRange(cmd, kind, strconv.Itoa(n), count)
		}
		return pos, nil
	case strings.HasPrefix(token, indexPrefix):
		raw := strings.TrimPrefix(token, indexPrefix)
		if !letters {
			return 0, fmt.Errorf("%s: %s picks a column by its letters, and a %s has none; write %s<n> or %s<name>", cmd, token, kind, numberPrefix, namePrefix)
		}
		pos, ok := insyra.ParseColIndex(raw)
		if !ok {
			return 0, fmt.Errorf("%s: %q is not a column index; %s takes Excel-style letters such as A or AB", cmd, raw, indexPrefix)
		}
		if pos >= count {
			return 0, outOfRange(cmd, kind, strings.ToUpper(raw), count)
		}
		return pos, nil
	case strings.HasPrefix(token, namePrefix):
		raw := strings.TrimPrefix(token, namePrefix)
		pos, found, err := namedPosition(cmd, kind, names, raw)
		if err != nil {
			return 0, err
		}
		if !found {
			return 0, notFound(cmd, kind, raw, names)
		}
		return pos, nil
	}

	var readings []reading
	numeric := false
	if n, err := strconv.Atoi(token); err == nil {
		numeric = true
		if pos, ok := numberPosition(n, count); ok {
			readings = append(readings, reading{pos, fmt.Sprintf("number %d", n), targetName(kind, names, pos), numberPrefix + token})
		}
	} else if letters {
		if pos, ok := insyra.ParseColIndex(token); ok && pos < count {
			readings = append(readings, reading{pos, "index " + strings.ToUpper(token), fmt.Sprintf("%s, number %d", targetName(kind, names, pos), pos), indexPrefix + token})
		}
	}
	pos, found, err := namedPosition(cmd, kind, names, token)
	if err != nil {
		return 0, err
	}
	if found {
		readings = append(readings, reading{pos, fmt.Sprintf("the %s named %q", kind, token), fmt.Sprintf("number %d", pos), namePrefix + token})
	}

	switch {
	case len(readings) == 0 && numeric:
		return 0, outOfRange(cmd, kind, token, count)
	case len(readings) == 0:
		return 0, notFound(cmd, kind, token, names)
	case len(readings) == 2 && readings[0].pos != readings[1].pos:
		a, b := readings[0], readings[1]
		return 0, fmt.Errorf("%s: %q could mean %s (%s) or %s (%s); write %s or %s",
			cmd, token, a.label, a.detail, b.label, b.detail, a.force, b.force)
	}
	return readings[0].pos, nil
}

// numberPosition turns a 0-based number, negative counting from the end, into
// a position.
func numberPosition(n, count int) (int, bool) {
	if n < 0 {
		n += count
	}
	return n, n >= 0 && n < count
}

// namedPosition finds the one target called name. Two targets sharing it is
// an error, since the name cannot say which.
func namedPosition(cmd, kind string, names []string, name string) (int, bool, error) {
	var hits []string
	pos := -1
	for i, n := range names {
		if n == name && name != "" {
			if pos < 0 {
				pos = i
			}
			hits = append(hits, strconv.Itoa(i))
		}
	}
	if len(hits) > 1 {
		return 0, false, fmt.Errorf("%s: %d %ss are named %q (numbers %s); write %s<n> to pick one", cmd, len(hits), kind, name, strings.Join(hits, ", "), numberPrefix)
	}
	return pos, pos >= 0, nil
}

// targetName says what the target at pos is called, for an ambiguity message.
func targetName(kind string, names []string, pos int) string {
	if names[pos] == "" {
		return "an unnamed " + kind
	}
	return fmt.Sprintf("the %s named %q", kind, names[pos])
}

func outOfRange(cmd, kind, what string, count int) error {
	return fmt.Errorf("%s: %s %s is out of range (the table has %d %ss)", cmd, kind, what, count, kind)
}

func notFound(cmd, kind, token string, names []string) error {
	if kind == "row" {
		return fmt.Errorf("%s: row %q not found", cmd, token)
	}
	return fmt.Errorf("%s: column %q not found (available: %s)", cmd, token, strings.Join(names, ", "))
}

// checkTableErr turns an error the library recorded on the table into a
// returned error, so a failed operation cannot be reported as a success.
func checkTableErr(cmd string, table *insyra.DataTable) error {
	if err := table.PopErr(); err != nil {
		return fmt.Errorf("%s: %w", cmd, err)
	}
	return nil
}

// parseSortDirection accepts the documented spellings and nothing else: a
// misspelling must not silently mean ascending.
func parseSortDirection(raw string) (descending bool, err error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "asc", "ascending":
		return false, nil
	case "desc", "descending":
		return true, nil
	default:
		return false, fmt.Errorf("invalid direction %q (use asc or desc)", raw)
	}
}

// colSelectors resolves column tokens into library selectors, so a column
// list follows the same rule as a single column.
func colSelectors(cmd string, table *insyra.DataTable, tokens []string) ([]any, error) {
	positions, err := resolveColumnTokens(cmd, table, tokens)
	if err != nil {
		return nil, err
	}
	out := make([]any, len(positions))
	for i, pos := range positions {
		out[i] = selectorAt(table, pos)
	}
	return out, nil
}

// selectorAt is the library selector for the column at pos: its name when the
// name picks it alone, and its position otherwise. The name is preferred
// because some operations name what they produce after the selector they were
// given, and a resample by position labelled its output columns B, C, D
// instead of Open, High, Low.
func selectorAt(table *insyra.DataTable, pos int) any {
	names := table.ColNames()
	name := names[pos]
	if name == "" {
		return pos
	}
	for i, other := range names {
		if other == name && i != pos {
			return pos
		}
	}
	return insyra.Name(name)
}

// colSelector resolves one column token into a position the library takes as
// a selector.
func colSelector(cmd string, table *insyra.DataTable, token string) (any, error) {
	pos, err := resolveColumnToken(cmd, table, token)
	if err != nil {
		return nil, err
	}
	return selectorAt(table, pos), nil
}

// splitColumnSpec splits a `<col>:<op>[:<name>]` spec on ':' into at most n
// parts (n < 0 for all), keeping a number:, index: or name: prefix on the
// column part so `name:price:sum` reads as the column name:price.
func splitColumnSpec(spec string, n int) []string {
	for _, prefix := range []string{numberPrefix, indexPrefix, namePrefix} {
		if strings.HasPrefix(spec, prefix) {
			parts := strings.SplitN(strings.TrimPrefix(spec, prefix), ":", n)
			parts[0] = prefix + parts[0]
			return parts
		}
	}
	return strings.SplitN(spec, ":", n)
}
