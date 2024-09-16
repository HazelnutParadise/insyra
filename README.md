# Insyra

[![Test](https://github.com/HazelnutParadise/insyra/actions/workflows/test.yml/badge.svg)](https://github.com/HazelnutParadise/insyra/actions/workflows/test.yml)

A Golang data analysis library.

**[繁體中文版 README](README_TW.md)**

## Overview

The **Insyra** library is a dynamic and versatile tool designed for managing and analyzing data in Go. It offers a rich set of features for data manipulation, statistical calculations, and more, making it an essential toolkit for developers handling complex data structures.

## Important Note

For any functions or methods not explicitly listed in Insyra documents, it indicates that the feature is still under active development. These experimental features might provide unstable results. 

Please refer to our latest updates in **[Docs](/Docs)** folder for more details.


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

- **[stats](/Docs/stats.md)**: Provides statistical functions for data analysis, including skewness, kurtosis, and moment calculations.
- **[parallel](/Docs/parallel.md)**: Offers parallel processing capabilities for data manipulation and analysis.
- **[plot](/Docs/plot.md)**: Provides a wrapper around the powerful [github.com/go-echarts/go-echarts](https://github.com/go-echarts/go-echarts) library, designed to simplify data visualization.

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

