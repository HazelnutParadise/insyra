package lp

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize/convex/lp"
)

func SolveFromFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		insyra.LogWarning("Error reading LP file: %v\n", err)
		return
	}
	defer file.Close()

	var objFn []float64
	var constraints [][]float64
	var rhs []float64
	var lowerBounds, upperBounds []float64
	var generalVars, binaryVars []string
	variableMap := make(map[string]int)
	constraintLabels := make(map[string]bool) // 新增：用於存儲約束標籤

	scanner := bufio.NewScanner(file)
	section := ""
	objectiveLine := ""
	isMinimize := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.ToLower(line) == "end" {
			continue
		}

		switch {
		case strings.HasPrefix(strings.ToLower(line), "minimize"):
			isMinimize = true
			section = "objective"
			objectiveLine = strings.TrimPrefix(strings.ToLower(line), "minimize")
		case strings.HasPrefix(strings.ToLower(line), "maximize") || strings.HasPrefix(line, "obj:"):
			section = "objective"
			objectiveLine = strings.TrimPrefix(strings.ToLower(line), "obj:")
		case strings.HasPrefix(strings.ToLower(line), "subject to"):
			section = "constraints"
		case strings.HasPrefix(strings.ToLower(line), "bounds"):
			section = "bounds"
		case strings.HasPrefix(strings.ToLower(line), "general"):
			section = "general"
		case strings.HasPrefix(strings.ToLower(line), "binary"):
			section = "binary"
		default:
			switch section {
			case "objective":
				objectiveLine += " " + line
			case "constraints":
				// 提取並記錄約束標籤
				colonIndex := strings.Index(line, ":")
				if colonIndex != -1 {
					label := strings.TrimSpace(line[:colonIndex])
					constraintLabels[label] = true
					line = strings.TrimSpace(line[colonIndex+1:])
				}

				coef, rhsValue := parseConstraint(line, variableMap)
				constraints = append(constraints, coef)
				rhs = append(rhs, rhsValue)
			case "bounds":
				parseBounds(line, variableMap, &lowerBounds, &upperBounds, constraintLabels) // 傳遞 constraintLabels
			case "general":
				generalVars = append(generalVars, strings.Fields(line)...)
			case "binary":
				binaryVars = append(binaryVars, strings.Fields(line)...)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		insyra.LogWarning("Error scanning LP file: %v\n", err)
		return
	}

	objFn = parseObjective(objectiveLine, variableMap, isMinimize)

	// 在解析完所有數據後，清理目標函數和約束
	objFn, variableMap = cleanProblem(objFn, constraints, variableMap)

	// 確保所有變量都有合理的邊界
	for i := 0; i < len(objFn); i++ {
		if i >= len(lowerBounds) {
			lowerBounds = append(lowerBounds, 0)
		}
		if i >= len(upperBounds) {
			upperBounds = append(upperBounds, 1e6)
		}
	}

	numVars := len(objFn)
	numConstraints := len(constraints)

	if numConstraints == 0 || numVars == 0 {
		insyra.LogWarning("Error: no valid constraints or variables found.")
		return
	}

	// 確保所有約束行具有正確的長度
	for i, row := range constraints {
		if len(row) != numVars {
			newRow := make([]float64, numVars)
			copy(newRow, row[:min(len(row), numVars)])
			constraints[i] = newRow
		}
	}

	A := mat.NewDense(numConstraints, numVars, nil)
	for i, row := range constraints {
		A.SetRow(i, row)
	}

	// Print problem information
	fmt.Printf("Number of variables: %d\n", numVars)
	fmt.Printf("Number of constraints: %d\n", numConstraints)
	fmt.Printf("Objective function: %v\n", objFn)
	fmt.Println("Constraint matrix A:")
	matPrint(A)
	fmt.Printf("RHS vector b: %v\n", rhs)

	// Add this logging
	fmt.Println("Variable bounds:")
	for varName, index := range variableMap {
		// 如果變量是約束標籤，則跳過
		if constraintLabels[varName] {
			continue
		}
		lower := "0"
		upper := "+inf"
		if index < len(lowerBounds) {
			lower = fmt.Sprintf("%.2f", lowerBounds[index])
		}
		if index < len(upperBounds) && upperBounds[index] < 1e100 {
			upper = fmt.Sprintf("%.2f", upperBounds[index])
		}
		fmt.Printf("%s: [%s, %s]\n", varName, lower, upper)
	}

	// Convert the problem to standard form
	c, A_standard, b_standard := lp.Convert(objFn, A, rhs, nil, nil)

	// Check for unboundedness before solving
	if checkUnboundedness(c, A_standard, b_standard) {
		insyra.LogWarning("The problem may be unbounded. Consider adding constraints or adjusting the objective function.")
		return
	}

	// Print converted problem information
	fmt.Println("Converted problem:")
	fmt.Printf("c: %v\n", c)
	fmt.Println("A_standard:")
	matPrint(A_standard)
	fmt.Printf("b_standard: %v\n", b_standard)

	// Solve the converted problem
	objectiveValue, solution, err := lp.Simplex(c, A_standard, b_standard, 0, nil)
	if err != nil {
		if err == lp.ErrUnbounded {
			insyra.LogWarning("The problem is unbounded. Consider adding constraints or adjusting the objective function.")
		} else {
			insyra.LogWarning("Simplex optimization failed: %v\n", err)
		}
		return
	}

	insyra.LogInfo("Optimization successful.")
	if isMinimize {
		objectiveValue = -objectiveValue
	}
	fmt.Printf("Objective value: %f\n", objectiveValue)

	for varName, index := range variableMap {
		// 如果變量是約束標籤，則跳過
		if constraintLabels[varName] {
			continue
		}
		if index < len(solution) {
			fmt.Printf("%s: %f\n", varName, solution[index])
		} else {
			fmt.Printf("%s: Not used in optimization\n", varName)
		}
	}
}

func parseObjective(line string, variableMap map[string]int, isMinimize bool) []float64 {
	re := regexp.MustCompile(`([-+]?\d*\.?\d+)?\s*([a-zA-Z]\d*)`)
	terms := re.FindAllStringSubmatch(line, -1)
	objFn := make([]float64, 0)

	for _, match := range terms {
		coefStr := match[1]
		varName := match[2]

		if coefStr == "" {
			coefStr = "1"
		} else if coefStr == "-" {
			coefStr = "-1"
		}

		coef, _ := strconv.ParseFloat(coefStr, 64)

		if _, exists := variableMap[varName]; !exists {
			variableMap[varName] = len(variableMap)
		}

		index := variableMap[varName]
		for len(objFn) <= index {
			objFn = append(objFn, 0)
		}
		objFn[index] = coef
	}

	if isMinimize {
		for i := range objFn {
			objFn[i] = -objFn[i]
		}
	}

	return objFn
}

func parseConstraint(line string, variableMap map[string]int) ([]float64, float64) {
	coefs := make([]float64, 0)
	re := regexp.MustCompile(`([-+]?(?:\d*\.)?\d+)?\s*([a-zA-Z]\d*)`)
	matches := re.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		coefStr := match[1]
		varName := match[2]

		if coefStr == "" {
			coefStr = "1"
		} else if coefStr == "-" {
			coefStr = "-1"
		}

		coef, _ := strconv.ParseFloat(coefStr, 64)

		if _, exists := variableMap[varName]; !exists {
			variableMap[varName] = len(variableMap)
		}

		index := variableMap[varName]
		for len(coefs) <= index {
			coefs = append(coefs, 0)
		}
		coefs[index] = coef
	}

	rhsRegex := regexp.MustCompile(`(<=|>=)\s*([-+]?(?:\d*\.)?\d+)`)
	rhsMatch := rhsRegex.FindStringSubmatch(line)
	var rhs float64
	if len(rhsMatch) > 2 {
		rhs, _ = strconv.ParseFloat(rhsMatch[2], 64)
		if rhsMatch[1] == ">=" {
			for i := range coefs {
				coefs[i] = -coefs[i]
			}
			rhs = -rhs
		}
	}

	return coefs, rhs
}

func parseBounds(line string, variableMap map[string]int, lowerBounds, upperBounds *[]float64, constraintLabels map[string]bool) {
	parts := strings.Fields(line)
	if len(parts) != 5 {
		return
	}

	varName := parts[2]
	// 如果變量名稱是約束標籤，則跳過
	if constraintLabels[varName] {
		return
	}

	if _, exists := variableMap[varName]; !exists {
		variableMap[varName] = len(variableMap)
	}

	index := variableMap[varName]
	for len(*lowerBounds) <= index {
		*lowerBounds = append(*lowerBounds, 0)
	}
	for len(*upperBounds) <= index {
		*upperBounds = append(*upperBounds, 1e6) // 使用一個較大但有限的數作為默認上界
	}

	lower, _ := strconv.ParseFloat(parts[0], 64)
	upper, _ := strconv.ParseFloat(parts[4], 64)

	(*lowerBounds)[index] = lower
	(*upperBounds)[index] = upper
}

func allZeros(slice []float64) bool {
	for _, v := range slice {
		if v != 0 {
			return false
		}
	}
	return true
}

// Helper function to append a row to a matrix
func appendRow(m *mat.Dense, row []float64) *mat.Dense {
	r, c := m.Dims()
	newM := mat.NewDense(r+1, c, nil)
	newM.Copy(m)
	newM.SetRow(r, row)
	return newM
}

// Helper function to print a matrix
func matPrint(X mat.Matrix) {
	fa := mat.Formatted(X, mat.Prefix(""), mat.Squeeze())
	fmt.Printf("%v\n", fa)
}

func checkUnboundedness(c []float64, A *mat.Dense, b []float64) bool {
	m, n := A.Dims()
	for j := 0; j < n; j++ {
		if c[j] < 0 {
			allNonPositive := true
			for i := 0; i < m; i++ {
				if A.At(i, j) > 0 {
					allNonPositive = false
					break
				}
			}
			if allNonPositive {
				return true
			}
		}
	}
	return false
}

// 新增這個函數來清理問題
func cleanProblem(objFn []float64, constraints [][]float64, variableMap map[string]int) ([]float64, map[string]int) {
	usedVars := make(map[int]bool)

	// 檢查目標函數中使用的變量
	for i, coef := range objFn {
		if coef != 0 {
			usedVars[i] = true
		}
	}

	// 檢查約束中使用的變量
	for _, row := range constraints {
		for i, coef := range row {
			if coef != 0 {
				usedVars[i] = true
			}
		}
	}

	// 創建新的目標函數和變量映射
	newObjFn := []float64{}
	newVariableMap := make(map[string]int)
	oldToNewIndex := make(map[int]int)

	for varName, oldIndex := range variableMap {
		if usedVars[oldIndex] {
			newIndex := len(newVariableMap)
			newVariableMap[varName] = newIndex
			oldToNewIndex[oldIndex] = newIndex
			if oldIndex < len(objFn) {
				newObjFn = append(newObjFn, objFn[oldIndex])
			} else {
				newObjFn = append(newObjFn, 0)
			}
		}
	}

	return newObjFn, newVariableMap
}

// 輔助函數
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
