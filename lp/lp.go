package lp

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// SolveLPWithGLPK solve lp file with glpk and set a timeout in seconds.
// Returns two DataTables: one with the parsed results and one with additional info.
func SolveLPWithGLPK(lpFile string, timeoutSeconds ...int) (*insyra.DataTable, *insyra.DataTable) {
	timeout := 0 * time.Second
	if len(timeoutSeconds) == 1 {
		timeout = time.Duration(timeoutSeconds[0]) * time.Second
	} else if len(timeoutSeconds) > 1 {
		insyra.LogWarning("SolveLPWithGLPK: Only one timeout can be set")
		return nil, nil
	}

	// 創建一個帶有超時的 context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 使用 GLPK 命令行工具 glpsol 解決 LP 問題
	cmd := exec.CommandContext(ctx, "glpsol", "--lp", lpFile)
	start := time.Now()
	output, err := cmd.CombinedOutput()
	executionTime := time.Since(start).Seconds()

	// 將原本的輸出打印出來
	fmt.Printf("GLPK Output:\n%s", output)

	if ctx.Err() == context.DeadlineExceeded {
		insyra.LogWarning("SolveLPWithGLPK: Command timed out after %d seconds", timeoutSeconds)
		return nil, createAdditionalInfoDataTable("Timeout", executionTime, output)
	}

	if err != nil {
		insyra.LogWarning("Failed to solve LP file with GLPK: %v\n", err)
		return nil, createAdditionalInfoDataTable("Error", executionTime, output)
	}

	// 解析 GLPK 輸出並生成 DataTable
	resultTable := parseGLPKOutputToDataTable(string(output))
	additionalInfoTable := createAdditionalInfoDataTable("Success", executionTime, output)

	return resultTable, additionalInfoTable
}

// parseGLPKOutputToDataTable 解析 GLPK 輸出並將結果存入 DataTable
func parseGLPKOutputToDataTable(output string) *insyra.DataTable {
	parsedData := make(map[string]string)

	// 解析目標函數值
	reObjective := regexp.MustCompile(`\*\s*\d+:\s*obj\s*=\s*([-\d.]+).*`)
	matchObjective := reObjective.FindStringSubmatch(output)
	if len(matchObjective) > 1 {
		parsedData["Objective Value"] = matchObjective[1]
	}

	// 解析變數及其值
	reVars := regexp.MustCompile(`(\w\d+)\s*=\s*([-\d.]+)`)
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
	dataTable := insyra.NewDataTable()

	// 創建一個 DataList，並將變數和值存入其中
	for name, value := range data {
		dataList := insyra.NewDataList(value)
		dataList.SetName(name)
		dataTable.AppendColumns(dataList)
	}

	return dataTable
}

// createAdditionalInfoDataTable 創建存儲額外資訊的 DataTable
func createAdditionalInfoDataTable(status string, executionTime float64, output []byte) *insyra.DataTable {
	additionalInfo := map[string]interface{}{
		"Status":         status,
		"Execution Time": fmt.Sprintf("%.2f seconds", executionTime),
		"Warnings":       parseWarnings(output),
		"Full Output":    string(output), // 完整原始輸出
	}

	// 將額外信息存入 DataTable
	dataTable := insyra.NewDataTable()

	// 將額外資訊存入 DataList
	for name, value := range additionalInfo {
		dataList := insyra.NewDataList(value)
		dataList.SetName(name)
		dataTable.AppendColumns(dataList)
	}

	return dataTable
}

// parseWarnings 用於解析 GLPK 輸出中的警告訊息
func parseWarnings(output []byte) string {
	// 假設 GLPK 的警告訊息會包含 "warning"
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	warnings := []string{}

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "warning") {
			warnings = append(warnings, strings.TrimSpace(line))
		}
	}

	if len(warnings) == 0 {
		return "No warnings"
	}

	return strings.Join(warnings, "; ")
}
