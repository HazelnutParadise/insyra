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
	tBOOLEAN   // 布林值標記類型
	tCOL_INDEX // [A] 形式的欄位索引引用
	tCOL_NAME  // ['colName'] 形式的欄位名稱引用
)

type cclToken struct {
	typ   cclTokenType
	value string
}

// CCLNode is the exported type alias for compiled CCL AST nodes.
// This allows external packages to store and reuse compiled formulas.
type CCLNode = cclNode

type cclNode any
type cclNumberNode struct{ value float64 }
type cclStringNode struct{ value string }
type cclIdentifierNode struct{ name string }
type cclBooleanNode struct{ value bool }    // 布林值節點
type cclColIndexNode struct{ index string } // [A] 形式的欄位索引引用節點
type cclColNameNode struct{ name string }   // ['colName'] 形式的欄位名稱引用節點
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
