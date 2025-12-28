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
	tNIL       // nil 標記類型
	tCOL_INDEX // [A] 形式的欄位索引引用
	tCOL_NAME  // ['colName'] 形式的欄位名稱引用
	tSEMICOLON // ; 分號，用於分隔多條 CCL 語句
	tASSIGN    // = 賦值運算符
	tDOT       // . 運算符，用於指定列
	tAT        // @ 運算符，用於表示所有欄
	tROW_INDEX // # 運算符，用於表示當前行索引
	tCOLON     // : 運算符，用於表示範圍
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
type cclAtNode struct{}                     // @ 形式的節點
type cclRowIndexNode struct{}               // # 形式的節點
type cclBooleanNode struct{ value bool }    // 布林值節點
type cclNilNode struct{}                    // nil 節點
type cclColIndexNode struct{ index string } // [A] 形式的欄位索引引用節點
type cclColNameNode struct{ name string }   // ['colName'] 形式的欄位名稱引用節點
type cclResolvedColNode struct {
	index int
	name  string // Optional: original name for fallback or error messages
}
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

// cclAssignmentNode 賦值語句節點
type cclAssignmentNode struct {
	target string  // 賦值目標（欄位名稱或索引）
	expr   cclNode // 賦值表達式
}

// cclNewColNode 創建新欄位節點
type cclNewColNode struct {
	colName string  // 新欄位名稱
	expr    cclNode // 計算表達式
}
