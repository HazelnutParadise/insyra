// py/py.go

package py

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	json "github.com/goccy/go-json"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
)

// ReinstallPyEnv deletes the environment directory, .insyra_env/py26a_<os>_<arch>
// under the working directory, with everything in it, packages installed with
// PipInstall included, and builds it again from the pinned versions. The
// pinned uv, kept beside that directory, is not downloaded again.
func ReinstallPyEnv() error {
	pyInitMu.Lock()
	defer pyInitMu.Unlock()
	insyra.LogInfo("py", "reinstall", "Reinstalling Python environment...")
	// Not ready from here on: a delete that fails part-way must leave the
	// next call to build the environment again, not to run what is left.
	isPyEnvInit = false
	if err := os.RemoveAll(absInstallDir); err != nil {
		return fmt.Errorf("failed to remove install directory: %w", err)
	}
	if err := prepareEnvironmentLocked(context.Background()); err != nil {
		return err
	}
	insyra.LogInfo("py", "reinstall", "Python environment reinstalled successfully!")
	return nil
}

// Setup prepares the Python environment now instead of on first use: it
// downloads and verifies the pinned uv if it is missing and has uv bring the
// environment directory to the pinned versions, as the first RunCode,
// PipInstall or other call would. Calling it is optional; call it at start-up
// to fail there rather than on a first request, and to bound the downloads
// with ctx. On an environment that is already prepared it returns nil without
// running uv. A nil ctx is an error, and a ctx that is already done returns
// ctx.Err() before anything starts.
func Setup(ctx context.Context) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	return pyEnvInit(ctx)
}

// errNilContext is what a Context form returns for a nil context, where
// exec.CommandContext would panic.
var errNilContext = errors.New("py: nil context")

// checkContext returns errNilContext for a nil ctx and ctx.Err() once ctx is
// done.
func checkContext(ctx context.Context) error {
	if ctx == nil {
		return errNilContext
	}
	return ctx.Err()
}

// Run runs code and returns the value it passes to insyra.Return, decoded
// into a T by the rules RunCode binds with: *insyra.DataTable or
// *insyra.DataList for a DataFrame or a Series, and a struct, map, slice or
// scalar through JSON. $v1, $v2, … in code are replaced from args as RunCodef
// replaces them. ctx bounds the environment setup and the Python process; a
// nil ctx is an error. On failure Run returns T's zero value and the error.
// T must not be insyra.DataTable or insyra.DataList itself, which cannot be
// copied; use a pointer.
func Run[T any](ctx context.Context, code string, args ...any) (T, error) {
	var out T
	switch any(&out).(type) {
	case *insyra.DataTable:
		return out, errors.New("py: Run cannot return an insyra.DataTable by value; use Run[*insyra.DataTable]")
	case *insyra.DataList:
		return out, errors.New("py: Run cannot return an insyra.DataList by value; use Run[*insyra.DataList]")
	}
	if err := RunCodefContext(ctx, &out, code, args...); err != nil {
		var zero T
		return zero, err
	}
	return out, nil
}

// Run the Python file and bind the result to the provided struct pointer.
func RunFile(out any, filePath string) error {
	return RunFileContext(context.Background(), out, filePath)
}

// Run the Python file with the given Golang variables and bind the result to the provided struct pointer.
// The codeTemplate should use $v1, $v2, etc. placeholders for variable substitution.
func RunFilef(out any, filePath string, args ...any) error {
	return RunFilefContext(context.Background(), out, filePath, args...)
}

// Run the Python code and bind the result to the provided struct pointer.
func RunCode(out any, code string) error {
	return RunCodeContext(context.Background(), out, code)
}

// Run the Python code with the given Golang variables and bind the result to the provided struct pointer.
// The codeTemplate should use $v1, $v2, etc. placeholders for variable substitution.
func RunCodef(out any, code string, args ...any) error {
	return RunCodefContext(context.Background(), out, code, args...)
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
	file, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read Python file: %w", err)
	}
	code := string(file)
	return runPythonCodeContext(ctx, out, code)
}

// Run the Python file with the given Golang variables and bind the result to the provided struct pointer, with context.
func RunFilefContext(ctx context.Context, out any, filePath string, args ...any) error {
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

// RunCodeWithTimeout runs the code under a context that ends after timeout,
// and returns context.DeadlineExceeded when it does.
//
// Deprecated: use RunCodeContext with a context from context.WithTimeout,
// which is what this does. Removed in the release after the one that
// deprecated it.
func RunCodeWithTimeout(timeout time.Duration, out any, code string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return RunCodeContext(ctx, out, code)
}

// runPythonCodeContext executes the Python code and binds the result to out.
// ctx bounds the environment setup and the Python process.
func runPythonCodeContext(ctx context.Context, out any, code string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	addr, err := acquireIPCServer()
	if err != nil {
		return err
	}
	defer releaseIPCServer()
	if err := pyEnvInit(ctx); err != nil {
		return err
	}

	// 生成執行ID
	executionID := generateExecutionID()

	code = generateDefaultPyCode(executionID, addr) + fmt.Sprintf(`
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
	go func(python, path string) {
		defer close(processDone)
		pythonCmd := exec.CommandContext(ctx, python, path)
		pythonCmd.Stdout = os.Stdout
		pythonCmd.Stderr = os.Stderr
		utils.ApplyHideWindow(pythonCmd)
		if err := pythonCmd.Run(); err != nil {
			execErr <- err
		}
	}(pyPath, scriptPath)

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

// checkDependencyName refuses an argument that uv would read as an option
// rather than a package. A single argv such as "--requirement=/etc/reqs.txt"
// makes `uv pip install` read that file instead, so a caller passing a name
// through from somewhere else could install anything.
func checkDependencyName(dep string) error {
	if dep == "" {
		return fmt.Errorf("dependency name is empty")
	}
	if strings.HasPrefix(dep, "-") {
		return fmt.Errorf("dependency name %q starts with '-', which uv would read as an option, not a package", dep)
	}
	return nil
}

// PipInstall installs a package into the environment with uv pip. It is
// PipInstallContext with context.Background().
func PipInstall(dep string) error {
	return PipInstallContext(context.Background(), dep)
}

// PipInstallContext installs a package into the environment with uv pip. ctx
// bounds the environment setup and the install: when it ends, the install is
// stopped and ctx.Err() returned. A nil ctx is an error.
func PipInstallContext(ctx context.Context, dep string) error {
	if err := checkDependencyName(dep); err != nil {
		return fmt.Errorf("PipInstall: %w", err)
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	if err := pyEnvInit(ctx); err != nil {
		return err
	}
	// The "--" stops uv reading anything after it as an option.
	pythonCmd := exec.CommandContext(ctx, uvPath, "pip", "install", "--python", pyPath, "--", dep)
	pythonCmd.Dir = absInstallDir
	var stdout, stderr bytes.Buffer
	pythonCmd.Stdout = &stdout
	pythonCmd.Stderr = &stderr
	utils.ApplyHideWindow(pythonCmd)
	if err := pythonCmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("failed to install dependency %s: %w. stderr: %s", dep, err, stderr.String())
	}
	insyra.LogInfo("py", "PipInstall", "Installed dependency: %s", dep)
	return nil
}

// PipUninstall uninstalls a package from the environment with uv pip. It is
// PipUninstallContext with context.Background().
func PipUninstall(dep string) error {
	return PipUninstallContext(context.Background(), dep)
}

// PipUninstallContext uninstalls a package from the environment with uv pip. ctx
// bounds the environment setup and the uninstall: when it ends, the uninstall is
// stopped and ctx.Err() returned. A nil ctx is an error.
func PipUninstallContext(ctx context.Context, dep string) error {
	if err := checkDependencyName(dep); err != nil {
		return fmt.Errorf("PipUninstall: %w", err)
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	if err := pyEnvInit(ctx); err != nil {
		return err
	}
	pythonCmd := exec.CommandContext(ctx, uvPath, "pip", "uninstall", "--python", pyPath, "--", dep)
	pythonCmd.Dir = absInstallDir
	var stdout, stderr bytes.Buffer
	pythonCmd.Stdout = &stdout
	pythonCmd.Stderr = &stderr
	utils.ApplyHideWindow(pythonCmd)
	if err := pythonCmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("failed to uninstall dependency %s: %w. stderr: %s", dep, err, stderr.String())
	}
	insyra.LogInfo("py", "PipUninstall", "Uninstalled dependency: %s", dep)
	return nil
}

// PipList returns a map of installed package names to their versions for the Python environment managed by uv.
// It runs `uv pip list --format=json --python <pyPath>` and parses the JSON output.
func PipList() (map[string]string, error) {
	if err := pyEnvInit(context.Background()); err != nil {
		return nil, err
	}

	cmd := exec.Command(uvPath, "pip", "list", "--format=json", "--python", pyPath)
	cmd.Dir = absInstallDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	utils.ApplyHideWindow(cmd)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	type pipPkg struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	var pkgs []pipPkg
	if err := json.Unmarshal(stdout.Bytes(), &pkgs); err != nil {
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
	if err := pyEnvInit(context.Background()); err != nil {
		return nil, err
	}

	cmd := exec.Command(uvPath, "pip", "freeze", "--python", pyPath)
	cmd.Dir = absInstallDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	utils.ApplyHideWindow(cmd)

	if err := cmd.Run(); err != nil {
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
		// attempt best-effort cleanup without logging (caller expects an error)
		_ = tmpFile.Close()
		_ = os.Remove(scriptPath)
		return "", nil, fmt.Errorf("failed to write temp python file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		// attempt best-effort cleanup without logging (caller expects an error)
		_ = os.Remove(scriptPath)
		return "", nil, fmt.Errorf("failed to close temp python file: %w", err)
	}

	cleanup := func() {
		if rerr := os.Remove(scriptPath); rerr != nil && !os.IsNotExist(rerr) {
			insyra.LogWarning("py", "createTempPythonScript", "failed to remove temp file: %v", rerr)
		}
	}

	return scriptPath, cleanup, nil
}

func generateDefaultPyCode(executionID, addr string) string {
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
`, imports, builtInFunc(addr, executionID))
}

// placeholderPattern matches $v1, $v2, … by their whole number. A number with
// a leading zero is not a placeholder.
var placeholderPattern = regexp.MustCompile(`\$v([1-9][0-9]*)`)

// replacePlaceholders replaces $v1, $v2, … in template with the Python
// literals of the arguments at those positions. It reads the template once, so
// text inserted for one placeholder is never searched for another, and $v10 is
// the tenth argument rather than $v1 followed by 0. A placeholder past the last
// argument is left as written. Only the arguments the template uses are
// converted, and one that cannot be written as a Python literal is an error.
func replacePlaceholders(template string, args ...any) (string, error) {
	literals := make([]string, len(args))
	converted := make([]bool, len(args))
	var convErr error
	result := placeholderPattern.ReplaceAllStringFunc(template, func(placeholder string) string {
		n, err := strconv.Atoi(placeholder[len("$v"):])
		if err != nil || n > len(args) || convErr != nil {
			return placeholder
		}
		if !converted[n-1] {
			literal, err := pythonLiteral(args[n-1])
			if err != nil {
				convErr = fmt.Errorf("%s: %w", placeholder, err)
				return placeholder
			}
			literals[n-1], converted[n-1] = literal, true
		}
		return literals[n-1]
	})
	if convErr != nil {
		return "", convErr
	}
	return result, nil
}

// pythonLiteral writes arg as Python source that evaluates to it.
func pythonLiteral(arg any) (string, error) {
	var replacement string
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
			// Marshal the name to a JSON string literal so quotes/newlines in
			// the name cannot break out of the Python source (code injection).
			// JSON double-quoted string literals are valid Python literals.
			nameBytes, err := json.Marshal(name)
			if err != nil {
				return "", fmt.Errorf("failed to marshal IDataList name: %w", err)
			}
			replacement += "name=" + string(nameBytes) + ","
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
		// For float64 slices, convert to Python list format. A NaN or an
		// infinity has no Python literal, so it is refused as a scalar one is.
		var elements []string
		for _, val := range v {
			if math.IsNaN(val) || math.IsInf(val, 0) {
				return "", fmt.Errorf("a []float64 holding %v cannot be written as a Python value", val)
			}
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
		// For other types, try to marshal as JSON for complex structures.
		// A value JSON cannot write is refused rather than written as
		// formatted text, which would put it into the script as code.
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("a %T cannot be written as a Python value: %w", v, err)
		}
		replacement = replaceJsonLiterals(string(jsonBytes))
	}
	return replacement, nil
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
			if c == quoteChar {
				// The closing quote is escaped only when preceded by an ODD number
				// of backslashes; `\\"` (escaped backslash then quote) is NOT escaped.
				bs := 0
				for k := i - 1; k >= 0 && jsonStr[k] == '\\'; k-- {
					bs++
				}
				if bs%2 == 0 {
					inString = false
				}
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
