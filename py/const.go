// py/const.go

package py

import (
	"path/filepath"
	"runtime"
)

var (
	pythonVersion = "3.12.*"
	installDir    = filepath.Join(".insyra_env", "py25c_"+runtime.GOOS+"_"+runtime.GOARCH)
	port          = "9955"
	backupPort    = "9956"
)

var (
	// uvInstallCmd     []string
	absInstallDir, _ = filepath.Abs(installDir)
	pyPath           string // 將在pyEnvInit中設置
	pyDependencies   = map[string]string{
		"import requests":                   "requests",       // HTTP requests
		"import json":                       "",               // JSON data processing (built-in module)
		"import numpy as np":                "numpy",          // Numerical operations
		"import pandas as pd":               "pandas",         // Data analysis and processing
		"import polars as pl":               "polars",         // Data analysis and processing (faster alternative to pandas)
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
