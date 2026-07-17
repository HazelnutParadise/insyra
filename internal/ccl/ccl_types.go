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

// 注意：新增「含子節點」的節點型別時，必須同步更新 exceedsDepth
// （ccl_compiler.go 的編譯期深度保護）與 Bind 的走訪分支，否則該型別的
// 子樹會漏掉深度檢查與欄位綁定。
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

// cclFoldChainNode 表示左結合運算子鏈的攤平形式：
// ((init ops[0] operands[0]) ops[1] operands[1]) ...
// parser 把同一優先級迴圈內連續的一般運算子（不含 '.' 與 ':'）攤平成
// 此節點，讓任意長度的鏈產生 O(1) 深度的 AST，求值時以迴圈左摺疊。
// 不變量：len(ops) == len(operands) >= 2（0 個運算子不建節點、1 個維持
// cclBinaryOpNode）。
type cclFoldChainNode struct {
	init     cclNode
	ops      []string
	operands []cclNode
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
