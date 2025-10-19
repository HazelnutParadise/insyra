// py/py.go

package py

import (
	"fmt"
	"log"
	"os"
	"os/exec"

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
// The codeTemplate should use fmt.Sprintf style formatting (e.g., %q, %d, %v).
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

	// 生成執行ID
	executionID := generateExecutionID()

	code = generateDefaultPyCode(executionID) + code

	// 創建進程結束通知channel
	processDone := make(chan struct{})

	// 在goroutine中執行Python代碼
	go func() {
		defer close(processDone)
		pythonCmd := exec.Command(pyPath, "-c", code)
		pythonCmd.Stdout = os.Stdout
		pythonCmd.Stderr = os.Stderr
		err := pythonCmd.Run()
		if err != nil {
			insyra.LogFatal("py", "RunCode", "Failed to run Python code: %v", err)
		}
	}()

	// 等待並接收結果，當進程結束時自動返回空map
	return waitForResult(executionID, processDone)
}

// Run the Python code with the given Golang variables and return the result.
// The codeTemplate should use fmt.Sprintf style formatting (e.g., %q, %d, %v).
func RunCodef(codeTemplate string, args ...any) map[string]any {
	pyEnvInit()

	// 生成執行ID
	executionID := generateExecutionID()

	// 使用fmt.Sprintf格式化代碼模板
	formattedCode := fmt.Sprintf(codeTemplate, args...)

	// 產生包含 insyra_return 函數的預設 Python 代碼
	fullCode := generateDefaultPyCode(executionID) + formattedCode

	// 創建進程結束通知channel
	processDone := make(chan struct{})

	// 在goroutine中執行Python代碼
	go func() {
		defer close(processDone)
		pythonCmd := exec.Command(pyPath, "-c", fullCode)
		pythonCmd.Stdout = os.Stdout
		pythonCmd.Stderr = os.Stderr
		err := pythonCmd.Run()
		if err != nil {
			log.Fatalf("Failed to run Python code: %v", err)
		}
	}()

	// 等待並接收結果，當進程結束時自動返回空map
	return waitForResult(executionID, processDone)
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
def insyra_return(data, url="http://localhost:%v/pyresult"):
    payload = {"execution_id": "%s", "data": data}
    response = requests.post(url, json=payload)
    if response.status_code != 200:
        print(f"Failed to send result: {response.status_code}")

`, imports, port, executionID)
}
