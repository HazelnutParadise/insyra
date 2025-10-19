// `py` package provides functions for working with Python.
package py

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/HazelnutParadise/insyra"
)

// 用於allpkgs安裝
func init() {}

var isPyEnvInit = false
var isServerRunning = false

// 主要邏輯
// 使用uv管理Python環境
func pyEnvInit() {
	if !isServerRunning {
		isServerRunning = true
		go func() {
			startServer()
		}()
	}
	if isPyEnvInit {
		return
	}

	// 設置pyPath
	if runtime.GOOS == "windows" {
		pyPath = filepath.Join(absInstallDir, ".venv", "Scripts", "python.exe")
	} else {
		pyPath = filepath.Join(absInstallDir, ".venv", "bin", "python")
	}

	// 檢查虛擬環境是否存在，如果存在就不初始化
	if _, err := os.Stat(pyPath); err == nil {
		isPyEnvInit = true
		insyra.LogDebug("py", "init", "Virtual environment already exists, skipping initialization!")
		return
	}

	isPyEnvInit = true

	insyra.LogInfo("py", "init", "Preparing Python environment with uv...")

	// 檢查並安裝uv
	err := ensureUvInstalled()
	if err != nil {
		insyra.LogFatal("py", "init", "Failed to ensure uv is installed: %v", err)
	}

	// 確保安裝目錄存在並清空舊文件
	err = prepareInstallDir()
	if err != nil {
		insyra.LogFatal("py", "init", "Failed to prepare install directory: %v", err)
	}

	// 初始化uv項目和虛擬環境
	err = setupUvEnvironment()
	if err != nil {
		insyra.LogFatal("py", "init", "Failed to setup uv environment: %v", err)
	}

	// 安裝依賴
	err = installDependenciesUv(absInstallDir)
	if err != nil {
		insyra.LogFatal("py", "init", "Failed to install dependencies: %v", err)
	}
	insyra.LogInfo("py", "init", "Dependencies installed successfully!")
}

// 準備安裝目錄
func prepareInstallDir() error {
	// 確保目錄存在
	if _, err := os.Stat(absInstallDir); os.IsNotExist(err) {
		err := os.MkdirAll(absInstallDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", absInstallDir, err)
		}
	}
	return nil
}

// 設置uv環境（初始化項目和創建虛擬環境）
func setupUvEnvironment() error {
	// 檢查虛擬環境是否存在
	venvExists := false
	if _, err := os.Stat(pyPath); err == nil {
		venvExists = true
		insyra.LogDebug("py", "init", "Virtual environment already exists!")
	}

	if venvExists {
		// 虛擬環境存在，直接返回
		return nil
	}

	// 清空安裝目錄
	if err := os.RemoveAll(absInstallDir); err != nil {
		return fmt.Errorf("failed to remove install directory: %w", err)
	}

	// 重新創建目錄
	if err := os.MkdirAll(absInstallDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to recreate install directory: %w", err)
	}

	// 初始化uv項目
	initCmd := exec.Command("uv", "init", "--bare", "--python", pythonVersion)
	initCmd.Dir = absInstallDir
	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("failed to init uv project: %w", err)
	}

	// 同步並創建虛擬環境
	syncCmd := exec.Command("uv", "sync")
	syncCmd.Dir = absInstallDir
	if err := syncCmd.Run(); err != nil {
		return fmt.Errorf("failed to sync uv project: %w", err)
	}

	return nil
}

// 檢查並確保uv已安裝
func ensureUvInstalled() error {
	// 檢查uv是否已安裝
	cmd := exec.Command("uv", "--version")
	err := cmd.Run()
	if err == nil {
		insyra.LogDebug("py", "init", "uv is already installed")
		return nil
	}

	insyra.LogInfo("py", "init", "uv not found, installing uv...")

	// 根據作業系統安裝uv
	if runtime.GOOS == "windows" {
		// 使用PowerShell安裝uv
		installCmd := exec.Command("powershell", "-ExecutionPolicy", "ByPass", "-c", "irm https://astral.sh/uv/install.ps1 | iex")
		return installCmd.Run()
	} else {
		// 優先使用curl，如果失敗則使用wget
		curlCmd := exec.Command("sh", "-c", "curl -LsSf https://astral.sh/uv/install.sh | sh")
		err := curlCmd.Run()
		if err == nil {
			return nil
		}

		// 如果curl失敗，使用wget
		wgetCmd := exec.Command("sh", "-c", "wget -qO- https://astral.sh/uv/install.sh | sh")
		return wgetCmd.Run()
	}
}

// 使用uv安裝依賴
func installDependenciesUv(projectDir string) error {
	fmt.Println("Installing dependencies with uv...")
	totalDeps := len(pyDependencies)
	completed := 0

	for _, dep := range pyDependencies {
		if dep == "" {
			completed++
			showProgress(completed, totalDeps)
			continue
		}

		cmd := exec.Command("uv", "pip", "install", dep, "--python", pyPath)
		cmd.Dir = projectDir
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			fmt.Printf("Stdout: %s", stdout.String())
			fmt.Printf("Stderr: %s", stderr.String())
			return fmt.Errorf("failed to install dependency %s: %w", dep, err)
		}

		completed++
		showProgress(completed, totalDeps)
	}

	fmt.Println()
	return nil
}

// 顯示進度條的輔助函數（單行持續更新）
func showProgress(completed, total int) {
	// 防止進度超過 100%
	if completed > total {
		completed = total
	}

	percentage := (completed * 100) / total
	barWidth := 50
	progressBar := (completed * barWidth) / total

	// 使用 "\r" 符號來覆蓋當前行，實現單行更新
	fmt.Printf("\r[")
	for i := 0; i < progressBar; i++ {
		fmt.Print("=")
	}
	for i := progressBar; i < barWidth; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("] %d%% (%d/%d prepared)", percentage, completed, total)

	// 強制刷新輸出
	_ = os.Stdout.Sync()
}
