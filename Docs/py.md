# [ py ] Package

> [!NOTE]
>
> - **Windows**: This package only works on Windows 10 and above. Needs Visual Studio C++ Build Tools installed.
> - **MacOS**: Needs Xcode installed.
> - **Linux**: Needs following dependencies installed:
>
>  ```sh
>  sudo apt-get update
>  sudo apt-get install build-essential libssl-dev zlib1g-dev libbz2-dev libreadline-dev libsqlite3-dev libffi-dev liblzma-dev wget tar
>  ```

The `py` package allows Golang programs to execute Python code seamlessly and interactively. It provides functionality to pass Go variables into Python scripts, and execute the Python code. Results from the Python script can be sent back to the Go program automatically.

`py` package automatically installs common Python libraries, you can use them directly in your Python code. Installing your own dependencies is also supported.

## Functions

### `RunCode`

```go
func RunCode(code string) map[string]any
```

This function is used to execute arbitrary Python code. It appends default Python code required for communication and runs the code.

#### Parameters

- `code` (string): The Python code to be executed.

#### Returns

- `map[string]any`: A map representing the result received from the Python server. This map will contain the data returned from Python through the `insyra_return` function.

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

This function is used to execute Python code with variables passed from Go. The function automatically serializes the Go variables into JSON and makes them available in the Python code through the `vars` dictionary.

In the Python code template, use `$v1`, `$v2`, `$v3`, etc., as placeholders for the Go variables. These placeholders are replaced by corresponding variables from Go before execution.

#### Parameters

- `codeTemplate` (string): The Python code template where Go variables are passed.
- `args` (`...any`): A variable-length argument list of Go variables to be passed to Python.

#### Returns

- `map[string]any`: A map representing the result received from the Python server. This map will contain the data returned from Python through the `insyra_return` function.

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
x = np.array($v1)
y = np.array($v2)

sns.set(style="whitegrid")

sns.scatterplot(x=x, y=y)

plt.title("Scatter Plot from Go DataList")
plt.xlabel("X Values")
plt.ylabel("Y Values")

plt.show()
`, xData.Data(), yData.Data())
}

```

### `RunFile`

Run Python code from a file.

#### Parameters

- `filepath` (string): The Python file to be executed.

#### Returns

- `map[string]any`: A map representing the result received from the Python server. This map will contain the data returned from Python through the `insyra_return` function.

### `RunFilef`

Run Python code from a file with variables passed from Go.

#### Parameters

- `filepath` (string): The Python file to be executed.
- `args` (`...any`): A variable-length argument list of Go variables to be passed to Python.

#### Returns

- `map[string]any`: A map representing the result received from the Python server. This map will contain the data returned from Python through the `insyra_return` function.

### `PipInstall`

```go
func PipInstall(dep string)
```

This function is used to install Python dependencies using pip.

#### Parameters

- `dep` (string): The name of the dependency to be installed.

### `PipUninstall`

```go
func PipUninstall(dep string)
```

This function is used to uninstall Python dependencies using pip.

#### Parameters

- `dep` (string): The name of the dependency to be uninstalled.

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

- **Python Environment**: Insyra automatically installs Python environment in the `.insyra_py_env` directory in your project root.
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
