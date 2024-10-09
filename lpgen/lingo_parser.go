package lpgen

import (
	"fmt"
	"strings"
)

// AST node definition
type Node struct {
	Type     string
	Value    string
	Children []*Node
}

// Parser structure definition
type Parser struct {
	tokens  []Token
	current int
}

// Construct a new Parser
func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens:  tokens,
		current: 0,
	}
}

// Get the current Token
func (p *Parser) currentToken() Token {
	if p.current < len(p.tokens) {
		return p.tokens[p.current]
	}
	return Token{} // 空的 Token 代表結束
}

func (p *Parser) nextToken() Token {
	if p.current < len(p.tokens)-1 {
		p.current++
	}
	return p.currentToken()
}

func (p *Parser) match(tokenType string) bool {
	if p.currentToken().Type == tokenType {
		p.nextToken() // Consume the token
		return true
	}
	return false
}

// parseExpression handles nested expressions and parentheses matching.
func (p *Parser) parseExpression() *Node {
	node := &Node{Type: "EXPRESSION"}

	for p.current < len(p.tokens) {
		currentToken := p.currentToken()

		switch currentToken.Type {
		case "VARIABLE":
			varNode := &Node{Type: "VARIABLE", Value: currentToken.Value}
			node.Children = append(node.Children, varNode)
			p.nextToken()

		case "NUMBER":
			numNode := &Node{Type: "NUMBER", Value: currentToken.Value}
			node.Children = append(node.Children, numNode)
			p.nextToken()

		case "OPERATOR":
			opNode := &Node{Type: "OPERATOR", Value: currentToken.Value}
			node.Children = append(node.Children, opNode)
			p.nextToken()

		case "SEPARATOR":
			if currentToken.Value == "(" {
				// Start parsing a nested expression
				p.nextToken()                     // Consume the opening parenthesis
				nestedExpr := p.parseExpression() // Recursively parse the nested expression
				node.Children = append(node.Children, nestedExpr)
			} else if currentToken.Value == ")" {
				// End of the current expression level
				p.nextToken() // Consume the closing parenthesis
				return node   // Return to the previous level of expression
			} else {
				// If it's another type of separator, just move on
				p.nextToken()
			}

		default:
			// Skip any unrecognized token types
			fmt.Printf("Skipping unrecognized token: Type=%s, Value=%s\n", currentToken.Type, currentToken.Value)
			p.nextToken()
		}
	}

	return node
}

// 解析 Assignment，並確保 token 逐步消耗
func (p *Parser) parseAssignment() *Node {
	assignNode := &Node{Type: "ASSIGNMENT"}

	if p.currentToken().Type == "VARIABLE" {
		varNode := &Node{Type: "VARIABLE", Value: p.currentToken().Value}
		assignNode.Children = append(assignNode.Children, varNode)
		p.nextToken() // Move to '='

		if p.match("OPERATOR") && p.currentToken().Value == "=" {
			p.nextToken() // Move to the first value

			valuesNode := &Node{Type: "VALUES"}
			for p.current < len(p.tokens) && p.currentToken().Type != "SEPARATOR" {
				if p.currentToken().Type == "NUMBER" {
					valueNode := &Node{Type: "NUMBER", Value: p.currentToken().Value}
					valuesNode.Children = append(valuesNode.Children, valueNode)
				}
				p.nextToken() // 正確消耗每個值 token
			}

			assignNode.Children = append(assignNode.Children, valuesNode)
			return assignNode
		}
	}

	return nil
}

// parseData 的改良版本，確保 token 消耗正確
func (p *Parser) parseData() *Node {
	node := &Node{Type: "DATA"}
	if p.match("KEYWORD") && p.currentToken().Value == "DATA" {
		p.nextToken() // Move to the next token

		for p.current < len(p.tokens) {
			if p.currentToken().Type == "KEYWORD" && p.currentToken().Value == "ENDDATA" {
				p.nextToken() // Consume ENDDATA
				break
			}

			if p.currentToken().Type == "VARIABLE" {
				assignmentNode := p.parseAssignment()
				if assignmentNode != nil {
					node.Children = append(node.Children, assignmentNode)
				}
			}

			p.nextToken() // 確保每個 token 都被正確消耗
		}
	}
	return node
}

// parseSets 的改良版本，確保消耗所有無效的 token
func (p *Parser) parseSets() *Node {
	node := &Node{Type: "SETS"}

	if p.match("KEYWORD") && p.currentToken().Value == "SETS" {
		p.nextToken() // Consume SETS keyword

		for p.current < len(p.tokens) {
			if p.currentToken().Type == "KEYWORD" && p.currentToken().Value == "ENDSETS" {
				p.nextToken() // Consume ENDSETS keyword
				break
			}

			if p.currentToken().Type == "VARIABLE" {
				varNode := &Node{Type: "VARIABLE", Value: p.currentToken().Value}
				node.Children = append(node.Children, varNode)
			} else if p.currentToken().Type == "OPERATOR" && p.currentToken().Value == "/" {
				rangeNode := p.parseRange()
				if rangeNode != nil {
					node.Children = append(node.Children, rangeNode)
				}
			}

			p.nextToken() // 移動到下一個 token
		}
	}

	return node
}

// parseRange handles parsing of ranges in the form a / .. / b.
func (p *Parser) parseRange() *Node {
	rangeNode := &Node{Type: "RANGE"}

	if p.match("OPERATOR") && p.currentToken().Value == "/" {
		start := p.currentToken()
		p.nextToken() // Move to the range separator
		if p.match("OPERATOR") && p.currentToken().Value == ".." {
			p.nextToken() // Move to the end of the range
			end := p.currentToken()
			p.nextToken() // Move past the end
			if p.match("OPERATOR") && p.currentToken().Value == "/" {
				rangeNode.Value = fmt.Sprintf("%s..%s", start.Value, end.Value)
				return rangeNode
			}
		}
	}

	return nil
}

func (p *Parser) parseSum() *Node {
	node := &Node{Type: "SUM"}

	if p.match("KEYWORD") && p.currentToken().Value == "@SUM" {
		p.nextToken() // Consume @SUM

		if p.match("SEPARATOR") && p.currentToken().Value == "(" {
			node.Children = append(node.Children, p.parseExpression())
			if p.currentToken().Value == ")" {
				p.nextToken() // Consume the closing parenthesis
			}
		}
	}
	return node
}

func (p *Parser) parseFor() *Node {
	node := &Node{Type: "FOR"}

	if p.match("KEYWORD") && p.currentToken().Value == "@FOR" {
		p.nextToken() // Consume @FOR

		if p.match("SEPARATOR") && p.currentToken().Value == "(" {
			node.Children = append(node.Children, p.parseExpression())
			if p.currentToken().Value == ")" {
				p.nextToken() // Consume the closing parenthesis
			}
		}
	}
	return node
}

func (p *Parser) parseBin() *Node {
	node := &Node{Type: "BIN"}

	if p.match("KEYWORD") && p.currentToken().Value == "@BIN" {
		p.nextToken() // Consume @BIN

		if p.match("SEPARATOR") && p.currentToken().Value == "(" {
			node.Children = append(node.Children, p.parseExpression())
			if p.currentToken().Value == ")" {
				p.nextToken() // Consume the closing parenthesis
			}
		}
	}
	return node
}

func (p *Parser) parsePow() *Node {
	node := &Node{Type: "POW"}

	if p.match("KEYWORD") && p.currentToken().Value == "@POW" {
		p.nextToken() // Consume @POW

		if p.match("SEPARATOR") && p.currentToken().Value == "(" {
			node.Children = append(node.Children, p.parseExpression())
			if p.currentToken().Value == ")" {
				p.nextToken() // Consume the closing parenthesis
			}
		}
	}
	return node
}

func (p *Parser) Parse() *Node {
	fmt.Println("Parse function started")
	root := &Node{Type: "PROGRAM"}

	for p.current < len(p.tokens) {
		currentToken := p.currentToken()
		fmt.Printf("Parsing token: Type=%s, Value=%s\n", currentToken.Type, currentToken.Value)

		switch currentToken.Type {
		case "KEYWORD":
			switch currentToken.Value {
			case "SETS":
				fmt.Println("Parsing SETS")
				root.Children = append(root.Children, p.parseSets())
			case "DATA":
				fmt.Println("Parsing DATA")
				root.Children = append(root.Children, p.parseData())
			case "@SUM":
				fmt.Println("Parsing @SUM")
				root.Children = append(root.Children, p.parseSum())
			case "@FOR":
				fmt.Println("Parsing @FOR")
				root.Children = append(root.Children, p.parseFor())
			case "@BIN":
				fmt.Println("Parsing @BIN")
				root.Children = append(root.Children, p.parseBin())
			case "@POW":
				fmt.Println("Parsing @POW")
				root.Children = append(root.Children, p.parsePow())
			default:
				fmt.Printf("Skipping unknown keyword: %s\n", currentToken.Value)
				p.nextToken()
			}
		case "SEPARATOR":
			if currentToken.Value == ";" {
				fmt.Println("Skipping semicolon")
				p.nextToken() // 確保跳過分號
			} else {
				fmt.Printf("Moving to next token for separator: %s\n", currentToken.Value)
				p.nextToken()
			}
		default:
			fmt.Printf("Skipping token: Type=%s, Value=%s\n", currentToken.Type, currentToken.Value)
			p.nextToken()
		}

		// 增加防止無限循環的邏輯，跳過無意義 token，並確保有意義的 token 被解析
		if p.currentToken().Type == "SEPARATOR" && p.currentToken().Value == ";" {
			continue // 繼續跳過不必要的分號
		}
	}

	fmt.Println("Parse function completed")
	return root
}

// Helper function to print AST for debugging purposes
func PrintAST(node *Node, level int) {
	indent := strings.Repeat("  ", level)
	fmt.Printf("%sNode Type: %s, Value: %s\n", indent, node.Type, node.Value)
	for _, child := range node.Children {
		PrintAST(child, level+1)
	}
}
