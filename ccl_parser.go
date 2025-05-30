package insyra

import (
	"fmt"
	"strconv"
)

type parser struct {
	tokens []cclToken
	pos    int
}

func parse(tokens []cclToken) (cclNode, error) {
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

	return left, nil
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
