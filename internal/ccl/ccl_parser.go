package ccl

import (
	"fmt"
	"strconv"
	"strings"
)

type parser struct {
	tokens []cclToken
	pos    int
}

// ParseExpression parses a CCL expression (no assignment).
// For expressions like: A + B * C, IF(A > 0, 1, 0)
func ParseExpression(tokens []cclToken) (cclNode, error) {
	p := &parser{tokens: tokens}
	return p.parseExpression(0)
}

// ParseStatement parses a single statement that may include assignment
// Returns the parsed node which can be either an expression or an assignment
func ParseStatement(tokens []cclToken) (cclNode, error) {
	p := &parser{tokens: tokens}
	return p.parseStatement()
}

// parseStatement handles both assignments and expressions
func (p *parser) parseStatement() (cclNode, error) {
	tok := p.current()

	// Check for assignment: IDENT = expr or ['colName'] = expr or [colIndex] = expr
	if tok.typ == tIDENT || tok.typ == tCOL_NAME || tok.typ == tCOL_INDEX {
		// Look ahead for assignment operator
		if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].typ == tASSIGN {
			var target string
			switch tok.typ {
			case tIDENT:
				target = tok.value
			case tCOL_NAME:
				target = "'" + tok.value + "'" // Mark it as column name
			case tCOL_INDEX:
				target = tok.value // Column index like A, B, C
			}
			p.advance() // Skip target
			p.advance() // Skip '='
			expr, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			return &cclAssignmentNode{target: target, expr: expr}, nil
		}
	}

	// Check for NEW function: NEW('colName') = expr
	if tok.typ == tIDENT && strings.ToUpper(tok.value) == "NEW" {
		return p.parseNewFunction()
	}

	// Otherwise, parse as expression
	return p.parseExpression(0)
}

// parseNewFunction parses NEW('colName') = expr syntax
func (p *parser) parseNewFunction() (cclNode, error) {
	p.advance() // Skip NEW
	if p.current().typ != tLPAREN {
		return nil, fmt.Errorf("expected '(' after NEW")
	}
	p.advance() // Skip '('

	// Parse column name (must be a string)
	if p.current().typ != tSTRING {
		return nil, fmt.Errorf("NEW requires a string literal for column name")
	}
	colName := p.current().value
	p.advance()

	// Expect closing parenthesis
	if p.current().typ != tRPAREN {
		return nil, fmt.Errorf("expected ')' after column name in NEW")
	}
	p.advance()

	// Expect assignment operator
	if p.current().typ != tASSIGN {
		return nil, fmt.Errorf("expected '=' after NEW('colName')")
	}
	p.advance()

	// Parse expression
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	return &cclNewColNode{colName: colName, expr: expr}, nil
}

func (p *parser) current() cclToken {
	if p.pos >= len(p.tokens) {
		return cclToken{typ: tEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() {
	p.pos++
}

func (p *parser) parseExpression(precedence int) (cclNode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	// 處理連續比較運算，例如 1 < A <= B
	tok := p.current()
	if tok.typ == tOPERATOR && isComparisonOperator(tok.value) && getPrecedence(tok.value) >= precedence {
		// 開始建構連續比較
		ops := []string{}
		values := []cclNode{left} // 第一個值已經解析過了

		for p.current().typ == tOPERATOR && isComparisonOperator(p.current().value) && getPrecedence(p.current().value) >= precedence {
			ops = append(ops, p.current().value)
			p.advance()

			// 解析右側運算元
			nextValue, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			values = append(values, nextValue)

			// 檢查下一個標記是否還是比較運算符
			if p.current().typ != tOPERATOR || !isComparisonOperator(p.current().value) {
				break
			}
		}

		// 如果至少有兩個值和一個運算符，則創建連續比較節點
		if len(values) > 1 && len(ops) > 0 {
			// 如果只有一個運算符，則使用普通的二元運算節點
			if len(ops) == 1 && len(values) == 2 {
				return &cclBinaryOpNode{op: ops[0], left: values[0], right: values[1]}, nil
			}
			// 否則創建一個連續比較節點
			return &cclChainedComparisonNode{ops: ops, values: values}, nil
		}
	} else {
		// 常規的二元運算表達式處理
		for {
			tok := p.current()
			if (tok.typ != tOPERATOR && tok.typ != tDOT) || getPrecedence(tok.value) < precedence {
				break
			}
			op := tok.value
			p.advance()
			right, err := p.parseExpression(getPrecedence(op) + 1)
			if err != nil {
				return nil, err
			}
			left = &cclBinaryOpNode{op: op, left: left, right: right}
		}
	}

	return left, nil
}

// 檢查是否為比較運算符
func isComparisonOperator(op string) bool {
	switch op {
	case "<", ">", "<=", ">=", "==", "!=":
		return true
	default:
		return false
	}
}

func (p *parser) parsePrimary() (cclNode, error) {
	tok := p.current()
	switch tok.typ {
	case tNUMBER:
		p.advance()
		val, _ := strconv.ParseFloat(tok.value, 64)
		return &cclNumberNode{value: val}, nil
	case tSTRING:
		p.advance()
		return &cclStringNode{value: tok.value}, nil
	case tBOOLEAN:
		p.advance()
		val := tok.value == "true"
		return &cclBooleanNode{value: val}, nil
	case tNIL:
		p.advance()
		return &cclNilNode{}, nil
	case tAT:
		p.advance()
		return &cclAtNode{}, nil
	case tIDENT:
		name := tok.value
		p.advance()
		if p.current().typ == tLPAREN {
			p.advance()
			args := []cclNode{}
			for p.current().typ != tRPAREN && p.current().typ != tEOF {
				arg, err := p.parseExpression(0)
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
				if p.current().typ == tCOMMA {
					p.advance()
				}
			}
			if p.current().typ == tRPAREN {
				p.advance()
			}
			return &funcCallNode{name: name, args: args}, nil
		}
		return &cclIdentifierNode{name: name}, nil
	case tCOL_INDEX:
		// [A] 形式的欄位索引引用
		p.advance()
		return &cclColIndexNode{index: tok.value}, nil
	case tCOL_NAME:
		// ['colName'] 形式的欄位名稱引用
		p.advance()
		return &cclColNameNode{name: tok.value}, nil
	case tLPAREN:
		p.advance()
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if p.current().typ == tRPAREN {
			p.advance()
		}
		return expr, nil
	default:
		return nil, fmt.Errorf("unexpected token: %v", tok)
	}
}

func getPrecedence(op string) int {
	switch op {
	case "||": // 邏輯或優先級最低
		return 1
	case "&&": // 邏輯與優先級次低
		return 2
	case "=", "==", "!=", ">", "<", ">=", "<=":
		return 3
	case "&": // 字串連接，與加減同級
		return 4
	case "+", "-":
		return 4
	case "*", "/", "%":
		return 5
	case "^":
		return 6
	case ".":
		return 7
	default:
		return 0
	}
}

// CheckExpressionMode checks if tokens contain assignment syntax or NEW function.
// Expression mode (AddColUsingCCL, EditColByIndexUsingCCL, EditColByNameUsingCCL)
// does not allow assignment syntax or NEW function.
// Returns an error if such syntax is found.
func CheckExpressionMode(tokens []cclToken) error {
	for i, tok := range tokens {
		// 檢查賦值運算符
		if tok.typ == tASSIGN {
			return fmt.Errorf("CCL expression mode does not support assignment syntax (=). Use ExecuteCCL for statements with assignment")
		}
		// 檢查 NEW 函數
		if tok.typ == tIDENT && strings.ToUpper(tok.value) == "NEW" {
			// 確認後面是括號（確實是 NEW 函數調用）
			if i+1 < len(tokens) && tokens[i+1].typ == tLPAREN {
				return fmt.Errorf("CCL expression mode does not support NEW function. Use ExecuteCCL for creating new columns")
			}
		}
	}
	return nil
}
