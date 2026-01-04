// py/py.go

package py

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	json "github.com/goccy/go-json"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
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
	if err := pyEnvInit(); err != nil {
		return err
	}
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
	if err := pyEnvInit(); err != nil {
		return err
	}
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
	formattedCode, err := replacePlaceholders(code, args...)
	if err != nil {
		return fmt.Errorf("failed to format code: %w", err)
	}
	return runPythonCode(out, formattedCode)
}

// runPythonCode executes the Python code and binds the result to the provided struct pointer.
func runPythonCode(out any, code string) error {
	if err := pyEnvInit(); err != nil {
		return err
	}

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

	scriptPath, cleanup, err := createTempPythonScript(code)
	if err != nil {
		return err
	}
	defer cleanup()

	// 在goroutine中執行Python代碼
	go func(path string) {
		defer close(processDone)
		pythonCmd := exec.Command(pyPath, path)
		pythonCmd.Stdout = os.Stdout
		pythonCmd.Stderr = os.Stderr
		utils.ApplyHideWindow(pythonCmd)
		if err := pythonCmd.Run(); err != nil {
			execErr <- err
		}
	}(scriptPath)

	// 等待並接收結果
	pyResult := waitForResult(executionID, processDone, execErr)
	// 如果有錯誤（從系統執行或 Python 返回），直接返回
	if pyResult[1] != nil {
		return fmt.Errorf("%v", pyResult[1])
	}
	// 正常執行且無錯誤；即使回傳值為 nil 也要呼叫 bindPyResult 以便把 nil 綁定到 out（例如清空 interface 變數）
	if pyResult[0] == nil {
		if out != nil {
			if err := bindPyResult(out, nil); err != nil {
				return err
			}
		}
		return nil
	}

	// 將結果 bind 到傳入的結構指標
	if out != nil {
		if err := bindPyResult(out, pyResult[0]); err != nil {
			return err
		}
	}
	return nil
}

// Run the Python code using the provided context. If the context is canceled
// the underlying Python process will be killed and the function will return
// the context error (e.g., context.Canceled or context.DeadlineExceeded).
func RunCodeContext(ctx context.Context, out any, code string) error {
	// Delegate to a context-aware runner
	return runPythonCodeContext(ctx, out, code)
}

// Run the Python code with the given Golang variables and a Context.
// The codeTemplate should use $v1, $v2, etc. placeholders for variable substitution.
func RunCodefContext(ctx context.Context, out any, code string, args ...any) error {
	formattedCode, err := replacePlaceholders(code, args...)
	if err != nil {
		return fmt.Errorf("failed to format code: %w", err)
	}
	return runPythonCodeContext(ctx, out, formattedCode)
}

// Run the Python file and bind the result to the provided struct pointer, with context.
func RunFileContext(ctx context.Context, out any, filePath string) error {
	if err := pyEnvInit(); err != nil {
		return err
	}
	file, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read Python file: %w", err)
	}
	code := string(file)
	return runPythonCodeContext(ctx, out, code)
}

// Run the Python file with the given Golang variables and bind the result to the provided struct pointer, with context.
func RunFilefContext(ctx context.Context, out any, filePath string, args ...any) error {
	if err := pyEnvInit(); err != nil {
		return err
	}
	file, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read Python file: %w", err)
	}
	code := string(file)
	formattedCode, err := replacePlaceholders(code, args...)
	if err != nil {
		return fmt.Errorf("failed to format code: %w", err)
	}
	return runPythonCodeContext(ctx, out, formattedCode)
}

// Run the Python code with a timeout. This is a convenience wrapper that creates
// a context with timeout and runs the code. If the timeout occurs it returns
// context.DeadlineExceeded (i.e., ctx.Err()).
func RunCodeWithTimeout(timeout time.Duration, out any, code string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return runPythonCodeContext(ctx, out, code)
}

// runPythonCodeContext executes the Python code and binds the result to the provided struct pointer.
// It behaves like runPythonCode but uses the provided Context so callers can cancel the execution.
func runPythonCodeContext(ctx context.Context, out any, code string) error {
	if err := pyEnvInit(); err != nil {
		return err
	}

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

	scriptPath, cleanup, err := createTempPythonScript(code)
	if err != nil {
		return err
	}
	defer cleanup()

	// 在goroutine中執行Python代碼
	go func(path string) {
		defer close(processDone)
		pythonCmd := exec.CommandContext(ctx, pyPath, path)
		pythonCmd.Stdout = os.Stdout
		pythonCmd.Stderr = os.Stderr
		utils.ApplyHideWindow(pythonCmd)
		if err := pythonCmd.Run(); err != nil {
			execErr <- err
		}
	}(scriptPath)

	// 等待並接收結果
	pyResult := waitForResult(executionID, processDone, execErr)
	// 如果有錯誤（從系統執行或 Python 返回），直接返回；
	// 若原因是 context 取消/截止，優先回傳 ctx.Err()（例如 context.Canceled / context.DeadlineExceeded）
	if pyResult[1] != nil {
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%v", pyResult[1])
	}
	// 正常執行且無錯誤；即使回傳值為 nil 也要呼叫 bindPyResult 以便把 nil 綁定到 out（例如清空 interface 變數）
	if pyResult[0] == nil {
		if out != nil {
			if err := bindPyResult(out, nil); err != nil {
				return err
			}
		}
		return nil
	}

	// 將結果 bind 到傳入的結構指標
	if out != nil {
		if err := bindPyResult(out, pyResult[0]); err != nil {
			return err
		}
	}
	return nil
}

// Install dependencies using uv pip
func PipInstall(dep string) error {
	if err := pyEnvInit(); err != nil {
		return err
	}
	pythonCmd := exec.Command("uv", "pip", "install", dep, "--python", pyPath)
	pythonCmd.Dir = absInstallDir
	pythonCmd.Stdout = os.Stdout
	pythonCmd.Stderr = os.Stderr
	err := pythonCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to install dependency %s: %w", dep, err)
	}
	insyra.LogInfo("py", "PipInstall", "Installed dependency: %s", dep)
	return nil
}

// Uninstall dependencies using uv pip
func PipUninstall(dep string) error {
	if err := pyEnvInit(); err != nil {
		return err
	}
	pythonCmd := exec.Command("uv", "pip", "uninstall", dep, "--python", pyPath)
	pythonCmd.Dir = absInstallDir
	pythonCmd.Stdout = os.Stdout
	pythonCmd.Stderr = os.Stderr
	err := pythonCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to uninstall dependency %s: %w", dep, err)
	}
	insyra.LogInfo("py", "PipUninstall", "Uninstalled dependency: %s", dep)
	return nil
}

// PipList returns a map of installed package names to their versions for the Python environment managed by uv.
// It runs `uv pip list --format=json --python <pyPath>` and parses the JSON output.
func PipList() (map[string]string, error) {
	if err := pyEnvInit(); err != nil {
		return nil, err
	}

	cmd := exec.Command("uv", "pip", "list", "--format=json", "--python", pyPath)
	cmd.Dir = absInstallDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		insyra.LogInfo("py", "PipList", "Failed to list installed packages. Stdout: %s Stderr: %s Error: %v", stdout.String(), stderr.String(), err)
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	type pipPkg struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	var pkgs []pipPkg
	if err := json.Unmarshal(stdout.Bytes(), &pkgs); err != nil {
		insyra.LogInfo("py", "PipList", "Failed to parse pip list JSON: %v", err)
		return nil, fmt.Errorf("failed to parse pip list output: %w", err)
	}

	result := make(map[string]string, len(pkgs))
	for _, p := range pkgs {
		result[p.Name] = p.Version
	}

	insyra.LogInfo("py", "PipList", "Found %d installed packages", len(pkgs))
	return result, nil
}

// PipFreeze returns the lines produced by `uv pip freeze --python <pyPath>` (one line per package, e.g. package==version).
func PipFreeze() ([]string, error) {
	if err := pyEnvInit(); err != nil {
		return nil, err
	}

	cmd := exec.Command("uv", "pip", "freeze", "--python", pyPath)
	cmd.Dir = absInstallDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		insyra.LogInfo("py", "PipFreeze", "Failed to run pip freeze. Stdout: %s Stderr: %s Error: %v", stdout.String(), stderr.String(), err)
		return nil, fmt.Errorf("failed to freeze installed packages: %w", err)
	}

	outStr := strings.TrimSpace(stdout.String())
	if outStr == "" {
		return []string{}, nil
	}
	lines := strings.Split(outStr, "\n")
	return lines, nil
}

func createTempPythonScript(code string) (string, func(), error) {
	tmpFile, err := os.CreateTemp("", "insyra-*.py")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp python file: %w", err)
	}

	scriptPath := tmpFile.Name()

	if _, err := tmpFile.WriteString(code); err != nil {
		tmpFile.Close()
		os.Remove(scriptPath)
		return "", nil, fmt.Errorf("failed to write temp python file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(scriptPath)
		return "", nil, fmt.Errorf("failed to close temp python file: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(scriptPath)
	}

	return scriptPath, cleanup, nil
}

func generateDefaultPyCode(executionID string) string {
	imports := ""
	for imps := range pyDependencies {
		if imps != "" {
			imports += fmt.Sprintf("%s\n", imps)
		}
	}
	// Get IPC address (waits for server if needed)
	addr := getIPCAddress()
	return fmt.Sprintf(`
%v
import sys
sent = False
%v
`, imports, builtInFunc(addr, executionID))
}

// replacePlaceholders replaces $v1, $v2, etc. placeholders with the corresponding argument values
func replacePlaceholders(template string, args ...any) (string, error) {
	result := template
	for i, arg := range args {
		placeholder := fmt.Sprintf("$v%d", i+1)
		var replacement string

		// Convert the argument to a string representation suitable for Python
		switch v := arg.(type) {
		case insyra.IDataList:
			// For IDataList, marshal to JSON and format as pd.Series
			jsonBytes, err := json.Marshal(v.Data())
			if err != nil {
				return "", fmt.Errorf("failed to marshal IDataList argument: %w", err)
			}
			jsonStr := string(jsonBytes)
			// Replace JSON literals with Python literals, avoiding strings
			jsonStr = replaceJsonLiterals(jsonStr)
			replacement = "pd.Series("
			if name := v.GetName(); name != "" {
				replacement += "name='" + name + "',"
			}
			replacement += "data=" + jsonStr + ")"
		case insyra.IDataTable:
			data := v.To2DSlice()
			// For IDataTable, marshal to JSON and format as pd.DataFrame
			jsonBytes, err := json.Marshal(data)
			if err != nil {
				return "", fmt.Errorf("failed to marshal IDataTable argument: %w", err)
			}
			jsonStr := string(jsonBytes)
			// Replace JSON literals with Python literals, avoiding strings
			jsonStr = replaceJsonLiterals(jsonStr)
			replacement = "pd.DataFrame("
			if colnames := v.ColNames(); len(colnames) > 0 {
				var allempty = true
				for _, name := range colnames {
					if name != "" {
						allempty = false
						break
					}
				}
				if !allempty {
					colsJson, _ := json.Marshal(colnames)
					replacement += "columns=" + replaceJsonLiterals(string(colsJson)) + ","
				}
			}
			if rownames := v.RowNames(); len(rownames) > 0 {
				var allempty = true
				for _, name := range rownames {
					if name != "" {
						allempty = false
						break
					}
				}
				if !allempty {
					rownamesJson, _ := json.Marshal(rownames)
					replacement += "index=" + replaceJsonLiterals(string(rownamesJson)) + ","
				}
			}
			replacement += "data=" + jsonStr + ")"
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
				replacement = replaceJsonLiterals(string(jsonBytes))
			} else {
				// Fallback to fmt.Sprintf %v
				replacement = fmt.Sprintf("%v", v)
			}
		}

		result = strings.ReplaceAll(result, placeholder, replacement)
	}
	return result, nil
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

// replaceJsonLiterals replaces JSON literals true/false/null with Python literals True/False/None, avoiding strings
func replaceJsonLiterals(jsonStr string) string {
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
			// check for null
			if i+3 < len(jsonStr) && jsonStr[i:i+4] == "null" {
				result.WriteString("None")
				i += 4
				continue
			}
		}
		result.WriteByte(c)
		i++
	}
	return result.String()
}
