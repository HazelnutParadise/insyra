package lp

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"

	"github.com/HazelnutParadise/insyra"
)

// init 函數將會自動檢查並安裝 GLPK
func init() {
	installGLPK()
}

// 自動安裝 GLPK 的函數
func installGLPK() {
	switch runtime.GOOS {
	case "linux":
		initializeOnLinux()
	case "darwin":
		initializeOnMacOS()
	case "windows":
		initializeOnWindows()
	default:
		log.Println("Unsupported operating system.")
	}
}

// =========================== macOS 安裝邏輯 ===========================
func initializeOnMacOS() {
	// 常見的 macOS 路徑，用於查找 glpsol，包括使用者安裝目錄
	commonPaths := []string{
		"/usr/local/bin/glpsol",
		"/opt/homebrew/bin/glpsol",
		filepath.Join(os.Getenv("HOME"), "local", "bin", "glpsol"),
	}

findGLPK:
	glpsolPath, err := findInPaths(commonPaths)
	if err == nil {
		insyra.LogInfo("lp", "init", "GLPK already installed at: %s", glpsolPath)
		// 設置 GLPK_PATH 環境變數
		glpkDir := filepath.Dir(glpsolPath)
		os.Setenv("GLPK_PATH", glpkDir)

		// 將 GLPK 目錄添加到 PATH 環境變數
		currentPath := os.Getenv("PATH")
		newPath := glpkDir + string(os.PathListSeparator) + currentPath
		os.Setenv("PATH", newPath)

		return
	}

	insyra.LogInfo("lp", "init", "GLPK not found, installing from source on macOS...")

	// 下載並安裝 GLPK 源碼
	downloadAndInstallGLPK_Source()
	goto findGLPK
}

// =========================== Linux 安裝邏輯 ===========================
func initializeOnLinux() {
	// 常見的 Linux 路徑，用於查找 glpsol
	commonPaths := []string{
		"/usr/local/bin/glpsol",
		"/usr/bin/glpsol",
		"/bin/glpsol",
		filepath.Join(os.Getenv("HOME"), "local", "bin", "glpsol"),
	}

findGLPK:
	glpsolPath, err := findInPaths(commonPaths)
	if err == nil {
		insyra.LogInfo("lp", "init", "GLPK already installed at: %s", glpsolPath)
		// 設置 GLPK_PATH 環境變數
		glpkDir := filepath.Dir(glpsolPath)
		os.Setenv("GLPK_PATH", glpkDir)

		// 將 GLPK 目錄添加到 PATH 環境變數
		currentPath := os.Getenv("PATH")
		newPath := glpkDir + string(os.PathListSeparator) + currentPath
		os.Setenv("PATH", newPath)

		insyra.LogDebug("lp", "init", "GLPK environment variables set. GLPK_PATH=%s", glpkDir)
		return
	}

	insyra.LogInfo("lp", "init", "GLPK not found, installing from source on Linux...")

	// 下載並安裝 GLPK 源碼
	downloadAndInstallGLPK_Source()
	goto findGLPK
}

// =========================== Windows 安裝邏輯 ===========================
func initializeOnWindows() {
	// 檢查 glpsol 是否已經安裝
	glpsolPath, err := locateOrInstallGLPK_Win()
	if err != nil {
		insyra.LogFatal("lp", "init", "Failed to initialize: %v", err)
	}

	// 設置 GLPK_PATH 環境變數
	glpkDir := filepath.Dir(glpsolPath)
	os.Setenv("GLPK_PATH", glpkDir)

	// 將 GLPK 目錄添加到 PATH 環境變數
	currentPath := os.Getenv("PATH")
	newPath := glpkDir + string(os.PathListSeparator) + currentPath
	os.Setenv("PATH", newPath)

	insyra.LogDebug("lp", "init", "GLPK environment variables set. GLPK_PATH=%s", glpkDir)
}

// 用於下載並安裝 GLPK 源碼
func downloadAndInstallGLPK_Source() {
	// 下載 GLPK 源碼包
	downloadURL := "https://ftp.gnu.org/gnu/glpk/glpk-5.0.tar.gz"
	tarPath := filepath.Join(os.TempDir(), "glpk.tar.gz")
	installDir := filepath.Join(os.TempDir(), "glpk")

	insyra.LogDebug("lp", "init", "Downloading GLPK from %s", downloadURL)
	if err := downloadFile(tarPath, downloadURL); err != nil {
		insyra.LogFatal("lp", "init", "Failed to download GLPK: %v", err)
	}

	// 解壓並安裝
	if err := untar(tarPath, installDir); err != nil {
		insyra.LogFatal("lp", "init", "Failed to extract GLPK: %v", err)
	}

	// 查找 configure 文件的路徑
	configurePath, err := findSubDirWithConfigure(installDir)
	if err != nil {
		insyra.LogFatal("lp", "init", "Failed to find configure file: %v", err)
	}

	// 編譯安裝 GLPK
	buildAndInstallGLPK(configurePath)
}

// 查找 configure 文件所在的目錄
func findSubDirWithConfigure(baseDir string) (string, error) {
	var configureDir string
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 如果找到 configure 文件，返回它的目錄
		if !info.IsDir() && filepath.Base(path) == "configure" {
			configureDir = filepath.Dir(path)
			return filepath.SkipDir
		}
		return nil
	})
	if configureDir == "" {
		return "", fmt.Errorf("configure script not found in %s", baseDir)
	}
	return configureDir, err
}

// 編譯並安裝 GLPK
func buildAndInstallGLPK(configurePath string) {
	// 設置 configure 文件的執行權限
	if err := exec.Command("chmod", "+x", filepath.Join(configurePath, "configure")).Run(); err != nil {
		insyra.LogFatal("lp", "init", "Failed to set configure executable permission: %v", err)
	}

	// 執行 configure
	configureCmd := exec.Command("./configure", "--prefix="+filepath.Join(os.Getenv("HOME"), "local"))
	configureCmd.Dir = configurePath
	if output, err := configureCmd.CombinedOutput(); err != nil {
		insyra.LogFatal("lp", "init", "Failed to configure GLPK: %v, output: %s", err, string(output))
	}

	// 執行 make
	makeCmd := exec.Command("make", "-j"+strconv.Itoa(runtime.NumCPU()))
	makeCmd.Dir = configurePath
	if output, err := makeCmd.CombinedOutput(); err != nil {
		insyra.LogFatal("lp", "init", "Failed to make GLPK: %v, output: %s", err, string(output))
	}

	// 執行 make install，將 GLPK 安裝到用戶目錄
	makeInstallCmd := exec.Command("make", "install")
	makeInstallCmd.Dir = configurePath
	if output, err := makeInstallCmd.CombinedOutput(); err != nil {
		insyra.LogFatal("lp", "init", "Failed to install GLPK: %v, output: %s", err, string(output))
	}

	insyra.LogInfo("lp", "init", "GLPK installed successfully.")
}

// 用於查找常見路徑中的 glpsol
func findInPaths(paths []string) (string, error) {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("glpsol not found in specified paths")
}

// Windows 平台的 GLPK 安裝邏輯
func locateOrInstallGLPK_Win() (string, error) {
	// 首先檢查常見安裝位置
	commonPaths := []string{
		"C:\\glpk\\w64\\glpsol.exe",
		"C:\\glpk\\glpk-*\\w64\\glpsol.exe",
		"C:\\glpk\\glpk-*\\w32\\glpsol.exe",
	}

	for _, pathPattern := range commonPaths {
		matches, err := filepath.Glob(pathPattern)
		if err == nil && len(matches) > 0 {
			// 如果找到多個匹配，使用最新的版本
			sort.Slice(matches, func(i, j int) bool {
				return matches[i] > matches[j]
			})
			insyra.LogInfo("lp", "init", "GLPK found at: %s", matches[0])
			return matches[0], nil
		}
	}

	// 如果沒有找到，嘗試在 PATH 中查找
	glpsolPath, err := exec.LookPath("glpsol.exe")
	if err == nil {
		insyra.LogInfo("lp", "init", "GLPK found in PATH: %s", glpsolPath)
		return glpsolPath, nil
	}

	// 如果還是沒有找到，則下載並安裝 GLPK
	insyra.LogInfo("lp", "init", "GLPK not found, installing...")

	// 下載 GLPK 安裝包
	downloadURL := "https://sourceforge.net/projects/winglpk/files/latest/download"
	zipPath := filepath.Join(os.TempDir(), "glpk.zip")
	insyra.LogDebug("lp", "init", "Downloading GLPK from %s", downloadURL)

	if err := downloadFile(zipPath, downloadURL); err != nil {
		return "", fmt.Errorf("failed to download GLPK: %v", err)
	}

	// 解壓縮
	installDir := "C:\\glpk"
	if err := unzip(zipPath, installDir); err != nil {
		return "", fmt.Errorf("failed to unzip GLPK: %v", err)
	}

	// 查找新安裝的 glpsol.exe
	glpsolPath, err = findGLPKExecutable(installDir)
	if err != nil {
		return "", fmt.Errorf("failed to find GLPK executable after installation: %v", err)
	}

	insyra.LogInfo("lp", "init", "GLPK installed successfully at %s", glpsolPath)
	return glpsolPath, nil
}

// 動態尋找包含 w64 的資料夾
func findGLPKExecutable(baseDir string) (string, error) {
	var glpkExecPath string
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && filepath.Base(path) == "w64" {
			potentialExecPath := filepath.Join(path, "glpsol.exe")
			if _, err := os.Stat(potentialExecPath); err == nil {
				glpkExecPath = potentialExecPath
				return filepath.SkipDir
			}
		} else if err == nil && info.IsDir() && filepath.Base(path) == "w32" {
			potentialExecPath := filepath.Join(path, "glpsol.exe")
			if _, err := os.Stat(potentialExecPath); err == nil {
				glpkExecPath = potentialExecPath
				return filepath.SkipDir
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return glpkExecPath, nil
}

// =========================== 輔助函數 ===========================

// 用於下載文件的輔助函數
func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// 用於解壓 tar.gz 文件的輔助函數
func untar(src string, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	uncompressedStream, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer uncompressedStream.Close()

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // 沒有更多文件了
		}
		if err != nil {
			return err
		}

		fpath := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(fpath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// 用於解壓 zip 文件的輔助函數
func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			err := os.MkdirAll(fpath, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			rc, err := f.Open()
			if err != nil {
				return err
			}

			_, err = io.Copy(outFile, rc)

			outFile.Close()
			rc.Close()

			if err != nil {
				return err
			}
		}
	}
	return nil
}
