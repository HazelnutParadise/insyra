# Insyra

[![Test](https://github.com/HazelnutParadise/insyra/actions/workflows/test.yml/badge.svg)](https://github.com/HazelnutParadise/insyra/actions/workflows/test.yml)
[![GolangCI-Lint](https://github.com/HazelnutParadise/insyra/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/HazelnutParadise/insyra/actions/workflows/golangci-lint.yml)
[![Govulncheck](https://github.com/HazelnutParadise/insyra/actions/workflows/govulncheck.yml/badge.svg)](https://github.com/HazelnutParadise/insyra/actions/workflows/govulncheck.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/HazelnutParadise/insyra.svg)](https://github.com/HazelnutParadise/insyra)
[![Go Report Card](https://goreportcard.com/badge/github.com/HazelnutParadise/insyra)](https://goreportcard.com/report/github.com/HazelnutParadise/insyra)
[![GoDoc](https://godoc.org/github.com/HazelnutParadise/insyra?status.svg)](https://pkg.go.dev/github.com/HazelnutParadise/insyra)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)


A next-generation data analysis library for Golang. Supports **parallel processing**, **data visualization**, and **seamless integration with Python**.

**Official Website: https://insyra.hazelnut-paradise.com**

![logo](logo/logo_transparent.png)


**[繁體中文版 README](README_TW.md)**

Welcome to join [**Side Project Taiwan**(Discord Server)](https://discord.com/channels/1205906503073140776/1280539658551558368) for further discussions.

## Fast, Lovely, Easy To Use.

The **Insyra** library is a dynamic and versatile tool designed for managing and analyzing data in Go. It offers a rich set of features for data manipulation, statistical calculations, data visualization, and more, making it an essential toolkit for developers handling complex data structures.

> [!NOTE]
> If some functions or methods in the documentation are not working, it may be because the feature is not yet included in the latest release. Please refer to the documentation in the source code of the corresponding version in **[Releases](https://github.com/HazelnutParadise/insyra/releases)**.

> [!IMPORTANT]
> **For any functions or methods not explicitly listed in Insyra documents, it indicates that the feature is still under active development. These experimental features might provide unstable results.** 
>
> Please refer to our latest updates in **[Docs](/Docs)** folder for more details.

## [Idensyra](https://github.com/HazelnutParadise/idensyra)
We provide a mini Go IDE, `Idensyra`, which aims to make data analysis even more easier (though Insyra has already made it very easy).

`Idensyra` comes with Insyra pre-installed, and allows you to run Go code without installing Go environment!

**[Know more about Idensyra](https://github.com/HazelnutParadise/idensyra)**

## Getting Started

### For those new to Golang

> [!TIP]
> Jump to [Installation](#installation) or [Quick Example](#quick-example) if you are familiar with Go.

1. Download and install Golang from [here](https://golang.org/dl/).
2. Set up your editor, we recommend using [VSCode](https://code.visualstudio.com/). Or even lighter weight, [Idensyra](https://github.com/HazelnutParadise/idensyra).
3. Open or create a folder for your project, and open it in the editor.

4. Create a new project by running the following command:

    ```sh
    go mod init your_project_name
    ```

5. Install **Insyra**:

    ```sh
    go get github.com/HazelnutParadise/insyra
    ```

6. Create a new file, e.g., `main.go`, and write the following code:

    ```go
    package main

    import (
        "fmt"
        "github.com/HazelnutParadise/insyra"
    )

    func main() {
        // Your code here
    }
    ```

7. Run your project:

    ```sh
    go run main.go
    ```

### Installation

- To start using **Insyra**, install it with the following command:

    ```sh
    go get github.com/HazelnutParadise/insyra
    ```

- Update **Insyra** to the latest version:

    ```sh
    go get -u github.com/HazelnutParadise/insyra
    ```

    or

    ```sh
    go get -u github.com/HazelnutParadise/insyra@latest
    ```

### Quick Example

```go
package main

import (
    "fmt"
    "github.com/HazelnutParadise/insyra"
)

func main() {
    dl := insyra.NewDataList(1, 2, 3, 4, 5)
    dl.Append(6)
    fmt.Println("DataList:", dl.Data())
    fmt.Println("Mean:", dl.Mean())
}
```

## [DataList](/Docs/DataList.md)

The `DataList` is the core structure in **Insyra**, enabling the storage, management, and analysis of dynamic data collections. It offers various methods for data manipulation and statistical analysis. 

For a complete list of methods and features, please refer to the **[DataList Documentation](/Docs/DataList.md)**.

## [DataTable](/Docs/DataTable.md)

The `DataTable` structure provides a tabular data representation, allowing for the storage and manipulation of data in a structured format. It offers methods for data filtering, sorting, and aggregation, making it a powerful tool for data analysis.

**You can also convert between DataTables and CSV files with simply one line of code, enabling seamless integration with external data sources.**

For a complete list of methods and features, please refer to the **[DataTable Documentation](/Docs/DataTable.md)**.

## Packages

**Insyra** also provides several expansion packages, each focusing on a specific aspect of data analysis.

### **[datafetch](/Docs/datafetch.md)**

Allows you to fetch data easily. It currently supports fetching comments from stores on Google Maps.

### **[stats](/Docs/stats.md)**

Provides statistical functions for data analysis, including skewness, kurtosis, and moment calculations.

### **[parallel](/Docs/parallel.md)**
Offers parallel processing capabilities for data manipulation and analysis. Allows you to execute any function and automatically wait for all goroutines to complete.

### **[plot](/Docs/plot.md)**

Provides a wrapper around the powerful [github.com/go-echarts/go-echarts](https://github.com/go-echarts/go-echarts) library, designed to simplify data visualization.

### **[gplot](/Docs/gplot.md)**

A visualization package based on [github.com/gonum/plot](https://github.com/gonum/plot). Fast and no need for Chrome. Even supports function plot.

### **[lpgen](/Docs/lpgen.md)**

Provides a **super simple** and intuitive way to generate linear programming (LP) models and save them as `.lp` files. It supports setting objectives, adding constraints, defining variable bounds, and specifying binary or integer variables.

### **[lp](/Docs/lp.md)**

Fully automatic linear programming (LP) solver using [GLPK](https://www.gnu.org/software/glpk/).

### **[csvxl](/Docs/csvxl.md)**

Work with Excel and CSV files. Such as convert CSV to Excel.

### **[py](/Docs/py.md)**

Execute Python code in Go without manually installing Python environment and dependencies. Allows passing variables between Go and Python.

## Advanced Usage

Beyond basic usage, **Insyra** provides extensive capabilities for handling different data types and performing complex statistical operations. Explore more in the **[detailed documentation](/Docs)**.

## Contributing

Contributions are welcome! You can contribute to **Insyra** by:
- **[Issues](https://github.com/HazelnutParadise/insyra/issues):** Reporting issues or suggesting new features.
- **[Pull Requests](https://github.com/HazelnutParadise/insyra/pulls):** Submitting pull requests to enhance the library.
- **[Discussions](https://github.com/HazelnutParadise/insyra/discussions):** Sharing your feedback and ideas to improve the project.
<!-- For more details, see the [contributing guidelines](https://github.com/HazelnutParadise/insyra/blob/main/CONTRIBUTING.md). -->

## Contributors
[![contributors](https://contrib.rocks/image?repo=HazelnutParadise/insyra)](https://github.com/HazelnutParadise/insyra/contributors)

## License

Insyra is licensed under the MIT License. See the [LICENSE](LICENSE) file for more information.

