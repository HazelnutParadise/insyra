package lp

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// SolveLPWithGLPK solves an LP file with GLPK and sets a timeout in seconds.
// Returns two DataTables: one with the parsed results and one with additional info.
func SolveLPWithGLPK(lpFile string, timeoutSeconds ...int) (*insyra.DataTable, *insyra.DataTable) {
	timeout := 0 * time.Second
	if len(timeoutSeconds) == 1 {
		timeout = time.Duration(timeoutSeconds[0]) * time.Second
	} else if len(timeoutSeconds) > 1 {
		insyra.LogWarning("SolveLPWithGLPK: Only one timeout can be set")
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
		insyra.LogWarning("SolveLPWithGLPK: Command timed out after %d seconds", timeoutSeconds)
		return nil, createAdditionalInfoDataTable("Timeout", executionTime, "", string(output), "", "")
	}

	if err != nil {
		insyra.LogWarning("Failed to solve LP file with GLPK: %v\n", err)
		return nil, createAdditionalInfoDataTable("Error", executionTime, err.Error(), string(output), "", "")
	}

	// Parse the solution file and store results in DataTables
	resultTable := parseGLPKOutputFromFile(tmpFile)
	iterations, nodes := extractIterationNodeCounts(tmpFile)
	additionalInfoTable := createAdditionalInfoDataTable("Success", executionTime, extractWarnings(output), string(output), iterations, nodes)

	// Clean up temporary file
	_ = os.Remove(tmpFile)

	return resultTable, additionalInfoTable
}

// parseGLPKOutputFromFile parses the GLPK solution output from the given file.
func parseGLPKOutputFromFile(filePath string) *insyra.DataTable {
	dataTable := insyra.NewDataTable()

	// Open the file and read line by line
	file, err := os.Open(filePath)
	if err != nil {
		insyra.LogWarning("Failed to open solution file: %v", err)
		return nil
	}
	defer file.Close()

	// Scan through each line and extract variable values
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Create a DataList for each line and append as a row
		dataList := insyra.NewDataList(line)
		dataTable.AppendRowsFromDataList(dataList)
	}

	if err := scanner.Err(); err != nil {
		insyra.LogWarning("Error reading solution file: %v", err)
	}

	return dataTable
}

// createAdditionalInfoDataTable stores additional info like execution time, status, and warnings
func createAdditionalInfoDataTable(status string, executionTime float64, warnings, fullOutput, iterations, nodes string) *insyra.DataTable {
	additionalInfo := map[string]interface{}{
		"Status":         status,
		"Execution Time": fmt.Sprintf("%.2f seconds", executionTime),
		"Warnings":       warnings,
		"Full Output":    fullOutput,
		"Iterations":     iterations,
		"Nodes":          nodes,
	}

	dataTable := insyra.NewDataTable()
	rowNames := []string{}
	values := []interface{}{}

	for name, value := range additionalInfo {
		rowNames = append(rowNames, name)
		values = append(values, value)
	}

	// Append results to a horizontal row
	rowNameDl := insyra.NewDataList(rowNames)
	dataList := insyra.NewDataList(values...).SetName("Additional Info")
	dataTable.AppendColumns(rowNameDl, dataList)

	dataTable.SetColumnToRowNames("A")

	return dataTable
}

// extractIterationNodeCounts extracts iterations and node counts from the GLPK output file
func extractIterationNodeCounts(filePath string) (string, string) {
	iterations := ""
	nodes := ""

	// Open the GLPK output file and extract iteration and node counts
	file, err := os.Open(filePath)
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)

		reIter := regexp.MustCompile(`\*\s*\d+:\s*obj\s*=\s*[-\d.]+\s*inf\s*=\s*[-\d.]+\s*\((\d+)\)`)
		reNode := regexp.MustCompile(`\+\s*\d+:\s*mip\s*=\s*[-\d.]+\s*<=\s*[-\d.]+\s*\d+.\d+%\s*\((\d+);\s*(\d+)\)`)

		for scanner.Scan() {
			line := scanner.Text()

			// Extract iteration counts
			if match := reIter.FindStringSubmatch(line); len(match) > 1 {
				iterations = match[1]
			}

			// Extract node counts
			if match := reNode.FindStringSubmatch(line); len(match) > 1 {
				nodes = match[1]
			}
		}
	}

	return iterations, nodes
}

// extractWarnings extracts warnings from the output
func extractWarnings(output []byte) string {
	warnings := []string{}
	re := regexp.MustCompile(`warning:.*`)
	matches := re.FindAllString(string(output), -1)
	for _, match := range matches {
		warnings = append(warnings, match)
	}
	return strings.Join(warnings, "; ")
}
