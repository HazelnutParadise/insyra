// py/py.go

package py

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/HazelnutParadise/insyra"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigFastest

// Run the Python file and return the result.
func RunFile(filePath string) map[string]any {
	pyEnvInit()
	file, err := os.ReadFile(filePath)
	if err != nil {
		insyra.LogFatal("py", "RunFile", "Failed to read Python file: %v", err)
	}
	code := string(file)
	return RunCode(code)
}

// Run the Python file with the given Golang variables and return the result.
// The variables should be passed by $v1, $v2, $v3... in the codeTemplate.
func RunFilef(filePath string, args ...any) map[string]any {
	pyEnvInit()
	file, err := os.ReadFile(filePath)
	if err != nil {
		insyra.LogFatal("py", "RunFilef", "Failed to read Python file: %v", err)
	}
	code := string(file)
	return RunCodef(code, args...)
}

// Run the Python code and return the result.
func RunCode(code string) map[string]any {
	pyEnvInit()
	code = generateDefaultPyCode() + code

	// 執行 Python 代碼
	pythonCmd := exec.Command(pyPath, "-c", code)
	pythonCmd.Dir = installDir
	pythonCmd.Stdout = os.Stdout
	pythonCmd.Stderr = os.Stderr
	err := pythonCmd.Run()
	if err != nil {
		insyra.LogFatal("py", "RunCode", "Failed to run Python code: %v", err)
	}

	// 從 server 接收資料
	return pyResult
}

// Run the Python code with the given Golang variables and return the result.
// The variables should be passed by $v1, $v2, $v3... in the codeTemplate.
func RunCodef(codeTemplate string, args ...any) map[string]any {
	pyEnvInit()
	// 產生包含 insyra_return 函數的預設 Python 代碼
	codeTemplate = generateDefaultPyCode() + codeTemplate

	// 將所有 Go 變數轉換為一個 JSON 字典傳遞給 Python
	dataMap := make(map[string]any)
	for i, arg := range args {
		pythonVarName := fmt.Sprintf("var%d", i+1)
		dataMap[pythonVarName] = arg
	}

	// 將字典序列化為 JSON
	jsonData, err := json.Marshal(dataMap)
	if err != nil {
		insyra.LogFatal("py", "RunCodef", "Failed to serialize variables: %v", err)
	}

	// 替換 codeTemplate 中的 $v1, $v2... 佔位符為對應的 vars['var1'], vars['var2']...
	for i := range args {
		placeholder := fmt.Sprintf("$v%d", i+1)
		codeTemplate = strings.ReplaceAll(codeTemplate, placeholder, fmt.Sprintf("IN5YRA_變數v['var%d']", i+1))
	}

	// 在 Python 中生成變數賦值語句
	pythonVarCode := fmt.Sprintf("IN5YRA_變數v = json.loads('%s')", string(jsonData))

	// 構建完整的 Python 代碼，並確保正確導入 json 模組
	fullCode := fmt.Sprintf(`
import json
%s
%s
`, pythonVarCode, codeTemplate)

	// 執行 Python 代碼
	pythonCmd := exec.Command(pyPath, "-c", fullCode)
	pythonCmd.Stdout = os.Stdout
	pythonCmd.Stderr = os.Stderr

	// 執行 Python 命令
	err = pythonCmd.Run()
	if err != nil {
		log.Fatalf("Failed to run Python code: %v", err)
	}

	// 從 server 接收資料
	return pyResult
}

// Install dependencies using pip
func PipInstall(dep string) {
	pyEnvInit()
	pythonCmd := exec.Command(pyPath, "-m", "pip", "install", dep)
	pythonCmd.Dir = installDir
	pythonCmd.Stdout = os.Stdout
	pythonCmd.Stderr = os.Stderr
	err := pythonCmd.Run()
	if err != nil {
		insyra.LogFatal("py", "PipInstall", "Failed to install dependency: %v", err)
	} else {
		insyra.LogInfo("py", "PipInstall", "Installed dependency: %s", dep)
	}
}

// Uninstall dependencies using pip
func PipUninstall(dep string) {
	pyEnvInit()
	pythonCmd := exec.Command(pyPath, "-m", "pip", "uninstall", "-y", dep)
	pythonCmd.Dir = installDir
	pythonCmd.Stdout = os.Stdout
	pythonCmd.Stderr = os.Stderr
	err := pythonCmd.Run()
	if err != nil {
		insyra.LogFatal("py", "PipUninstall", "Failed to uninstall dependency: %v", err)
	} else {
		insyra.LogInfo("py", "PipUninstall", "Uninstalled dependency: %s", dep)
	}
}

func generateDefaultPyCode() string {
	imports := ""
	for imps, _ := range pyDependencies {
		if imps != "" {
			imports += fmt.Sprintf("%s\n", imps)
		}
	}
	return fmt.Sprintf(`
%v
def insyra_return(data, url="http://localhost:%v/pyresult"):
    response = requests.post(url, json=data)
    if response.status_code != 200:
        print(f"Failed to send result: {response.status_code}")

`, imports, port)
}
