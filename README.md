# Insyra

A Golang data analysis library.

## Overview

The **Insyra** library is a dynamic and versatile tool designed for managing and analyzing data in Go. It offers a rich set of features for data manipulation, statistical calculations, and more, making it an essential toolkit for developers handling complex data structures.

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

For a complete list of methods and features, please refer to the **[DataList Documentation](https://github.com/HazelnutParadise/insyra/tree/main/Docs/DataList.md)**.

## Advanced Usage

Beyond basic usage, **Insyra** provides extensive capabilities for handling different data types and performing complex statistical operations. Explore more in the **[detailed documentation](https://github.com/HazelnutParadise/insyra/tree/main/Docs)**.

<!-- ## Contributing

Contributions are welcome! For more details, see the [contributing guidelines](https://github.com/HazelnutParadise/insyra/blob/main/CONTRIBUTING.md). -->

## License

Insyra is licensed under the MIT License. See the [LICENSE](https://github.com/HazelnutParadise/insyra/blob/main/LICENSE) file for more information.

---

This documentation provides an overview and quick start guide, emphasizing the importance of referring to the `DataList` documentation for detailed information.
