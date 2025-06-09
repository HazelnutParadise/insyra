package ccl

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
	tBOOLEAN // 新增布林值標記類型
)

type cclToken struct {
	typ   cclTokenType
	value string
}

type cclNode any
type cclNumberNode struct{ value float64 }
type cclStringNode struct{ value string }
type cclIdentifierNode struct{ name string }
type cclBooleanNode struct{ value bool } // 新增布林值節點
type cclBinaryOpNode struct {
	op    string
	left  cclNode
	right cclNode
}

// 新增連續比較運算節點
type cclChainedComparisonNode struct {
	ops    []string  // 運算符, 例如 "<", "<="
	values []cclNode // 值列表，例如 [1, A, B]
}
type funcCallNode struct {
	name string
	args []cclNode
}
