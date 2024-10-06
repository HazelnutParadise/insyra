// py/const.go

package py

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	pythonVersion = "3.11.9"
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
	pyDependencies   = map[string]string{
		"import requests":                   "requests",       // HTTP requests
		"import json":                       "",               // JSON data processing (built-in module)
		"import numpy as np":                "numpy",          // Numerical operations
		"import pandas as pd":               "pandas",         // Data analysis and processing
		"import matplotlib.pyplot as plt":   "matplotlib",     // Data visualization
		"import seaborn as sns":             "seaborn",        // Data visualization
		"import scipy":                      "scipy",          // Scientific computing
		"import sklearn":                    "scikit-learn",   // Machine learning
		"import statsmodels.api as sm":      "statsmodels",    // Statistical modeling
		"import plotly.graph_objects as go": "plotly",         // Interactive data visualization
		"import spacy":                      "spacy",          // Efficient natural language processing
		"import bs4":                        "beautifulsoup4", // Web scraping
	}
)
