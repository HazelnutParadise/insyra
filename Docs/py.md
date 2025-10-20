# [ py ] Package

The `py` package allows Golang programs to execute Python code seamlessly and interactively. It provides functionality to pass Go variables into Python scripts, and execute the Python code. Results from the Python script can be sent back to the Go program automatically.

The `py` package automatically installs common Python libraries, allowing you to use them directly in your Python code. You can also install additional dependencies as needed.

## Functions

### `RunCode`

```go
func RunCode(out any, code string) error
```

This function is used to execute arbitrary Python code and bind the result to the provided struct pointer. It appends default Python code required for communication and runs the code.

#### Parameters

- `out` (any): A pointer to a struct to bind the Python result to. You can optionally use `json` tags on struct fields for custom mapping, otherwise field names are matched to Python dictionary keys by default. If you don't need the Python return result, pass `nil`.
- `code` (string): The Python code to be executed.

#### Returns

- `error`: Returns an error if execution failed or result binding failed, `nil` otherwise.

#### Example

```go
type ResultData struct {
    Message string `json:"message"`
    Value   int    `json:"value"`
}

var result ResultData
err := RunCode(&result, `
print("Hello from Python")
insyra.Return({"message": "Hello from Python", "value": 123})
`)
if err != nil {
    fmt.Println("Error:", err)
} else {
    fmt.Println(result)
}
```

---

### `RunCodef`

```go
func RunCodef(out any, code string, args ...any) error
```

This function is used to execute Python code with variables passed from Go using `$v1`, `$v2`, etc. placeholders and bind the result to the provided struct pointer. The function replaces these placeholders with the provided arguments.

In the Python code template, use `$v1`, `$v2`, `$v3`, etc. as placeholders for the arguments passed to the function.

#### Parameters

- `out` (any): A pointer to a struct to bind the Python result to. You can optionally use `json` tags on struct fields for custom mapping, otherwise field names are matched to Python dictionary keys by default. If you don't need the Python return result, pass `nil`.
- `code` (string): The Python code template with `$v1`, `$v2`, etc. placeholders.
- `args` (`...any`): A variable-length argument list of Go variables to be substituted into the template.

#### Returns

- `error`: Returns an error if execution failed or result binding failed, `nil` otherwise.

#### Example

```go
package main

import (
 "fmt"
 "github.com/HazelnutParadise/insyra"
 "github.com/HazelnutParadise/insyra/py"
)

type PlotResult struct {
 Success bool   `json:"success"`
 Message string `json:"message"`
}

func main() {
 // Create DataList
 xData := insyra.NewDataList(45, 50, 55, 60, 65, 70, 75, 80, 85, 90)
 yData := insyra.NewDataList(110, 120, 135, 145, 150, 160, 170, 180, 190, 200)

 // Submit Code to Python
 var result PlotResult
 err := py.RunCodef(&result, `
x = $v1
y = $v2

sns.set(style="whitegrid")
sns.scatterplot(x=x, y=y)

plt.title($v3)
plt.xlabel($v4)
plt.ylabel($v5)

plt.show()
insyra.Return({"success": True, "message": "Plot created"})
`, xData.Data(), yData.Data(), "Scatter Plot from Go DataList", "X Values", "Y Values")
 if err != nil {
     fmt.Println("Error:", err)
 } else {
     fmt.Println("Result:", result)
 }
}
```

### `RunFile`

Run Python code from a file and bind the result to the provided struct pointer.

#### Parameters

- `out` (any): A pointer to a struct to bind the Python result to. You can optionally use `json` tags on struct fields for custom mapping, otherwise field names are matched to Python dictionary keys by default. If you don't need the Python return result, pass `nil`.
- `filepath` (string): The Python file to be executed.

#### Returns

- `error`: Returns an error if execution failed or result binding failed, `nil` otherwise.

### `RunFilef`

Run Python code from a file with variables passed from Go using `$v1`, `$v2`, etc. placeholders and bind the result to the provided struct pointer.

#### Parameters

- `out` (any): A pointer to a struct to bind the Python result to. You can optionally use `json` tags on struct fields for custom mapping, otherwise field names are matched to Python dictionary keys by default. If you don't need the Python return result, pass `nil`.
- `filepath` (string): The Python file to be executed.
- `args` (`...any`): A variable-length argument list of Go variables to be substituted into the template.

#### Returns

- `error`: Returns an error if execution failed or result binding failed, `nil` otherwise.

### `PipInstall`

```go
func PipInstall(dep string)
```

This function is used to install Python dependencies using uv pip. It logs the result and terminates the program if installation fails.

#### Parameters

- `dep` (string): The name of the dependency to be installed.

#### Notes

- This function does not return an error; instead, it logs fatal errors directly.
- The program will terminate if the installation fails.

### `PipUninstall`

```go
func PipUninstall(dep string)
```

This function is used to uninstall Python dependencies using uv pip. It logs the result and terminates the program if uninstallation fails.

#### Parameters

- `dep` (string): The name of the dependency to be uninstalled.

#### Notes

- This function does not return an error; instead, it logs fatal errors directly.
- The program will terminate if the uninstallation fails.

### `ReinstallPyEnv`

```go
func ReinstallPyEnv() error
```

This function completely reinstalls the Python environment by removing the existing virtual environment and reinstalling all dependencies. Useful when you want to reset the Python environment, update dependencies, or fix environment-related issues.

#### Returns

- `error`: Returns an error if the reinstallation fails, `nil` otherwise.

#### Example

```go
package main

import (
 "fmt"
 "github.com/HazelnutParadise/insyra/py"
)

func main() {
 err := py.ReinstallPyEnv()
 if err != nil {
     fmt.Println("Failed to reinstall Python environment:", err)
 } else {
     fmt.Println("Python environment reinstalled successfully!")
 }
}
```

## Concurrency Support

The `py` package supports concurrent execution of Python code. Multiple goroutines can call `RunCode`, `RunCodef`, `RunFile`, and `RunFilef` simultaneously without interference. Each execution gets a unique ID and processes independently.

## Functions for Python Code

Here are some functions that are useful when writing Python code to be executed with `RunCode` or `RunCodef`.

### `insyra.Return`

```python
insyra.Return(result=None, error=None, url)
```

This function is used to return data from Python to Go.

#### Parameters

- `result` (any): The result data to be returned to Go.
- `error` (string): The error message if an error occurred, None otherwise. The Insyra framework will automatically deal with errors, you don't need to set it manually.
- `url` (string): The URL to send the data to. Insyra will automatically set it, you don't need to set it manually.

#### Example

```python
insyra.Return({
 "message": "Hello from Python",
 "value": 123,
})
```

### `insyra_return`

```python
insyra_return(result=None, error=None, url)
```

This is an alias for `insyra.Return` for convenience. It provides the same functionality as `insyra.Return`.

#### Parameters

- `result` (any): The result data to be returned to Go.
- `error` (string): The error message if an error occurred, None otherwise. The Insyra framework will automatically deal with errors, you don't need to set it manually.
- `url` (string): The URL to send the data to. Insyra will automatically set it, you don't need to set it manually.

#### Example

```python
insyra_return({
 "message": "Hello from Python",
 "value": 123,
})
```

### `insyra.execution_id`

```python
insyra.execution_id
```

This variable contains the unique execution ID for the current Python code run. It can be used for logging, debugging, or tracking purposes.

#### Example

```python
print(f"Current execution ID: {insyra.execution_id}")
insyra.Return({"execution_id": insyra.execution_id, "data": "some data"})
```

## Pre-installed Dependencies

- **Python Environment**: Insyra automatically installs Python environment using `uv` in the `.insyra_py_env` directory in your project root.
- **Python Libraries**: Insyra automatically installs following Python libraries, you can use them directly in your Python code:

 ``` go
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
 ```
