// py/init.go

package py

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// 主要邏輯
func init() {
	if runtime.GOOS == "windows" {
		pyPath = filepath.Join(absInstallDir, "python", "python.exe")
	}

	go startServer()
	insyra.LogInfo("py.init: Preparing Python environment...")
	// 如果目錄不存在，自動創建
	if _, err := os.Stat(installDir); os.IsNotExist(err) {
		os.MkdirAll(installDir, os.ModePerm)
	} else {
		// 檢查 Python 執行檔是否已存在
		if _, err := os.Stat(pyPath); err == nil {
			insyra.LogDebug("Python installation already exists!")
			checkPythonPath()
			err = installDependencies()
			if err != nil {
				insyra.LogFatal("Failed to install dependencies: %v", err)
			}
			insyra.LogInfo("Dependencies installation completed successfully!")
			return
		}
	}

	// 安裝 Python
	err := installPython(pythonVersion)
	if err != nil {
		insyra.LogFatal("py.init: Failed to install Python: %v", err)
	}
	insyra.LogInfo("py.init: Python installation completed successfully!")

	checkPythonPath()
	err = installDependencies()
	if err != nil {
		insyra.LogFatal("py.init: Failed to install dependencies: %v", err)
	}
	insyra.LogInfo("py.init: Dependencies has been prepared successfully!")
}

func checkPythonPath() {
	if runtime.GOOS == "windows" {
		// 執行 Python 檢查腳本
		cmd := exec.Command(pyPath, "-c", fmt.Sprintf(`
		 import sys
		 sys.path.append(r'%s')
		 sys.path.append(r'%s')
		 try:
			 import encodings
			 print("encodings module loaded successfully")
		 except Exception as e:
			 print(f"Failed to load encodings module: {e}")
		 `, filepath.Join(absInstallDir, "python", "Lib"), filepath.Join(absInstallDir, "python", "DLLs")))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Failed to run Python script: %v\n", err)
		}
	}
}

// 下載並安裝 Python 的邏輯
func installPython(version string) error {
	if runtime.GOOS == "windows" {
		return installPythonOnWindows(version)
	}
	return installPythonOnUnix(version, absInstallDir)
}

func installPythonOnWindows(version string) error {
	// 指定安裝路徑
	pythonSourceDir := filepath.Join(absInstallDir, "python-source")
	pythonInstallDir := filepath.Join(absInstallDir, "python")

	// 下載 Python 原始碼
	err := downloadPythonSource(version, pythonSourceDir)
	if err != nil {
		return fmt.Errorf("failed to download Python source: %w", err)
	}

	// 編譯並安裝 Python
	err = compilePythonSourceWindows(pythonSourceDir, pythonInstallDir)
	if err != nil {
		return fmt.Errorf("failed to compile and install Python: %w", err)
	}

	// 將 Python 安裝路徑添加到 PATH
	err = os.Setenv("PATH", fmt.Sprintf("%s;%s", os.Getenv("PATH"), pythonInstallDir))
	if err != nil {
		return fmt.Errorf("failed to update PATH environment variable: %w", err)
	}

	moveLibDirectory(pythonSourceDir, pythonInstallDir)

	return nil
}

// Unix-like 平台安裝 (Linux/macOS)
func installPythonOnUnix(version string, installDir string) error {
	downloadURL := fmt.Sprintf("https://www.python.org/ftp/python/%s/Python-%s.tgz", version, version)
	pythonTar := filepath.Join(os.TempDir(), fmt.Sprintf("Python-%s.tgz", version))

	fmt.Println("Downloading Python for Unix-like systems...")
	err := downloadFile(pythonTar, downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download Python: %w", err)
	}

	fmt.Println("Extracting Python files...")
	err = extractTar(pythonTar, os.TempDir())
	if err != nil {
		return fmt.Errorf("failed to extract Python: %w", err)
	}

	pythonSrcDir := filepath.Join(os.TempDir(), fmt.Sprintf("Python-%s", version))

	fmt.Println("Configuring and installing Python...")
	return installPythonFromSource(pythonSrcDir, installDir)
}

// 確保目錄存在，如果不存在則創建
func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

func downloadPythonSource(version string, destDir string) error {
	// 確保目標目錄存在
	err := ensureDir(destDir)
	if err != nil {
		return fmt.Errorf("failed to ensure destination directory: %w", err)
	}

	downloadURL := fmt.Sprintf("https://www.python.org/ftp/python/%s/Python-%s.tgz", version, version)
	pythonTar := filepath.Join(destDir, fmt.Sprintf("Python-%s.tgz", version))

	fmt.Println("Downloading Python source code...")
	err = downloadFile(pythonTar, downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download Python source code: %w", err)
	}

	fmt.Println("Extracting Python source code...")
	err = extractTar(pythonTar, destDir)
	if err != nil {
		return fmt.Errorf("failed to extract Python source code: %w", err)
	}

	return nil
}

// 下載檔案
func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// 解壓縮 .tgz（適用於 Unix-like 系統）
func extractTar(filepath string, destDir string) error {
	cmd := exec.Command("tar", "-xvzf", filepath, "-C", destDir)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	return cmd.Run()
}

// 從原始碼編譯並安裝 Python（適用於 Unix-like 系統）
func installPythonFromSource(srcDir string, installDir string) error {
	configureCmd := exec.Command("./configure", fmt.Sprintf("--prefix=%s", installDir))
	configureCmd.Dir = srcDir
	// configureCmd.Stdout = os.Stdout
	// configureCmd.Stderr = os.Stderr
	if err := configureCmd.Run(); err != nil {
		return err
	}

	makeCmd := exec.Command("make")
	makeCmd.Dir = srcDir
	// makeCmd.Stdout = os.Stdout
	// makeCmd.Stderr = os.Stderr
	if err := makeCmd.Run(); err != nil {
		return err
	}

	installCmd := exec.Command("make", "install")
	installCmd.Dir = srcDir
	// installCmd.Stdout = os.Stdout
	// installCmd.Stderr = os.Stderr
	return installCmd.Run()
}

func compilePythonSourceWindows(sourceDir string, installDir string) error {
	// 使用 Windows 特定的構建方式
	buildCmd := exec.Command(filepath.Join(sourceDir, "Python-"+pythonVersion, "PCbuild", "build.bat"), "-e", "-v")
	buildCmd.Dir = filepath.Join(sourceDir, "Python-"+pythonVersion, "PCbuild")
	// buildCmd.Stdout = os.Stdout
	// buildCmd.Stderr = os.Stderr

	err := buildCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to compile Python source code using build.bat: %w", err)
	}

	// 將編譯後的文件複製到安裝目錄
	fmt.Println("Installing Python...")
	installCmd := exec.Command("xcopy", "/E", "/I", filepath.Join(sourceDir, "Python-"+pythonVersion, "PCbuild", "amd64"), installDir)
	// installCmd.Stdout = os.Stdout
	// installCmd.Stderr = os.Stderr

	err = installCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to install compiled Python: %w", err)
	}

	return nil
}

func moveLibDirectory(sourceDir, installDir string) error {
	// 檢查 sourceDir 中的 'Lib' 目錄是否存在
	libSourcePath := filepath.Join(sourceDir, "python-"+pythonVersion, "Lib")
	if _, err := os.Stat(libSourcePath); os.IsNotExist(err) {
		return fmt.Errorf("Lib directory not found in source directory: %s", sourceDir)
	}

	// 目標路徑
	libDestPath := filepath.Join(installDir, "Lib")

	// 使用系統命令將 'Lib' 目錄複製到安裝目錄
	fmt.Println("Moving Lib directory to installation path...")
	err := exec.Command("xcopy", libSourcePath, libDestPath, "/E", "/I").Run()
	if err != nil {
		return fmt.Errorf("failed to move Lib directory: %w", err)
	}

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
	os.Stdout.Sync()
}

func installDependencies() error {
	totalDeps := len(pyDependencies)           // 總依賴數量
	progressChan := make(chan bool, totalDeps) // 進度管道，設置緩衝區為依賴數量

	// 設置一個 Goroutine 來統一更新進度條
	go func() {
		completed := 0
		ticker := time.NewTicker(1 * time.Millisecond) // 每 1 毫秒統一更新一次
		defer ticker.Stop()

		for range ticker.C {
			select {
			case <-progressChan:
				completed++
				if completed >= totalDeps {
					showProgress(completed, totalDeps)
					return // 停止更新進度
				}
				showProgress(completed, totalDeps)
			}
		}
	}()

	// 單線程安裝依賴
	for _, dep := range pyDependencies {
		if dep == "" {
			progressChan <- true
			continue
		}

		// 構建命令
		cmdSlice := append([]string{pyPath, "-m"}, "pip", "install", dep)

		// 執行命令，將輸出丟棄
		cmd := exec.Command(cmdSlice[0], cmdSlice[1:]...)
		cmd.Dir = absInstallDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard

		err := cmd.Run()
		if err != nil {
			progressChan <- true // 即便失敗，依然更新進度
			return fmt.Errorf("failed to install dependency %s: %w", dep, err)
		}

		// 安裝成功，發送進度信號
		progressChan <- true
	}

	close(progressChan) // 關閉進度管道

	return nil
}
