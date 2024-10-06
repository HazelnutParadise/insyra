package py

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	pythonVersion = "3.11.10"
	installDir    = ".insyra_py_env"
	port          = "9955"
	backupPort    = "9956"
)

// 從版本號中取得 major 和 minor 版本
var versionParts = strings.Split(pythonVersion, ".")
var pyExec = fmt.Sprintf("python%s.%s", versionParts[0], versionParts[1])

var (
	absInstallDir, _ = filepath.Abs(installDir)
	pyPath           = filepath.Join(absInstallDir, "bin", pyExec)
)
