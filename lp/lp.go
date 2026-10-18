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
	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/HazelnutParadise/insyra/lpgen"
)

// SolveFromFile solves an LP file with GLPK and sets a timeout in seconds.
// Returns two DataTables: one with the parsed results and one with additional info.
//
// Deprecated: use SolveFile for a Solution and error. SolveFile does not
// automatically install GLPK and its return type differs. Removed in 0.4.
func SolveFromFile(lpFile string, timeoutSeconds ...int) (*insyra.DataTable, *insyra.DataTable) {
	// Check the arguments before initGLPK: a call that is already wrong is no
	// reason to go looking for — or install — a solver.
	if len(timeoutSeconds) > 1 {
		return refusedResult("SolveFromFile", "only one timeout can be set")
	}

	initGLPK()
	timeout := 0 * time.Second
	if len(timeoutSeconds) == 1 {
		timeout = time.Duration(timeoutSeconds[0]) * time.Second
	}

	// Unique temporary file for GLPK output. A fixed "solution.txt" in the CWD
	// collided between concurrent calls and was left behind on error/timeout.
	solFile, err := os.CreateTemp("", "lp-solution-*.txt")
	if err != nil {
		return refusedResult("SolveFromFile", "failed to create temporary solution file: %v", err)
	}
	tmpFile := solFile.Name()
	_ = solFile.Close()
	defer func() { _ = os.Remove(tmpFile) }()

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
	utils.ApplyHideWindow(cmd)
	start := time.Now()
	output, err := cmd.CombinedOutput()
	executionTime := time.Since(start).Seconds()

	if ctx.Err() == context.DeadlineExceeded {
		return noSolution("SolveFromFile", "command timed out"), createAdditionalInfoDataTable("Timeout", executionTime, "", string(output), "", "")
	}

	if err != nil {
		return noSolution("SolveFromFile", "glpsol failed: %v", err), createAdditionalInfoDataTable("Error", executionTime, err.Error(), string(output), "", "")
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
//
// Deprecated: use Solve for a Solution and error. Solve does not automatically
// install GLPK and its return type differs. Removed in 0.4.
func SolveModel(model *lpgen.LPModel, timeoutSeconds ...int) (*insyra.DataTable, *insyra.DataTable) {
	initGLPK()
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
		return refusedResult("SolveModel", "failed to create temporary file for solution: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }() // 確保在解決完成後刪除臨時文件

	// 使用 GLPK 直接從標準輸入解 LP 問題，並將結果輸出到臨時文件
	// 將 LP 內容寫入真實臨時檔再交給 glpsol。原本用 "/dev/stdin" 在 Windows
	// 上不是有效路徑，會導致 glpsol 失敗、SolveModel 只回傳錯誤資訊表。
	lpFile, err := os.CreateTemp("", "model-*.lp")
	if err != nil {
		return refusedResult("SolveModel", "failed to create temporary LP file: %v", err)
	}
	defer func() { _ = os.Remove(lpFile.Name()) }()
	if _, werr := lpFile.Write(lpBuffer.Bytes()); werr != nil {
		_ = lpFile.Close()
		return noSolution("SolveModel", "failed to write LP file: %v", werr), createAdditionalInfoDataTable("Error", 0, werr.Error(), "", "", "")
	}
	_ = lpFile.Close()

	cmd := exec.CommandContext(ctx, "glpsol", "--lp", lpFile.Name(), "--output", tmpFile.Name())
	utils.ApplyHideWindow(cmd)
	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

	start := time.Now()
	err = cmd.Run()
	executionTime := time.Since(start).Seconds()

	// 處理 GLPK 執行錯誤
	if ctx.Err() == context.DeadlineExceeded {
		return noSolution("SolveModel", "command timed out"), createAdditionalInfoDataTable("Timeout", executionTime, "", outputBuffer.String(), "", "")
	}

	if err != nil {
		return noSolution("SolveModel", "glpsol failed: %v", err), createAdditionalInfoDataTable("Error", executionTime, err.Error(), outputBuffer.String(), "", "")
	}

	// 解析 GLPK 的解決結果
	resultTable := parseGLPKOutputFromFile(tmpFile.Name())
	iterations, nodes := extractIterationNodeCounts(outputBuffer.String())

	additionalInfoTable := createAdditionalInfoDataTable("Success", executionTime, extractWarnings(outputBuffer.Bytes()), outputBuffer.String(), iterations, nodes)

	return resultTable, additionalInfoTable
}

// noSolution is the result table of a solve that produced no solution: empty,
// never nil, so a caller that skips the check can still call methods on it, with
// the reason on its Err().
func noSolution(funcName, reason string, args ...any) *insyra.DataTable {
	return insyra.NewDataTable().SetErr("lp", funcName, reason, args...)
}

// refusedResult is the pair returned when the call never reached the solver:
// an empty solution table carrying the reason, and an info table whose Status is
// Error and whose Warnings row holds the same reason.
func refusedResult(funcName, reason string, args ...any) (*insyra.DataTable, *insyra.DataTable) {
	return noSolution(funcName, reason, args...),
		createAdditionalInfoDataTable("Error", 0, fmt.Sprintf(reason, args...), "", "", "")
}

// parseGLPKOutputFromFile parses the GLPK solution output from the given file.
func parseGLPKOutputFromFile(filePath string) *insyra.DataTable {
	dataTable := insyra.NewDataTable()

	// Open the file and read line by line
	file, err := os.Open(filePath)
	if err != nil {
		return noSolution("parseGLPKOutputFromFile", "failed to open solution file: %v", err)
	}
	defer func() { _ = file.Close() }()

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
		insyra.LogWarning("lp", "parseGLPKOutputFromFile", "Error reading solution file: %v", err)
	}

	return dataTable
}

// createAdditionalInfoDataTable stores additional info like execution time, status, and warnings
func createAdditionalInfoDataTable(status string, executionTime float64, warnings, fullOutput, iterations, nodes string) *insyra.DataTable {
	// Fixed order: a map here made the row order differ between runs.
	rowNames := []string{"Status", "Execution Time", "Warnings", "Full Output", "Iterations", "Nodes"}
	values := []any{status, fmt.Sprintf("%.2f seconds", executionTime), warnings, fullOutput, iterations, nodes}

	dataTable := insyra.NewDataTable()

	// Append results to a horizontal row
	rowNameDl := insyra.NewDataList(rowNames)
	dataList := insyra.NewDataList(values...).SetName("Additional Info")
	dataTable.AppendCols(rowNameDl, dataList)

	dataTable.SetColToRowNames("A")

	return dataTable
}

// Compiled once: these ran through regexp.MustCompile on every call.
var (
	lpIterRe    = regexp.MustCompile(`\*\s+(\d+):`)
	lpNodeRe    = regexp.MustCompile(`\+\s+(\d+):`)
	lpWarningRe = regexp.MustCompile(`warning:.*`)
)

// extractIterationNodeCounts extracts iterations and node counts from the GLPK output file
func extractIterationNodeCounts(output string) (string, string) {
	iterations := ""
	nodes := ""

	iterMatches := lpIterRe.FindAllStringSubmatch(output, -1)
	nodeMatches := lpNodeRe.FindAllStringSubmatch(output, -1)

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
	matches := lpWarningRe.FindAllString(string(output), -1)
	warnings = append(warnings, matches...)
	return strings.Join(warnings, "; ")
}
