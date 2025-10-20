# [ py ] Package

The `py` package allows Golang programs to execute Python code seamlessly and interactively. It provides functionality to pass Go variables into Python scripts, and execute the Python code. Results from the Python script can be sent back to the Go program automatically.

The `py` package automatically installs common Python libraries, allowing you to use them directly in your Python code. You can also install additional dependencies as needed.

## Functions

### `RunCode`

```go
func RunCode(code string) (map[string]any, error)
```

This function is used to execute arbitrary Python code. It appends default Python code required for communication and runs the code.

#### Parameters

- `code` (string): The Python code to be executed.

#### Returns

- `(map[string]any, error)`: A map containing the data returned from Python through the `insyra_return` function, and an error if execution failed.
  - On success: Returns the result data as a map and `nil` for error
  - On failure: Returns `nil` and an error message containing either the exception from Python or system execution error
  - If Python doesn't call `insyra_return`: Returns `nil, nil`

#### Example

```go
result, err := RunCode(`
print("Hello from Python")
insyra_return({"message": "Hello from Python", "value": 123})
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
func RunCodef(codeTemplate string, args ...any) (map[string]any, error)
```

This function is used to execute Python code with variables passed from Go using `$v1`, `$v2`, etc. placeholders. The function replaces these placeholders with the provided arguments.

In the Python code template, use `$v1`, `$v2`, `$v3`, etc. as placeholders for the arguments passed to the function.

#### Parameters

- `codeTemplate` (string): The Python code template with `$v1`, `$v2`, etc. placeholders.
- `args` (`...any`): A variable-length argument list of Go variables to be substituted into the template.

#### Returns

- `(map[string]any, error)`: A map containing the data returned from Python through the `insyra_return` function, and an error if execution failed.
  - On success: Returns the result data as a map and `nil` for error
  - On failure: Returns `nil` and an error message containing either the exception from Python or system execution error
  - If Python doesn't call `insyra_return`: Returns `nil, nil`

#### Example

```go
package main

import (
 "github.com/HazelnutParadise/insyra"
 "github.com/HazelnutParadise/insyra/py"
)

func main() {
 // Create DataList
 xData := insyra.NewDataList(45, 50, 55, 60, 65, 70, 75, 80, 85, 90)
 yData := insyra.NewDataList(110, 120, 135, 145, 150, 160, 170, 180, 190, 200)

 // Submit Code to Python
 result, err := py.RunCodef(`
x = $v1
y = $v2

sns.set(style="whitegrid")

sns.scatterplot(x=x, y=y)

plt.title($v3)
plt.xlabel($v4)
plt.ylabel($v5)

plt.show()
`, xData.Data(), yData.Data(), "Scatter Plot from Go DataList", "X Values", "Y Values")
 if err != nil {
     fmt.Println("Error:", err)
 }
}

```

### `RunFile`

Run Python code from a file.

#### Parameters

- `filepath` (string): The Python file to be executed.

#### Returns

- `(map[string]any, error)`: A map containing the data returned from Python through the `insyra_return` function, and an error if execution failed.
  - On success: Returns the result data as a map and `nil` for error
  - On failure: Returns `nil` and an error message containing either the exception from Python or system execution error
  - If Python doesn't call `insyra_return`: Returns `nil, nil`

### `RunFilef`

Run Python code from a file with variables passed from Go using `$v1`, `$v2`, etc. placeholders.

#### Parameters

- `filepath` (string): The Python file to be executed.
- `args` (`...any`): A variable-length argument list of Go variables to be substituted into the template.

#### Returns

- `(map[string]any, error)`: A map containing the data returned from Python through the `insyra_return` function, and an error if execution failed.
  - On success: Returns the result data as a map and `nil` for error
  - On failure: Returns `nil` and an error message containing either the exception from Python or system execution error
  - If Python doesn't call `insyra_return`: Returns `nil, nil`

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

The `py` package now supports concurrent execution of Python code. Multiple goroutines can call `RunCode`, `RunCodef`, `RunFile`, and `RunFilef` simultaneously without interference.

## Functions for Python Code

Here are some functions that are useful when writing Python code to be executed with `RunCode` or `RunCodef`.

### `insyra_return`

```python
insyra_return(result=None, error=None, url)
```

This function is used to return data from Python to Go.

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

## Pre-installed Dependencies

- **Python Environment**: Insyra automatically installs Python environment using `uv` in the `.insyra_py_env` directory in your project root.
- **Python Libraries**: Insyra automatically installs following Python libraries, you can use them directly in your Python code:
 ``` go
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
 ```
