// py/py.go

package py

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	json "github.com/goccy/go-json"

	"github.com/HazelnutParadise/insyra"
)

// reinstall the Python environment
func ReinstallPyEnv() error {
	insyra.LogInfo("py", "reinstall", "Reinstalling Python environment...")

	// 清空安裝目錄
	if err := os.RemoveAll(absInstallDir); err != nil {
		return fmt.Errorf("failed to remove install directory: %w", err)
	}

	// 重新創建目錄
	if err := os.MkdirAll(absInstallDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to recreate install directory: %w", err)
	}

	// 重置初始化標誌，強制重新初始化
	isPyEnvInit = false

	// 重新設置環境
	if err := setupUvEnvironment(); err != nil {
		return fmt.Errorf("failed to setup uv environment: %w", err)
	}

	// 重新安裝依賴
	if err := installDependenciesUv(absInstallDir); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	insyra.LogInfo("py", "reinstall", "Python environment reinstalled successfully!")
	return nil
}

// Run the Python file and bind the result to the provided struct pointer.
func RunFile(out any, filePath string) error {
	pyEnvInit()
	file, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read Python file: %w", err)
	}
	code := string(file)
	return RunCode(out, code)
}

// Run the Python file with the given Golang variables and bind the result to the provided struct pointer.
// The codeTemplate should use $v1, $v2, etc. placeholders for variable substitution.
func RunFilef(out any, filePath string, args ...any) error {
	pyEnvInit()
	file, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read Python file: %w", err)
	}
	code := string(file)
	return RunCodef(out, code, args...)
}

// Run the Python code and bind the result to the provided struct pointer.
func RunCode(out any, code string) error {
	return runPythonCode(out, code)
}

// Run the Python code with the given Golang variables and bind the result to the provided struct pointer.
// The codeTemplate should use $v1, $v2, etc. placeholders for variable substitution.
func RunCodef(out any, code string, args ...any) error {
	formattedCode := replacePlaceholders(code, args...)
	return runPythonCode(out, formattedCode)
}

// runPythonCode executes the Python code and binds the result to the provided struct pointer.
func runPythonCode(out any, code string) error {
	pyEnvInit()

	// 生成執行ID
	executionID := generateExecutionID()

	code = generateDefaultPyCode(executionID) + fmt.Sprintf(`
try:
%v
except Exception as e:
    import sys
    sys.stdout.flush()
    sys.stderr.flush()
    insyra_return(None, str(e))
finally:
    import sys
    sys.stdout.flush()
    sys.stderr.flush()
    if not sent:
        insyra_return(None, None)
`, indentCode(code))

	// 創建進程結束通知channel
	processDone := make(chan struct{})
	execErr := make(chan error, 1)

	// 在goroutine中執行Python代碼
	go func() {
		defer close(processDone)
		pythonCmd := exec.Command(pyPath, "-c", code)
		pythonCmd.Stdout = os.Stdout
		pythonCmd.Stderr = os.Stderr
		err := pythonCmd.Run()
		if err != nil {
			execErr <- err
		}
	}()

	// 等待並接收結果
	pyResult := waitForResult(executionID, processDone, execErr)
	// 如果有錯誤（從系統執行或 Python 返回），直接返回
	if pyResult[1] != nil {
		return fmt.Errorf("%v", pyResult[1])
	}
	// 正常執行且無錯誤
	if pyResult[0] == nil {
		return nil
	}

	// 將結果 bind 到傳入的結構指標
	if out != nil {
		jsonData, err := json.Marshal(pyResult[0])
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		if err := json.Unmarshal(jsonData, out); err != nil {
			return fmt.Errorf("failed to unmarshal result to struct: %w", err)
		}
	}
	return nil
}

// Install dependencies using uv pip
func PipInstall(dep string) {
	pyEnvInit()
	pythonCmd := exec.Command("uv", "pip", "install", dep, "--python", pyPath)
	pythonCmd.Dir = absInstallDir
	pythonCmd.Stdout = os.Stdout
	pythonCmd.Stderr = os.Stderr
	err := pythonCmd.Run()
	if err != nil {
		insyra.LogFatal("py", "PipInstall", "Failed to install dependency: %v", err)
	} else {
		insyra.LogInfo("py", "PipInstall", "Installed dependency: %s", dep)
	}
}

// Uninstall dependencies using uv pip
func PipUninstall(dep string) {
	pyEnvInit()
	pythonCmd := exec.Command("uv", "pip", "uninstall", dep, "--python", pyPath)
	pythonCmd.Dir = absInstallDir
	pythonCmd.Stdout = os.Stdout
	pythonCmd.Stderr = os.Stderr
	err := pythonCmd.Run()
	if err != nil {
		insyra.LogFatal("py", "PipUninstall", "Failed to uninstall dependency: %v", err)
	} else {
		insyra.LogInfo("py", "PipUninstall", "Uninstalled dependency: %s", dep)
	}
}

func generateDefaultPyCode(executionID string) string {
	imports := ""
	for imps := range pyDependencies {
		if imps != "" {
			imports += fmt.Sprintf("%s\n", imps)
		}
	}
	return fmt.Sprintf(`
%v
import sys
sent = False
%v
`, imports, builtInFunc(port, executionID))
}

// replacePlaceholders replaces $v1, $v2, etc. placeholders with the corresponding argument values
func replacePlaceholders(template string, args ...any) string {
	result := template
	for i, arg := range args {
		placeholder := fmt.Sprintf("$v%d", i+1)
		var replacement string

		// Convert the argument to a string representation suitable for Python
		switch v := arg.(type) {
		case insyra.IDataList:
			// For IDataList, marshal its Data() to JSON and replace booleans
			if jsonBytes, err := json.Marshal(v.Data()); err == nil {
				jsonStr := string(jsonBytes)
				// Replace JSON booleans with Python booleans, avoiding strings
				jsonStr = replaceBooleans(jsonStr)
				replacement = jsonStr
			} else {
				replacement = "[]"
			}
		case string:
			// For strings, wrap in quotes
			replacement = fmt.Sprintf("%q", v)
		case bool:
			// For bool, use Python boolean literals
			if v {
				replacement = "True"
			} else {
				replacement = "False"
			}
		case []int:
			// For int slices, convert to Python list format
			var elements []string
			for _, val := range v {
				elements = append(elements, strconv.Itoa(val))
			}
			replacement = fmt.Sprintf("[%s]", strings.Join(elements, ", "))
		case []float64:
			// For float64 slices, convert to Python list format
			var elements []string
			for _, val := range v {
				elements = append(elements, strconv.FormatFloat(val, 'f', -1, 64))
			}
			replacement = fmt.Sprintf("[%s]", strings.Join(elements, ", "))
		case []string:
			// For string slices, convert to Python list format
			var elements []string
			for _, val := range v {
				elements = append(elements, fmt.Sprintf("%q", val))
			}
			replacement = fmt.Sprintf("[%s]", strings.Join(elements, ", "))
		default:
			// For other types, try to marshal as JSON for complex structures
			if jsonBytes, err := json.Marshal(v); err == nil {
				replacement = string(jsonBytes)
			} else {
				// Fallback to fmt.Sprintf %v
				replacement = fmt.Sprintf("%v", v)
			}
		}

		result = strings.ReplaceAll(result, placeholder, replacement)
	}
	return result
}

func indentCode(code string) string {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = "    " + line
		}
	}
	return strings.Join(lines, "\n")
}

// replaceBooleans replaces JSON booleans true/false with Python booleans True/False, avoiding strings
func replaceBooleans(jsonStr string) string {
	var result strings.Builder
	inString := false
	quoteChar := byte(0)
	i := 0
	for i < len(jsonStr) {
		c := jsonStr[i]
		if !inString {
			if c == '"' {
				inString = true
				quoteChar = c
			}
		} else {
			if c == quoteChar && (i == 0 || jsonStr[i-1] != '\\') {
				inString = false
			}
		}
		if !inString {
			// check for true
			if i+3 < len(jsonStr) && jsonStr[i:i+4] == "true" {
				result.WriteString("True")
				i += 4
				continue
			}
			// check for false
			if i+4 < len(jsonStr) && jsonStr[i:i+5] == "false" {
				result.WriteString("False")
				i += 5
				continue
			}
		}
		result.WriteByte(c)
		i++
	}
	return result.String()
}
