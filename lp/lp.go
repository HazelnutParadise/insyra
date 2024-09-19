package lp

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// SolveLPWithGLPK solve lp file with glpk and set a timeout in seconds.
// Returns a DataTable with the parsed GLPK output.
func SolveLPWithGLPK(lpFile string, timeoutSeconds ...int) *insyra.DataTable {
	timeout := 0 * time.Second
	if len(timeoutSeconds) == 1 {
		timeout = time.Duration(timeoutSeconds[0]) * time.Second
	} else if len(timeoutSeconds) > 1 {
		insyra.LogWarning("SolveLPWithGLPK: Only one timeout can be set")
		return nil
	}

	// 創建一個帶有超時的 context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 使用 GLPK 命令行工具 glpsol 解決 LP 問題
	cmd := exec.CommandContext(ctx, "glpsol", "--lp", lpFile)
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		insyra.LogWarning("SolveLPWithGLPK: Command timed out after %d seconds", timeoutSeconds)
		fmt.Printf("GLPK Output (partial):\n%s", output)
		return nil
	}

	if err != nil {
		insyra.LogWarning("Failed to solve LP file with GLPK: %v\n", err)
		fmt.Printf("GLPK Output:\n%s", output)
		return nil
	}

	// 解析 GLPK 輸出並生成 DataTable
	dataTable := parseGLPKOutputToDataTable(string(output))
	return dataTable
}

// parseGLPKOutputToDataTable 解析 GLPK 輸出並將結果存入 DataTable
func parseGLPKOutputToDataTable(output string) *insyra.DataTable {
	parsedData := make(map[string]string)

	// 使用正則表達式來提取 GLPK 輸出的關鍵數據
	re := regexp.MustCompile(`\*\s*\d+:\s*obj\s*=\s*([-\d.]+).*`) // 解析目標函數值
	match := re.FindStringSubmatch(output)
	if len(match) > 1 {
		parsedData["Objective Value"] = match[1]
	}

	// 假設 x1、x2、x3 是變數，並且提取其對應的值
	reVars := regexp.MustCompile(`(\w\d)\s*=\s*([-\d.]+)`)
	varMatches := reVars.FindAllStringSubmatch(output, -1)
	for _, v := range varMatches {
		if len(v) == 3 {
			parsedData[v[1]] = v[2]
		}
	}

	// 將解析出的數據存入 DataTable
	return storeInDataTable(parsedData)
}

// storeInDataTable 將解析出的數據存入 DataTable
func storeInDataTable(data map[string]string) *insyra.DataTable {
	// 初始化一個 DataTable
	dataTable := insyra.NewDataTable()

	// 將解析出的結果添加為一列
	columnNames := []string{}
	values := []interface{}{}

	for name, value := range data {
		columnNames = append(columnNames, name)
		values = append(values, value)
	}

	// 創建一個 DataList，並將變數和值存入其中
	dataList := insyra.NewDataList(values...)
	dataTable.AppendColumns(dataList)

	return dataTable
}
