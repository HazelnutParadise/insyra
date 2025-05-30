package insyra

type cclTokenType int

const (
	tEOF cclTokenType = iota
	tIDENT
	tNUMBER
	tSTRING
	tLPAREN
	tRPAREN
	tCOMMA
	tOPERATOR
)

type cclToken struct {
	typ   cclTokenType
	value string
}

type cclNode any
type cclNumberNode struct{ value float64 }
type cclStringNode struct{ value string }
type cclIdentifierNode struct{ name string }
type cclBinaryOpNode struct {
	op    string
	left  cclNode
	right cclNode
}
type funcCallNode struct {
	name string
	args []cclNode
}
