# [ py ] Package

The `py` package allows Golang programs to execute Python code seamlessly and interactively. It provides functionality to pass Go variables into Python scripts, and execute the Python code. Results from the Python script can be sent back to the Go program automatically.

The `py` package automatically installs common Python libraries, allowing you to use them directly in your Python code. You can also install additional dependencies as needed.

## Functions

### `RunCode`

```go
func RunCode(code string) map[string]any
```

This function is used to execute arbitrary Python code. It appends default Python code required for communication and runs the code.

#### Parameters

- `code` (string): The Python code to be executed.

#### Returns

- `map[string]any`: A map representing the result received from the Python server. This map will contain the data returned from Python through the `insyra_return` function. Returns `nil` if Python code doesn't call `insyra_return`.

#### Example

```go
result := RunCode(`
print("Hello from Python")
insyra_return({"message": "Hello from Python", "value": 123})
`)
fmt.Println(result)
```

---

### `RunCodef`

```go
func RunCodef(codeTemplate string, args ...any) map[string]any
```

This function is used to execute Python code with variables passed from Go using `fmt.Sprintf` style formatting. The function formats the code template with the provided arguments.

In the Python code template, use `fmt.Sprintf` style formatting verbs like `%q`, `%d`, `%v`, etc.

#### Parameters

- `codeTemplate` (string): The Python code template with `fmt.Sprintf` style formatting.
- `args` (`...any`): A variable-length argument list of Go variables to be formatted into the template.

#### Returns

- `map[string]any`: A map representing the result received from the Python server. This map will contain the data returned from Python through the `insyra_return` function. Returns `nil` if Python code doesn't call `insyra_return`.

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
 py.RunCodef(`
x = np.array(%v)
y = np.array(%v)

sns.set(style="whitegrid")

sns.scatterplot(x=x, y=y)

plt.title(%q)
plt.xlabel(%q)
plt.ylabel(%q)

plt.show()
`, xData.Data(), yData.Data(), "Scatter Plot from Go DataList", "X Values", "Y Values")
}

```

### `RunFile`

Run Python code from a file.

#### Parameters

- `filepath` (string): The Python file to be executed.

#### Returns

- `map[string]any`: A map representing the result received from the Python server. This map will contain the data returned from Python through the `insyra_return` function. Returns `nil` if Python code doesn't call `insyra_return`.

### `RunFilef`

Run Python code from a file with variables passed from Go using `fmt.Sprintf` style formatting.

#### Parameters

- `filepath` (string): The Python file to be executed.
- `args` (`...any`): A variable-length argument list of Go variables to be formatted into the template.

#### Returns

- `map[string]any`: A map representing the result received from the Python server. This map will contain the data returned from Python through the `insyra_return` function. Returns `nil` if Python code doesn't call `insyra_return`.

### `PipInstall`

```go
func PipInstall(dep string)
```

This function is used to install Python dependencies using uv pip.

#### Parameters

- `dep` (string): The name of the dependency to be installed.

### `PipUninstall`

```go
func PipUninstall(dep string)
```

This function is used to uninstall Python dependencies using uv pip.

#### Parameters

- `dep` (string): The name of the dependency to be uninstalled.

### `ReinstallPyEnv`

```go
func ReinstallPyEnv() error
```

This function reinstalls the Python environment. Useful when you want to reset the Python environment or update dependencies.

#### Returns

- `error`: Returns an error if the reinstallation fails, nil otherwise.

## Concurrency Support

The `py` package now supports concurrent execution of Python code. Multiple goroutines can call `RunCode`, `RunCodef`, `RunFile`, and `RunFilef` simultaneously without interference.

## Functions for Python Code

Here are some functions that are useful when writing Python code to be executed with `RunCode` or `RunCodef`.

### `insyra_return`

```python
insyra_return(data, url)
```

This function is used to return data from Python to Go.

#### Parameters

- `data` (dict): The data to be returned to Go.
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
