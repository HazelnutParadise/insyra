package ccl

import (
	"fmt"
	"strconv"
)

type parser struct {
	tokens []cclToken
	pos    int
}

func Parse(tokens []cclToken) (cclNode, error) {
	p := &parser{tokens: tokens}
	return p.parseExpression(0)
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
			if tok.typ != tOPERATOR || getPrecedence(tok.value) < precedence {
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
	case "=", "==", "!=", ">", "<", ">=", "<=":
		return 1
	case "+", "-":
		return 2
	case "*", "/", "%":
		return 3
	case "^":
		return 4
	default:
		return 0
	}
}
