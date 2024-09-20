# Insyra

[![Test](https://github.com/HazelnutParadise/insyra/actions/workflows/test.yml/badge.svg)](https://github.com/HazelnutParadise/insyra/actions/workflows/test.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/HazelnutParadise/insyra.svg)](https://github.com/HazelnutParadise/insyra)
[![Go Report Card](https://goreportcard.com/badge/github.com/HazelnutParadise/insyra)](https://goreportcard.com/report/github.com/HazelnutParadise/insyra)
[![GoDoc](https://godoc.org/github.com/HazelnutParadise/insyra?status.svg)](https://pkg.go.dev/github.com/HazelnutParadise/insyra)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)


A next-generation data analysis library for Golang.

![logo](logo/logo_transparent.png)


**[繁體中文版 README](README_TW.md)**

## Too Fast, Too Lovely, Too Easy To Use.

The **Insyra** library is a dynamic and versatile tool designed for managing and analyzing data in Go. It offers a rich set of features for data manipulation, statistical calculations, data visualization, and more, making it an essential toolkit for developers handling complex data structures.

> [!NOTE]
> If some functions or methods in the documentation are not working, it may be because the feature is not yet included in the latest release. Please refer to the documentation in the corresponding version code in **[Releases](https://github.com/HazelnutParadise/insyra/releases)**.

> [!IMPORTANT]
> **For any functions or methods not explicitly listed in Insyra documents, it indicates that the feature is still under active development. These experimental features might provide unstable results.** 
>
> Please refer to our latest updates in **[Docs](/Docs)** folder for more details.


## Getting Started

### Installation

To start using **Insyra**, install it with the following command:

```sh
go get github.com/HazelnutParadise/insyra
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

## DataList

The `DataList` is the core structure in **Insyra**, enabling the storage, management, and analysis of dynamic data collections. It offers various methods for data manipulation and statistical analysis. 

For a complete list of methods and features, please refer to the **[DataList Documentation](/Docs/DataList.md)**.

## DataTable

The `DataTable` structure provides a tabular data representation, allowing for the storage and manipulation of data in a structured format. It offers methods for data filtering, sorting, and aggregation, making it a powerful tool for data analysis.

**You can also convert between DataTables and CSV files with simply one line of code, enabling seamless integration with external data sources.**

For a complete list of methods and features, please refer to the **[DataTable Documentation](/Docs/DataTable.md)**.

## Packages

**Insyra** also provides several expansion packages, each focusing on a specific aspect of data analysis.

### **[stats](/Docs/stats.md)**

Provides statistical functions for data analysis, including skewness, kurtosis, and moment calculations.

### **[parallel](/Docs/parallel.md)**
Offers parallel processing capabilities for data manipulation and analysis. Allows you to execute any function and automatically wait for all goroutines to complete.

### **[plot](/Docs/plot.md)**

Provides a wrapper around the powerful [github.com/go-echarts/go-echarts](https://github.com/go-echarts/go-echarts) library, designed to simplify data visualization.

### **[gplot](/Docs/gplot.md)**

A visualization package based on [github.com/gonum/plot](https://github.com/gonum/plot). Fast and no need for Chrome.

### **[lpgen](/Docs/lpgen.md)**

Provides a **super simple** and intuitive way to generate linear programming (LP) models and save them as `.lp` files. It supports setting objectives, adding constraints, defining variable bounds, and specifying binary or integer variables.

### **[lp](/Docs/lp.md)**

Fully automatic linear programming (LP) solver using [GLPK](https://www.gnu.org/software/glpk/).

## Advanced Usage

Beyond basic usage, **Insyra** provides extensive capabilities for handling different data types and performing complex statistical operations. Explore more in the **[detailed documentation](/Docs)**.

## Contributing

Contributions are welcome! You can contribute to **Insyra** by:
- **Issues:** Reporting issues or suggesting new features.
- **Pull Requests:** Submitting pull requests to enhance the library.
- **Discussions:** Sharing your feedback and ideas to improve the project.
<!-- For more details, see the [contributing guidelines](https://github.com/HazelnutParadise/insyra/blob/main/CONTRIBUTING.md). -->

## Contributors
![contributors](https://contrib.rocks/image?repo=HazelnutParadise/insyra)

## License

Insyra is licensed under the MIT License. See the [LICENSE](LICENSE) file for more information.

