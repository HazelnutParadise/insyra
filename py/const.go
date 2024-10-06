// py/const.go

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
	pyDependencies   = map[string]string{
		"import requests":                   "pip install requests",       // HTTP requests
		"import json":                       "",                           // JSON data processing (built-in module)
		"import numpy as np":                "pip install numpy",          // Numerical operations
		"import pandas as pd":               "pip install pandas",         // Data analysis and processing
		"import matplotlib.pyplot as plt":   "pip install matplotlib",     // Data visualization
		"import seaborn as sns":             "pip install seaborn",        // Data visualization
		"import scipy":                      "pip install scipy",          // Scientific computing
		"import sklearn":                    "pip install scikit-learn",   // Machine learning
		"import statsmodels.api as sm":      "pip install statsmodels",    // Statistical modeling
		"import plotly.graph_objects as go": "pip install plotly",         // Interactive data visualization
		"import nltk":                       "pip install nltk",           // Natural language processing
		"import spacy":                      "pip install spacy",          // Efficient natural language processing
		"import bs4":                        "pip install beautifulsoup4", // Web scraping
	}
)
