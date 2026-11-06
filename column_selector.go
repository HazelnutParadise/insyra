package insyra

// NameSelector marks a selector as a column name rather than an index. Build
// one with Name.
type NameSelector struct{ value string }

// Name says that a selector is a name rather than an index. A selector that
// takes it, such as arrowipc.ReadOptions.Columns, reads a bare string as an
// Excel-style index ("A", "B", ... "AA"), an int as a 0-based position, and a
// Name as a name compared exactly, so a call means the same thing whatever the
// columns happen to be called. It is the same type and rule as in 0.4.
func Name(value string) NameSelector { return NameSelector{value: value} }

// String returns the name itself, so a selector reads as what it selects.
func (n NameSelector) String() string { return n.value }

// Value returns the name a selector carries.
func (n NameSelector) Value() string { return n.value }
