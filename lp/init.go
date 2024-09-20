package lp

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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
		installOnLinux()
	case "darwin":
		installOnMacOS()
	case "windows":
		installOnWindows()
	default:
		log.Println("Unsupported operating system.")
	}
}

// Linux 平台的安裝邏輯
func installOnLinux() {
	cmd := exec.Command("which", "glpsol")
	err := cmd.Run()
	if err != nil {
		log.Println("GLPK not found, installing on Linux...")
		cmd = exec.Command("sudo", "apt-get", "install", "-y", "glpk-utils")
		err = cmd.Run()
		if err != nil {
			insyra.LogFatal("lp.init: Failed to install GLPK on Linux: %v", err)
		}
	} else {
		insyra.LogInfo("lp.init: GLPK already installed.")
	}
}

// macOS 平台的安裝邏輯
func installOnMacOS() {
	cmd := exec.Command("which", "glpsol")
	err := cmd.Run()
	if err != nil {
		insyra.LogInfo("lp.init: GLPK not found, installing on macOS...")
		cmd = exec.Command("brew", "install", "glpk")
		err = cmd.Run()
		if err != nil {
			insyra.LogFatal("lp.init: Failed to install GLPK on macOS: %v", err)
		}
	} else {
		insyra.LogInfo("lp.init: GLPK already installed.")
	}
}

// Windows 平台的安裝邏輯
func installOnWindows() {
	// 檢查 glpsol 是否已經安裝
	cmd := exec.Command("where", "glpsol.exe")
	err := cmd.Run()
	if err != nil {
		insyra.LogInfo("lp.init: GLPK not found, installing on Windows...")

		// 下載 GLPK 安裝包
		downloadURL := "https://sourceforge.net/projects/winglpk/files/latest/download"
		zipPath := filepath.Join(os.TempDir(), "glpk.zip")
		insyra.LogInfo("lp.init: Downloading GLPK from %s\n", downloadURL)

		err := downloadFile(zipPath, downloadURL)
		if err != nil {
			insyra.LogFatal("lp.init: Failed to download GLPK: %v", err)
		}

		// 解壓縮並設置路徑
		installDir := "C:\\glpk"
		err = unzip(zipPath, installDir)
		if err != nil {
			insyra.LogFatal("lp.init: Failed to unzip GLPK: %v", err)
		}

		// 動態查找包含 "w64" 的資料夾
		glpkPath := findGLPKExecutable(installDir)
		if glpkPath == "" {
			insyra.LogFatal("lp.init: Failed to find GLPK executable after extraction.")
		}

		insyra.LogInfo("lp.init: Adding GLPK to PATH.")
		os.Setenv("PATH", os.Getenv("PATH")+";"+filepath.Dir(glpkPath))

		// 檢查是否可執行
		if _, err := exec.LookPath(glpkPath); err != nil {
			insyra.LogFatal("lp.init: GLPK executable not found even after adding to PATH: %v", err)
		}
	} else {
		insyra.LogInfo("lp.init: GLPK already installed.")
	}
}

// 動態尋找包含 w64 的資料夾
func findGLPKExecutable(baseDir string) string {
	var glpkExecPath string
	filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && filepath.Base(path) == "w64" {
			potentialExecPath := filepath.Join(path, "glpsol.exe")
			if _, err := os.Stat(potentialExecPath); err == nil {
				glpkExecPath = potentialExecPath
				return filepath.SkipDir
			}
		}
		return nil
	})
	return glpkExecPath
}

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

// 用於解壓縮的輔助函數
func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
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
