// py/const.go

package py

import (
	"path/filepath"
	"runtime"
)

// envDirCode names the environment directory. It changes whenever the pinned
// Python version does, as py<two-digit year><letter>: a for the year's first
// new code, b for the second. A change of package versions alone keeps it,
// and the environment is synced in place.
const envDirCode = "py26a"

// envDirPython is the Python version envDirCode was given for. A test fails
// when the pinned Python differs, so a Python bump cannot keep the directory.
const envDirPython = "3.12.14"

var (
	installDir = filepath.Join(".insyra_env", envDirCode+"_"+runtime.GOOS+"_"+runtime.GOARCH)
)

var (
	absInstallDir, _ = filepath.Abs(installDir)
	pyPath           string // the environment's interpreter, set by pyEnvInit
	uvPath           string // the pinned uv, set by pyEnvInit
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
