package lpgen

import (
	"fmt"
)

// AST 節點定義
type Node struct {
	Type     string
	Value    string
	Children []*Node
}

// 定義 Parser 結構
type Parser struct {
	tokens  []Token
	current int
}

// 構造一個新的 Parser
func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens:  tokens,
		current: 0,
	}
}

// 獲取當前的 Token
func (p *Parser) currentToken() Token {
	if p.current < len(p.tokens) {
		return p.tokens[p.current]
	}
	return Token{} // 空 Token 表示結束
}

func (p *Parser) match(tokenType string) bool {
	fmt.Printf("Trying to match %s, current token: %s\n", tokenType, p.currentToken().Type)
	if p.currentToken().Type == tokenType {
		fmt.Println("Match success!")
		p.nextToken() // 消費掉該 Token
		return true
	}
	fmt.Println("Match failed!")
	return false
}

func (p *Parser) nextToken() Token {
	if p.current < len(p.tokens)-1 { // 防止超過範圍
		p.current++
	}
	fmt.Printf("Moving to next token: %s\n", p.currentToken().Value)
	return p.currentToken()
}

func (p *Parser) parseSets() *Node {
	node := &Node{Type: "SETS"}
	fmt.Println("Starting SETS parsing")

	// 只在關鍵字為 SETS 的時候解析
	if p.currentToken().Type == "KEYWORD" && p.currentToken().Value == "SETS" {
		fmt.Println("Matched SETS keyword")
		p.nextToken() // 消費掉 SETS

		for p.current < len(p.tokens) {
			currentToken := p.currentToken()
			fmt.Printf("Current token: Type = %s, Value = %s\n", currentToken.Type, currentToken.Value)

			// 檢查是否到達 ENDSETS
			if currentToken.Type == "KEYWORD" && currentToken.Value == "ENDSETS" {
				fmt.Println("Found ENDSETS, finishing SETS parsing")
				p.nextToken() // 消費掉 "ENDSETS"
				break         // 結束解析
			}

			// 解析變數名稱
			if currentToken.Type == "VARIABLE" {
				fmt.Printf("Found variable: %s\n", currentToken.Value)
				varNode := &Node{Type: "VARIABLE", Value: currentToken.Value}
				node.Children = append(node.Children, varNode)
			}

			// 處理運算符或其他符號，跳過分號等不重要符號
			if currentToken.Type == "SEPARATOR" && currentToken.Value == ";" {
				fmt.Println("Found semicolon, skipping")
			}

			// 推進到下一個 token
			p.nextToken()
		}
	} else {
		fmt.Printf("SETS keyword not matched: current token type = %s, value = %s\n", p.currentToken().Type, p.currentToken().Value)
	}

	fmt.Println("Finished SETS parsing")
	return node
}

func (p *Parser) parseData() *Node {
	node := &Node{Type: "DATA"}
	if p.match("KEYWORD") && p.currentToken().Value == "DATA" {
		p.nextToken() // 推進到下一個 token
		for {
			// 檢查是否到達 ENDDATA
			if p.currentToken().Value == "ENDDATA" {
				fmt.Println("Found ENDDATA, exiting data parsing")
				p.nextToken() // 消費掉 ENDDATA
				break         // 跳出迴圈
			}

			if p.match("VARIABLE") {
				varNode := &Node{Type: "VARIABLE", Value: p.currentToken().Value}
				node.Children = append(node.Children, varNode)
			}
			p.nextToken() // 確保每次迴圈中都推進
		}
	}
	return node
}

func (p *Parser) parseExpression() *Node {
	node := &Node{Type: "EXPRESSION"}
	fmt.Println("Starting expression parsing")

	for p.current < len(p.tokens) {
		currentToken := p.currentToken()

		switch currentToken.Type {
		case "VARIABLE":
			fmt.Printf("Found variable: %s\n", currentToken.Value)
			varNode := &Node{Type: "VARIABLE", Value: currentToken.Value}
			node.Children = append(node.Children, varNode)
		case "NUMBER":
			fmt.Printf("Found number: %s\n", currentToken.Value)
			numNode := &Node{Type: "NUMBER", Value: currentToken.Value}
			node.Children = append(node.Children, numNode)
		case "OPERATOR":
			fmt.Printf("Found operator: %s\n", currentToken.Value)
			opNode := &Node{Type: "OPERATOR", Value: currentToken.Value}
			node.Children = append(node.Children, opNode)
		case "SEPARATOR":
			if currentToken.Value == "(" {
				// 遇到左括號，進入遞迴處理嵌套表達式
				fmt.Println("Found left parenthesis, parsing nested expression")
				p.nextToken()                     // 消費掉左括號
				nestedExpr := p.parseExpression() // 遞迴解析括號內的內容
				node.Children = append(node.Children, nestedExpr)
			} else if currentToken.Value == ")" {
				// 遇到右括號，結束當前表達式
				fmt.Println("Found right parenthesis, ending expression parsing")
				p.nextToken() // 消費掉右括號
				return node
			}
		default:
			fmt.Printf("Unknown token in expression: %s\n", currentToken.Value)
		}

		p.nextToken() // 繼續處理下一個 token
	}

	return node
}

func (p *Parser) parseSum() *Node {
	node := &Node{Type: "SUM"}
	fmt.Println("Starting @SUM parsing")

	if p.match("KEYWORD") && p.currentToken().Value == "@SUM" {
		p.nextToken() // 消費掉 @SUM

		// 處理括號內的表達式
		if p.match("SEPARATOR") && p.currentToken().Value == "(" {
			fmt.Println("Parsing @SUM expression")
			node.Children = append(node.Children, p.parseExpression())
			if p.currentToken().Value == ")" {
				p.nextToken() // 消費掉右括號
			} else {
				fmt.Println("Error: Missing closing parenthesis in @SUM")
			}
		}
	}
	return node
}

func (p *Parser) parseFor() *Node {
	node := &Node{Type: "FOR"}
	fmt.Println("Starting @FOR parsing")

	if p.match("KEYWORD") && p.currentToken().Value == "@FOR" {
		p.nextToken() // 消費掉 @FOR

		// 處理括號內的表達式
		if p.match("SEPARATOR") && p.currentToken().Value == "(" {
			fmt.Println("Parsing @FOR expression")
			node.Children = append(node.Children, p.parseExpression())
			if p.currentToken().Value == ")" {
				p.nextToken() // 消費掉右括號
			}
		}
	}
	return node
}

// 解析 @BIN 語法
func (p *Parser) parseBin() *Node {
	node := &Node{Type: "BIN"}
	fmt.Println("Starting @BIN parsing")

	if p.match("KEYWORD") && p.currentToken().Value == "@BIN" {
		p.nextToken() // 消費掉 @BIN

		// 處理括號內的變量
		if p.match("SEPARATOR") && p.currentToken().Value == "(" {
			fmt.Println("Parsing @BIN expression")
			node.Children = append(node.Children, p.parseExpression())
			p.nextToken() // 消費掉右括號
		}
	}
	return node
}

// 解析 @POW 語法
func (p *Parser) parsePow() *Node {
	node := &Node{Type: "POW"}
	fmt.Println("Starting @POW parsing")

	if p.match("KEYWORD") && p.currentToken().Value == "@POW" {
		p.nextToken() // 消費掉 @POW

		// 處理括號內的運算
		if p.match("SEPARATOR") && p.currentToken().Value == "(" {
			fmt.Println("Parsing @POW expression")
			node.Children = append(node.Children, p.parseExpression())
			p.nextToken() // 消費掉右括號
		}
	}
	return node
}

func (p *Parser) Parse() *Node {
	root := &Node{Type: "PROGRAM"}

	for p.current < len(p.tokens) {
		currentToken := p.currentToken()

		switch currentToken.Type {
		case "KEYWORD":
			if currentToken.Value == "SETS" {
				root.Children = append(root.Children, p.parseSets())
			} else if currentToken.Value == "DATA" {
				root.Children = append(root.Children, p.parseData())
			} else if currentToken.Value == "@SUM" {
				root.Children = append(root.Children, p.parseSum())
			} else if currentToken.Value == "@FOR" {
				root.Children = append(root.Children, p.parseFor())
			} else if currentToken.Value == "@BIN" {
				root.Children = append(root.Children, p.parseBin())
			} else if currentToken.Value == "@POW" {
				root.Children = append(root.Children, p.parsePow())
			} else if currentToken.Value == "ENDDATA" {
				fmt.Println("Skipping ENDDATA keyword")
				p.nextToken() // 消費掉 ENDDATA
			} else {
				fmt.Println("Unknown keyword, skipping...")
			}
		case "SEPARATOR":
			if currentToken.Value == ";" {
				fmt.Println("Skipping semicolon")
				p.nextToken() // 跳過分號
			}
		default:
			p.nextToken() // 繼續讀取下一個 Token
		}
	}
	return root
}

// 打印 AST 的輔助函數
func PrintAST(node *Node, level int) {
	for i := 0; i < level; i++ {
		fmt.Print("  ")
	}
	fmt.Printf("Node Type: %s, Value: %s\n", node.Type, node.Value)
	for _, child := range node.Children {
		PrintAST(child, level+1)
	}
}
