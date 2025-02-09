package lp

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/lpgen"
)

// SolveFromFile solves an LP file with GLPK and sets a timeout in seconds.
// Returns two DataTables: one with the parsed results and one with additional info.
func SolveFromFile(lpFile string, timeoutSeconds ...int) (*insyra.DataTable, *insyra.DataTable) {
	timeout := 0 * time.Second
	if len(timeoutSeconds) == 1 {
		timeout = time.Duration(timeoutSeconds[0]) * time.Second
	} else if len(timeoutSeconds) > 1 {
		insyra.LogWarning("SolveFromFile: Only one timeout can be set")
		return nil, nil
	}

	// Temporary file to store GLPK output
	tmpFile := "solution.txt"

	var ctx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		// Create context with timeout
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	// Use GLPK command-line tool to solve LP problem and output to a file
	cmd := exec.CommandContext(ctx, "glpsol", "--lp", lpFile, "--output", tmpFile)
	start := time.Now()
	output, err := cmd.CombinedOutput()
	executionTime := time.Since(start).Seconds()

	if ctx.Err() == context.DeadlineExceeded {
		insyra.LogWarning("SolveFromFile: Command timed out after %d seconds", timeoutSeconds)
		return nil, createAdditionalInfoDataTable("Timeout", executionTime, "", string(output), "", "")
	}

	if err != nil {
		insyra.LogWarning("Failed to solve LP file with GLPK: %v\n", err)
		return nil, createAdditionalInfoDataTable("Error", executionTime, err.Error(), string(output), "", "")
	}

	// Parse the solution file and store results in DataTables
	resultTable := parseGLPKOutputFromFile(tmpFile)
	iterations, nodes := extractIterationNodeCounts(string(output))

	additionalInfoTable := createAdditionalInfoDataTable("Success", executionTime, extractWarnings(output), string(output), iterations, nodes)

	// Clean up temporary file
	_ = os.Remove(tmpFile)

	return resultTable, additionalInfoTable
}

// SolveModel solves an LPModel directly by passing the model to GLPK without generating a model file.
// Returns two DataTables: one with the parsed results and one with additional info.
func SolveModel(model *lpgen.LPModel, timeoutSeconds ...int) (*insyra.DataTable, *insyra.DataTable) {
	var timeout time.Duration
	if len(timeoutSeconds) > 0 {
		timeout = time.Duration(timeoutSeconds[0]) * time.Second
	} else {
		timeout = 0
	}

	// 將 LPModel 轉換為 LP 格式的文本
	var lpBuffer bytes.Buffer
	lpBuffer.WriteString(model.ObjectiveType + "\n")
	lpBuffer.WriteString("  " + model.Objective + "\n")

	// 添加約束條件
	lpBuffer.WriteString("Subject To\n")
	for _, constr := range model.Constraints {
		lpBuffer.WriteString("  " + constr + "\n")
	}

	// 添加變數邊界
	if len(model.Bounds) > 0 {
		lpBuffer.WriteString("Bounds\n")
		for _, bound := range model.Bounds {
			lpBuffer.WriteString("  " + bound + "\n")
		}
	}

	// 添加整數變數
	if len(model.IntegerVars) > 0 {
		lpBuffer.WriteString("General\n")
		for _, intVar := range model.IntegerVars {
			lpBuffer.WriteString("  " + intVar + "\n")
		}
	}

	// 添加二進制變數
	if len(model.BinaryVars) > 0 {
		lpBuffer.WriteString("Binary\n")
		for _, binVar := range model.BinaryVars {
			lpBuffer.WriteString("  " + binVar + "\n")
		}
	}

	// 結尾
	lpBuffer.WriteString("End\n")

	// 設置上下文與超時
	var ctx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	// 創建臨時文件來存儲解決結果
	tmpFile, err := os.CreateTemp("", "solution-*.txt")
	if err != nil {
		insyra.LogFatal("lp.SolveModel: Failed to create temporary file for solution: %v", err)
		return nil, nil
	}
	defer os.Remove(tmpFile.Name()) // 確保在解決完成後刪除臨時文件

	// 使用 GLPK 直接從標準輸入解 LP 問題，並將結果輸出到臨時文件
	cmd := exec.CommandContext(ctx, "glpsol", "--lp", "/dev/stdin", "--output", tmpFile.Name())
	cmd.Stdin = &lpBuffer // 將 LPModel 的內容傳給標準輸入
	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	start := time.Now()
	err = cmd.Run()
	executionTime := time.Since(start).Seconds()

	// 處理 GLPK 執行錯誤
	if ctx.Err() == context.DeadlineExceeded {
		insyra.LogWarning("lp.SolveModel: Command timed out after %d seconds", timeoutSeconds)
		return nil, createAdditionalInfoDataTable("Timeout", executionTime, "", outputBuffer.String(), "", "")
	}

	if err != nil {
		insyra.LogWarning("lp.SolveModel: Failed to solve LP model with GLPK: %v\n", err)
		return nil, createAdditionalInfoDataTable("Error", executionTime, err.Error(), outputBuffer.String(), "", "")
	}

	// 解析 GLPK 的解決結果
	resultTable := parseGLPKOutputFromFile(tmpFile.Name())
	iterations, nodes := extractIterationNodeCounts(outputBuffer.String())

	additionalInfoTable := createAdditionalInfoDataTable("Success", executionTime, extractWarnings(outputBuffer.Bytes()), outputBuffer.String(), iterations, nodes)

	return resultTable, additionalInfoTable
}

// parseGLPKOutputFromFile parses the GLPK solution output from the given file.
func parseGLPKOutputFromFile(filePath string) *insyra.DataTable {
	dataTable := insyra.NewDataTable()

	// Open the file and read line by line
	file, err := os.Open(filePath)
	if err != nil {
		insyra.LogWarning("lp.parseGLPKOutputFromFile: Failed to open solution file: %v", err)
		return nil
	}
	defer file.Close()

	// Scan through each line and extract variable values
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.ReplaceAll(line, "Rows:", "Rows(Constraints):")
		line = strings.ReplaceAll(line, "Columns:", "Columns(Variables):")
		if line == "" {
			continue
		}

		// Create a DataList for each line and append as a row
		dataList := insyra.NewDataList(line)
		dataTable.AppendRowsFromDataList(dataList)
	}

	if err := scanner.Err(); err != nil {
		insyra.LogWarning("lp.parseGLPKOutputFromFile: Error reading solution file: %v", err)
	}

	return dataTable
}

// createAdditionalInfoDataTable stores additional info like execution time, status, and warnings
func createAdditionalInfoDataTable(status string, executionTime float64, warnings, fullOutput, iterations, nodes string) *insyra.DataTable {
	additionalInfo := map[string]any{
		"Status":         status,
		"Execution Time": fmt.Sprintf("%.2f seconds", executionTime),
		"Warnings":       warnings,
		"Full Output":    fullOutput,
		"Iterations":     iterations,
		"Nodes":          nodes,
	}

	dataTable := insyra.NewDataTable()
	rowNames := []string{}
	values := []any{}

	for name, value := range additionalInfo {
		rowNames = append(rowNames, name)
		values = append(values, value)
	}

	// Append results to a horizontal row
	rowNameDl := insyra.NewDataList(rowNames)
	dataList := insyra.NewDataList(values...).SetName("Additional Info")
	dataTable.AppendCols(rowNameDl, dataList)

	dataTable.SetColToRowNames("A")

	return dataTable
}

// extractIterationNodeCounts extracts iterations and node counts from the GLPK output file
func extractIterationNodeCounts(output string) (string, string) {
	iterations := ""
	nodes := ""

	iterRegex := regexp.MustCompile(`\*\s+(\d+):`)
	nodeRegex := regexp.MustCompile(`\+\s+(\d+):`)

	iterMatches := iterRegex.FindAllStringSubmatch(output, -1)
	nodeMatches := nodeRegex.FindAllStringSubmatch(output, -1)

	if len(iterMatches) > 0 {
		iterations = iterMatches[len(iterMatches)-1][1]
	}
	if len(nodeMatches) > 0 {
		nodes = nodeMatches[len(nodeMatches)-1][1]
	}

	return iterations, nodes
}

// extractWarnings extracts warnings from the output
func extractWarnings(output []byte) string {
	warnings := []string{}
	re := regexp.MustCompile(`warning:.*`)
	matches := re.FindAllString(string(output), -1)
	warnings = append(warnings, matches...)
	return strings.Join(warnings, "; ")
}
