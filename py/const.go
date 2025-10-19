// py/const.go

package py

import (
	"path/filepath"
)

const (
	pythonVersion = "3.12.9"
	installDir    = "insyra_py25b"
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
