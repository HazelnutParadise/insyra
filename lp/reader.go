package lp

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// readLP reads CPLEX LP text the way GLPK 5.0's reader does (glp_read_lp in
// src/api/cplex.c), so one file means the same model to both engines: the same
// keywords, the same bounds, the same column order and the same errors, with
// the same line numbers.
//
// That includes GLPK's surprises. A keyword is recognised only at the very
// start of a line, and there any prefix of one counts, so a line starting with
// "e:" ends the file as if it said "End". A variable in a Binary section keeps
// a bound given in Bounds and only gains 0 or 1 on a side left open.
//
// One place is stricter than GLPK on purpose. GLPK reads "x >= -foo" as "x has
// no lower bound", because a test in that branch is inverted; this reader
// reports a missing lower bound instead of solving a different model. The
// CPLEX format allows only a number or an infinity after the sign, and the
// inverted test was confirmed upstream in September 2026 (bug-glpk archive,
// 2026-09). No GLPK release carries the fix.
func readLP(name string, text []byte) (*problem, error) {
	r := &lpReader{
		name: name, text: text, c: '\n',
		cols: map[string]int{}, rowNames: map[string]bool{},
		p: &problem{},
	}
	r.scanToken()
	if r.token != tMinimize && r.token != tMaximize {
		r.fail("'minimize' or 'maximize' keyword missing")
	}
	if r.err == nil {
		r.parseObjective()
	}
	if r.err == nil && r.token != tSubjectTo {
		r.fail("constraints section missing")
	}
	if r.err == nil {
		r.parseConstraints()
	}
	if r.err == nil && r.token == tBounds {
		r.parseBounds()
	}
	for r.err == nil && (r.token == tGeneral || r.token == tInteger || r.token == tBinary) {
		r.parseInteger()
	}
	if r.err == nil {
		switch r.token {
		case tEnd:
			r.scanToken()
		case tEOF:
			r.warn("keyword 'end' missing")
		default:
			r.fail("symbol '%s' in wrong position", r.image)
		}
	}
	if r.err == nil && r.token != tEOF {
		r.fail("extra symbol(s) detected beyond 'end'")
	}
	if r.err != nil {
		return nil, r.err
	}

	for j := range r.p.cols {
		lower, upper := r.lb[j], r.ub[j]
		if lower == unsetLower {
			lower = 0
		}
		if upper == unsetUpper {
			upper = math.MaxFloat64
		}
		// GLPK stores an infinite bound as ±DBL_MAX.
		if lower == -math.MaxFloat64 {
			lower = math.Inf(-1)
		}
		if upper == math.MaxFloat64 {
			upper = math.Inf(1)
		}
		r.p.cols[j].lower, r.p.cols[j].upper = lower, upper
	}
	r.p.warnings = r.warnings
	return r.p, nil
}

type tokenKind int

const (
	tEOF tokenKind = iota
	tMinimize
	tMaximize
	tSubjectTo
	tBounds
	tGeneral
	tInteger
	tBinary
	tEnd
	tName
	tNumber
	tPlus
	tMinus
	tColon
	tLE
	tGE
	tEQ
)

const (
	eof = -1
	// A bound nobody set, as GLPK marks it: the lower side +DBL_MAX, the upper
	// side -DBL_MAX.
	unsetLower = math.MaxFloat64
	unsetUpper = -math.MaxFloat64
	// maxToken is GLPK's token length limit.
	maxToken = 255
	// nameChars are the characters besides letters and digits a name may hold.
	// A name cannot start with a digit or '.'.
	nameChars = "!\"#$%&()/,.;?@_`'{}|~"
)

type lpReader struct {
	name  string
	text  []byte
	pos   int
	c     int // current character, a space for any whitespace but '\n', or eof
	count int // current line number

	token tokenKind
	image string
	value float64

	p              *problem
	cols           map[string]int
	rowNames       map[string]bool
	lb, ub         []float64
	lbWarn, ubWarn bool
	warnings       []string
	err            error
}

// fail records the first error and stops reading: the input is treated as
// ended, so every parsing loop finishes without reading further.
func (r *lpReader) fail(format string, args ...any) {
	if r.err == nil {
		r.err = fmt.Errorf("%s:%d: %s", r.name, r.count, fmt.Sprintf(format, args...))
	}
	r.token, r.c, r.pos = tEOF, eof, len(r.text)
}

func (r *lpReader) warn(format string, args ...any) {
	if r.err == nil {
		r.warnings = append(r.warnings, fmt.Sprintf("%s:%d: warning: %s", r.name, r.count, fmt.Sprintf(format, args...)))
	}
}

func (r *lpReader) readChar() {
	if r.c == eof {
		return
	}
	if r.c == '\n' {
		r.count++
	}
	if r.pos >= len(r.text) {
		if r.c == '\n' {
			r.count--
			r.c = eof
		} else {
			r.warn("missing final end of line")
			r.c = '\n'
		}
		return
	}
	c := int(r.text[r.pos])
	r.pos++
	switch {
	case c == '\n':
	case c == ' ' || c == '\t' || c == '\v' || c == '\f' || c == '\r':
		c = ' '
	case c < 0x20 || c == 0x7f:
		r.fail("invalid control character 0x%02X", c)
		return
	}
	r.c = c
}

func (r *lpReader) addChar() {
	if len(r.image) == maxToken {
		r.fail("token '%.15s...' too long", r.image)
		return
	}
	r.image += string(rune(r.c))
	r.readChar()
}

func isAlpha(c int) bool    { return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' }
func isDigit(c int) bool    { return '0' <= c && c <= '9' }
func isNameChar(c int) bool { return c >= 0 && c < 0x80 && strings.IndexByte(nameChars, byte(c)) >= 0 }

func lower(c int) int {
	if 'A' <= c && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

// sameAs reports whether image is a case-insensitive prefix of keyword. GLPK
// compares only as many characters as the image has, so "e" is "end".
func sameAs(image, keyword string) bool {
	return len(image) <= len(keyword) && strings.EqualFold(image, keyword[:len(image)])
}

func isInfinity(image string) bool { return sameAs(image, "infinity") || sameAs(image, "inf") }

func (r *lpReader) scanToken() {
	r.token, r.image, r.value = tEOF, "", 0
	for {
		keyword := false
		for r.c == ' ' {
			r.readChar()
		}
		switch {
		case r.c == eof:
			r.token = tEOF
		case r.c == '\n':
			r.readChar()
			if !isAlpha(r.c) {
				continue
			}
			keyword = true
			r.scanName(keyword)
		case r.c == '\\':
			for r.c != '\n' && r.c != eof {
				r.readChar()
			}
			continue
		case isAlpha(r.c) || r.c != '.' && isNameChar(r.c):
			r.scanName(keyword)
		case isDigit(r.c) || r.c == '.':
			r.scanNumber()
		case r.c == '+':
			r.token = tPlus
			r.addChar()
		case r.c == '-':
			r.token = tMinus
			r.addChar()
		case r.c == ':':
			r.token = tColon
			r.addChar()
		case r.c == '<':
			r.token = tLE
			r.addChar()
			if r.c == '=' {
				r.addChar()
			}
		case r.c == '>':
			r.token = tGE
			r.addChar()
			if r.c == '=' {
				r.addChar()
			}
		case r.c == '=':
			r.token = tEQ
			r.addChar()
			switch r.c {
			case '<':
				r.token = tLE
				r.addChar()
			case '>':
				r.token = tGE
				r.addChar()
			}
		case r.c < 0x80:
			r.fail("character '%c' not recognized", rune(r.c))
		default:
			r.fail("character 0x%02X not recognized", r.c)
		}
		break
	}
	if r.err != nil {
		r.token = tEOF
		return
	}
	for r.c == ' ' {
		r.readChar()
	}
}

func (r *lpReader) scanName(keyword bool) {
	r.token = tName
	for isAlpha(r.c) || isDigit(r.c) || isNameChar(r.c) {
		r.addChar()
	}
	if !keyword || r.err != nil {
		return
	}
	img := r.image
	switch {
	case sameAs(img, "minimize"), sameAs(img, "minimum"), sameAs(img, "min"):
		r.token = tMinimize
	case sameAs(img, "maximize"), sameAs(img, "maximum"), sameAs(img, "max"):
		r.token = tMaximize
	case sameAs(img, "subject"):
		r.scanTwoWordKeyword("to", "subject to")
	case sameAs(img, "such"):
		r.scanTwoWordKeyword("that", "such that")
	case sameAs(img, "st"), sameAs(img, "s.t."), sameAs(img, "st."):
		r.token = tSubjectTo
	case sameAs(img, "bounds"), sameAs(img, "bound"):
		r.token = tBounds
	case sameAs(img, "general"), sameAs(img, "generals"), sameAs(img, "gen"):
		r.token = tGeneral
	case sameAs(img, "integer"), sameAs(img, "integers"), sameAs(img, "int"):
		r.token = tInteger
	case sameAs(img, "binary"), sameAs(img, "binaries"), sameAs(img, "bin"):
		r.token = tBinary
	case sameAs(img, "end"):
		r.token = tEnd
	}
}

// scanTwoWordKeyword finishes "subject to" or "such that" after the first
// word. The words must be one space apart; anything else leaves the first word
// an ordinary name.
func (r *lpReader) scanTwoWordKeyword(second, keyword string) {
	if r.c != ' ' {
		return
	}
	r.readChar()
	if lower(r.c) != int(second[0]) {
		return
	}
	r.token = tSubjectTo
	r.image += " "
	r.addChar()
	for i := 1; i < len(second) && r.err == nil; i++ {
		if lower(r.c) != int(second[i]) {
			r.fail("keyword '%s' incomplete", keyword)
			return
		}
		r.addChar()
	}
	if r.err == nil && isAlpha(r.c) {
		r.fail("keyword '%s%c...' not recognized", r.image, rune(r.c))
	}
}

func (r *lpReader) scanNumber() {
	r.token = tNumber
	for isDigit(r.c) {
		r.addChar()
	}
	if r.c == '.' {
		r.addChar()
		if len(r.image) == 1 && !isDigit(r.c) {
			r.fail("invalid use of decimal point")
			return
		}
		for isDigit(r.c) {
			r.addChar()
		}
	}
	if r.c == 'e' || r.c == 'E' {
		r.addChar()
		if r.c == '+' || r.c == '-' {
			r.addChar()
		}
		if !isDigit(r.c) {
			r.fail("numeric constant '%s' incomplete", r.image)
			return
		}
		for isDigit(r.c) {
			r.addChar()
		}
	}
	if r.err != nil {
		return
	}
	v, err := strconv.ParseFloat(r.image, 64)
	if err != nil {
		r.fail("numeric constant '%s' out of range", r.image)
		return
	}
	r.value = v
}

func (r *lpReader) findCol(name string) int {
	if j, ok := r.cols[name]; ok {
		return j
	}
	j := len(r.p.cols)
	r.cols[name] = j
	r.p.cols = append(r.p.cols, column{name: name})
	r.lb = append(r.lb, unsetLower)
	r.ub = append(r.ub, unsetUpper)
	return j
}

// parseLinearForm reads terms like "3 x - y + 2.5 z". GLPK drops a term whose
// coefficient is zero once the form is read.
func (r *lpReader) parseLinearForm() []term {
	var terms []term
	used := map[int]bool{}
	for {
		sign := r.optionalSign()
		coef := 1.0
		if r.token == tNumber {
			coef = r.value
			r.scanToken()
		}
		if r.token != tName {
			r.fail("missing variable name")
			return nil
		}
		j := r.findCol(r.image)
		if used[j] {
			r.fail("multiple use of variable '%s' not allowed", r.image)
			return nil
		}
		used[j] = true
		terms = append(terms, term{col: j, coef: sign * coef})
		r.scanToken()
		if r.token != tPlus && r.token != tMinus {
			break
		}
	}
	kept := terms[:0]
	for _, t := range terms {
		if t.coef != 0 {
			kept = append(kept, t)
		}
	}
	return kept
}

func (r *lpReader) parseObjective() {
	r.p.maximize = r.token == tMaximize
	r.scanToken()
	if r.token == tName && r.c == ':' {
		r.scanToken() // the colon
		r.scanToken()
	}
	r.p.objective = r.parseLinearForm()
}

func (r *lpReader) parseConstraints() {
	r.scanToken()
	for r.err == nil {
		if r.token == tName && r.c == ':' {
			if r.rowNames[r.image] {
				r.fail("constraint '%s' multiply defined", r.image)
				return
			}
			r.rowNames[r.image] = true
			r.scanToken() // the colon
			r.scanToken()
		} else {
			// An unnamed row is named after the line it starts on, and an
			// explicit name met later may collide with it.
			r.rowNames[fmt.Sprintf("r.%d", r.count)] = true
		}
		row := row{terms: r.parseLinearForm()}
		switch r.token {
		case tLE:
			row.rel = lessEq
		case tGE:
			row.rel = greaterEq
		case tEQ:
			row.rel = equalTo
		default:
			r.fail("missing constraint sense")
			return
		}
		r.scanToken()
		sign := r.optionalSign()
		if r.token != tNumber {
			r.fail("missing right-hand side")
			return
		}
		row.rhs = sign * r.value
		r.p.rows = append(r.p.rows, row)
		if r.c != '\n' && r.c != eof {
			r.fail("invalid symbol(s) beyond right-hand side")
			return
		}
		r.scanToken()
		if !r.startsDefinition() {
			return
		}
	}
}

// optionalSign consumes a leading '+' or '-' and returns the sign it stands
// for, 1 when there is none.
func (r *lpReader) optionalSign() float64 {
	switch r.token {
	case tPlus:
		r.scanToken()
	case tMinus:
		r.scanToken()
		return -1
	}
	return 1
}

// startsDefinition reports whether the current token can begin a constraint or
// a bound definition.
func (r *lpReader) startsDefinition() bool {
	switch r.token {
	case tPlus, tMinus, tNumber, tName:
		return true
	}
	return false
}

func (r *lpReader) setLowerBound(j int, v float64) {
	if r.lb[j] != unsetLower && !r.lbWarn {
		r.warn("lower bound of variable '%s' redefined", r.p.cols[j].name)
		r.lbWarn = true
	}
	r.lb[j] = v
}

func (r *lpReader) setUpperBound(j int, v float64) {
	if r.ub[j] != unsetUpper && !r.ubWarn {
		r.warn("upper bound of variable '%s' redefined", r.p.cols[j].name)
		r.ubWarn = true
	}
	r.ub[j] = v
}

// signedBound reads what follows a sign: a number, which it consumes, or an
// infinity, which it leaves as the current token so a caller refusing it
// reports the line it is on. ok is false when it is neither.
func (r *lpReader) signedBound() (value float64, infinite, ok bool) {
	sign := 1.0
	if r.token == tMinus {
		sign = -1
	}
	r.scanToken()
	if r.token == tNumber {
		value = sign * r.value
		r.scanToken()
		return value, false, true
	}
	if isInfinity(r.image) {
		return sign * math.MaxFloat64, true, true
	}
	return 0, false, false
}

// signedLowerBound reads a signed lower bound, refusing +inf.
func (r *lpReader) signedLowerBound() (float64, bool) {
	v, infinite, ok := r.signedBound()
	if !ok {
		r.fail("missing lower bound")
		return 0, false
	}
	if infinite {
		if v > 0 {
			r.fail("invalid use of '+inf' as lower bound")
			return 0, false
		}
		r.scanToken()
	}
	return v, true
}

func (r *lpReader) parseBounds() {
	r.scanToken()
	for r.err == nil && r.startsDefinition() {
		hasLower, lowerValue := false, 0.0
		switch r.token {
		case tPlus, tMinus:
			v, ok := r.signedLowerBound()
			if !ok {
				return
			}
			hasLower, lowerValue = true, v
		case tNumber:
			hasLower, lowerValue = true, r.value
			r.scanToken()
		}
		if hasLower {
			if r.token != tLE {
				r.fail("missing '<', '<=', or '=<' after lower bound")
				return
			}
			r.scanToken()
		}
		if r.token != tName {
			r.fail("missing variable name")
			return
		}
		j := r.findCol(r.image)
		if hasLower {
			r.setLowerBound(j, lowerValue)
		}
		r.scanToken()

		switch {
		case r.token == tLE:
			r.scanToken()
			switch r.token {
			case tPlus, tMinus:
				v, infinite, ok := r.signedBound()
				if !ok {
					r.fail("missing upper bound")
					return
				}
				if infinite {
					if v < 0 {
						r.fail("invalid use of '-inf' as upper bound")
						return
					}
					r.scanToken()
				}
				r.setUpperBound(j, v)
			case tNumber:
				r.setUpperBound(j, r.value)
				r.scanToken()
			default:
				r.fail("missing upper bound")
				return
			}
		case r.token == tGE:
			if hasLower {
				r.fail("invalid bound definition")
				return
			}
			r.scanToken()
			switch r.token {
			case tPlus, tMinus:
				v, ok := r.signedLowerBound()
				if !ok {
					return
				}
				r.setLowerBound(j, v)
			case tNumber:
				r.setLowerBound(j, r.value)
				r.scanToken()
			default:
				r.fail("missing lower bound")
				return
			}
		case r.token == tEQ:
			if hasLower {
				r.fail("invalid bound definition")
				return
			}
			r.scanToken()
			sign := r.optionalSign()
			if r.token != tNumber {
				r.fail("missing fixed value")
				return
			}
			r.setLowerBound(j, sign*r.value)
			r.setUpperBound(j, sign*r.value)
			r.scanToken()
		case sameAs(r.image, "free"):
			if hasLower {
				r.fail("invalid bound definition")
				return
			}
			r.setLowerBound(j, -math.MaxFloat64)
			r.setUpperBound(j, math.MaxFloat64)
			r.scanToken()
		case !hasLower:
			r.fail("invalid bound definition")
			return
		}
	}
}

func (r *lpReader) parseInteger() {
	binary := r.token == tBinary
	r.scanToken()
	for r.err == nil && r.token == tName {
		j := r.findCol(r.image)
		r.p.cols[j].integer = true
		if binary {
			// Since GLPK 4.52 a bound from the Bounds section survives.
			lowerValue, upperValue := r.lb[j], r.ub[j]
			if lowerValue == unsetLower {
				lowerValue = 0
			}
			if upperValue == unsetUpper {
				upperValue = 1
			}
			r.setLowerBound(j, lowerValue)
			r.setUpperBound(j, upperValue)
		}
		r.scanToken()
	}
}
