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
	if p.current+1 < len(p.tokens) { // 這裡要確保 p.current 不會超出範圍
		fmt.Printf("Moving to next token: %s\n", p.tokens[p.current].Value)
		p.current++
	}
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

// 解析 DATA 區塊的語法
func (p *Parser) parseData() *Node {
	node := &Node{Type: "DATA"}
	if p.match("KEYWORD") && p.currentToken().Value == "DATA" {
		// 處理數據
		p.nextToken() // 推進到下一個 token
		for p.currentToken().Value != "ENDDATA" {
			if p.match("VARIABLE") {
				varNode := &Node{Type: "VARIABLE", Value: p.currentToken().Value}
				node.Children = append(node.Children, varNode)
			}
			p.nextToken() // 確保每次迴圈中都推進
		}
	}
	p.nextToken() // 消費掉 ENDDATA
	return node
}

// 解析運算式
func (p *Parser) parseExpression() *Node {
	node := &Node{Type: "EXPRESSION"}
	fmt.Println("Starting expression parsing")

	for p.current < len(p.tokens) {
		switch p.currentToken().Type {
		case "VARIABLE":
			fmt.Printf("Found variable: %s\n", p.currentToken().Value)
			varNode := &Node{Type: "VARIABLE", Value: p.currentToken().Value}
			node.Children = append(node.Children, varNode)
		case "NUMBER":
			fmt.Printf("Found number: %s\n", p.currentToken().Value)
			numNode := &Node{Type: "NUMBER", Value: p.currentToken().Value}
			node.Children = append(node.Children, numNode)
		case "OPERATOR":
			fmt.Printf("Found operator: %s\n", p.currentToken().Value)
			opNode := &Node{Type: "OPERATOR", Value: p.currentToken().Value}
			node.Children = append(node.Children, opNode)
		case "SEPARATOR":
			if p.currentToken().Value == ")" {
				fmt.Println("End of expression found")
				return node
			}
		default:
			fmt.Printf("Unknown token in expression: %s\n", p.currentToken().Value)
		}
		p.nextToken() // 繼續處理
	}

	return node
}

func (p *Parser) parseSum() *Node {
	node := &Node{Type: "SUM"}
	fmt.Println("Starting @SUM parsing")
	if p.match("KEYWORD") && p.currentToken().Value == "@SUM" {
		p.nextToken() // 消費掉 @SUM

		// 處理括號內的內容
		if p.match("SEPARATOR") && p.currentToken().Value == "(" {
			fmt.Println("Parsing @SUM expression")
			node.Children = append(node.Children, p.parseExpression())
			p.nextToken() // 消費掉右括號
		}
	}
	return node
}

func (p *Parser) parseFor() *Node {
	node := &Node{Type: "FOR"}
	fmt.Println("Starting @FOR parsing")
	if p.match("KEYWORD") && p.currentToken().Value == "@FOR" {
		p.nextToken() // 消費掉 @FOR

		// 處理括號內的內容
		if p.match("SEPARATOR") && p.currentToken().Value == "(" {
			fmt.Println("Parsing @FOR expression")
			node.Children = append(node.Children, p.parseExpression())
			p.nextToken() // 消費掉右括號
		}
	}
	return node
}

func (p *Parser) Parse() *Node {
	root := &Node{Type: "PROGRAM"}

	for p.current < len(p.tokens) {
		switch p.currentToken().Type {
		case "KEYWORD":
			fmt.Printf("Parsing keyword: %s\n", p.currentToken().Value)
			if p.currentToken().Value == "SETS" {
				root.Children = append(root.Children, p.parseSets())
			} else if p.currentToken().Value == "DATA" {
				root.Children = append(root.Children, p.parseData())
			} else if p.currentToken().Value == "@SUM" {
				fmt.Println("Detected @SUM keyword")
				root.Children = append(root.Children, p.parseSum())
			} else if p.currentToken().Value == "@FOR" {
				fmt.Println("Detected @FOR keyword")
				root.Children = append(root.Children, p.parseFor())
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
