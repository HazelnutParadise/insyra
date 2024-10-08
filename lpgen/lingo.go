package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// 定義一個結構來存儲集合和變數
type Set struct {
	Name       string
	Range      []int
	Attributes []string
}

// 解析 SETS 部分
func parseSets(lingoText string) []Set {
	sets := []Set{}
	re := regexp.MustCompile(`(\w+)\s*/(\d+)\.\.(\d+)\s*/:(.+);`)
	matches := re.FindAllStringSubmatch(lingoText, -1)

	for _, match := range matches {
		setName := match[1]
		startRange := parseInt(match[2])
		endRange := parseInt(match[3])
		attributes := strings.Split(match[4], ",")

		set := Set{
			Name:       setName,
			Range:      generateRange(startRange, endRange),
			Attributes: trimSpaces(attributes),
		}
		sets = append(sets, set)
	}

	return sets
}

// 將數字範圍生成為 slice
func generateRange(start, end int) []int {
	var result []int
	for i := start; i <= end; i++ {
		result = append(result, i)
	}
	return result
}

// 去除屬性名稱中的空格
func trimSpaces(attributes []string) []string {
	for i := range attributes {
		attributes[i] = strings.TrimSpace(attributes[i])
	}
	return attributes
}

// 將字串轉換為整數
func parseInt(value string) int {
	var result int
	fmt.Sscanf(value, "%d", &result)
	return result
}

// 解析數據部分，適用於所有 Lingo 檔案
func parseData(lingoText string) map[string][]float64 {
	data := make(map[string][]float64)
	re := regexp.MustCompile(`(\w+)\s*=\s*([0-9.\s]+);`)
	matches := re.FindAllStringSubmatch(lingoText, -1)

	for _, match := range matches {
		varName := match[1]
		values := strings.Fields(match[2])
		var floatValues []float64

		for _, value := range values {
			if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
				floatValues = append(floatValues, floatVal)
			}
		}

		data[varName] = floatValues
	}

	return data
}

// 處理約束條件並轉換為 LP 格式的 Subject To 部分
func parseConstraints(lingoText string, data map[string][]float64) string {
	constraints := ""
	re := regexp.MustCompile(`@FOR\(\w+\((\w+)\):\s*(.+)\)`)
	matches := re.FindAllStringSubmatch(lingoText, -1)

	for _, match := range matches {
		index := match[1]
		expression := match[2]

		for i := 0; i < len(data["group_size"]); i++ {
			currentExpr := strings.Replace(expression, index, strconv.Itoa(i+1), -1)
			constraints += fmt.Sprintf("constraint%d: %s\n", i+1, currentExpr)
		}
	}
	return constraints
}

// 處理變數邊界條件並轉換為 LP 格式的 Bounds 部分
func parseBounds(lingoText string, data map[string][]float64) string {
	bounds := ""
	re := regexp.MustCompile(`@FOR\(\w+\((\w+)\):\s*(.+)\)`)
	matches := re.FindAllStringSubmatch(lingoText, -1)

	for _, match := range matches {
		index := match[1]
		expression := match[2]

		for i := 0; i < len(data["group_size"]); i++ {
			currentExpr := strings.Replace(expression, index, strconv.Itoa(i+1), -1)
			bounds += fmt.Sprintf("Bound%d: %s\n", i+1, currentExpr)
		}
	}
	return bounds
}

// 計算表達式的值，包括 @SUM 和 @pow 等運算
func evaluateExpression(expression string, data map[string][]float64) float64 {
	rePow := regexp.MustCompile(`@pow\(([^,]+),([^)]+)\)`)
	expression = rePow.ReplaceAllStringFunc(expression, func(match string) string {
		subMatches := rePow.FindStringSubmatch(match)
		base, _ := strconv.ParseFloat(subMatches[1], 64)
		exp, _ := strconv.ParseFloat(subMatches[2], 64)
		return fmt.Sprintf("%.4f", math.Pow(base, exp))
	})

	re := regexp.MustCompile(`(\w+)\((\d+)\)`)
	matches := re.FindAllStringSubmatch(expression, -1)

	result := 1.0
	for _, match := range matches {
		varName := match[1]
		index, _ := strconv.Atoi(match[2])
		if values, exists := data[varName]; exists && index-1 < len(values) {
			result *= values[index-1]
		}
	}
	return result
}

// 將解析的 Lingo 轉換為 LP 格式
func convertToLP(lingoText string, data map[string][]float64) {
	// LP 格式的目標函數
	objectiveFunction := parseSumExpression("@SUM(group(I): group_size(I) * vaccine_coverage(I))", data)
	fmt.Println("Minimize\n obj:", objectiveFunction)

	// LP 格式的約束條件
	constraints := parseConstraints(lingoText, data)
	fmt.Println("Subject To\n", constraints)

	// LP 格式的變數邊界條件
	bounds := parseBounds(lingoText, data)
	fmt.Println("Bounds\n", bounds)
}

// 解析 @SUM 公式，適用於目標函數
func parseSumExpression(expression string, data map[string][]float64) float64 {
	total := 0.0
	re := regexp.MustCompile(`@SUM\((\w+)\((\w+)\): (.+)\)`)
	matches := re.FindStringSubmatch(expression)

	if len(matches) > 0 {
		indexVar := matches[2]
		operation := matches[3]

		for i := 0; i < len(data["group_size"]); i++ {
			currentOp := strings.Replace(operation, indexVar, strconv.Itoa(i+1), -1)
			total += evaluateExpression(currentOp, data)
		}
	}
	return total
}

func main() {
	lingoText := `
SETS:
group /1..5/: group_size, vaccine_coverage, next_generation, eignvector_subgroup;
ENDSETS

DATA: 
   group_size = 77 241 375 204 103;
   vaccine_coverage = 0.1 0.2 0.3 0.4 0.5;
   next_generation = 1.1 1.2 2.1 2.2 2.3;
ENDDATA

@FOR(group(I):  
	@SUM(group(J): next_generation(I,J)*eignvector_subgroup(J)) <= 1
);
@FOR(group(I): vaccine_coverage(I) <= 1 );
@FOR(group(I): vaccine_coverage(I) >= 0 );
`

	// 解析數據
	data := parseData(lingoText)

	// 將 Lingo 轉換為 LP 格式
	convertToLP(lingoText, data)
}
