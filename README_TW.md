# Insyra

[![Test](https://github.com/HazelnutParadise/insyra/actions/workflows/test.yml/badge.svg)](https://github.com/HazelnutParadise/insyra/actions/workflows/test.yml)
[![GolangCI-Lint](https://github.com/HazelnutParadise/insyra/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/HazelnutParadise/insyra/actions/workflows/golangci-lint.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/HazelnutParadise/insyra.svg)](https://github.com/HazelnutParadise/insyra)
[![Go Report Card](https://goreportcard.com/badge/github.com/HazelnutParadise/insyra)](https://goreportcard.com/report/github.com/HazelnutParadise/insyra)
[![GoDoc](https://godoc.org/github.com/HazelnutParadise/insyra?status.svg)](https://pkg.go.dev/github.com/HazelnutParadise/insyra)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)

Go 語言次世代資料分析庫。支援 **平行處理**、**資料視覺化**，並 **與 Python 無縫整合**。

![logo](logo/logo.webp)

歡迎加入 [**Side Project Taiwan**(Discord 社群)](https://discord.com/channels/1205906503073140776/1280539658551558368) 與我們一起討論。

## 太快、太美、太簡單。

**Insyra** 庫是一個動態且多功能的 Go 語言資料分析工具。提供了豐富的功能集，可用於數據操作、統計計算、資料視覺化等，對於處理複雜數據結構的開發者來說，是一個必不可少的工具包。

> [!NOTE]
> 如果文檔中的某些功能無法使用，可能是該功能還未包含在最新發布的版本中。請至 **[Releases](https://github.com/HazelnutParadise/insyra/releases)** 查看對應版本源碼中的文檔。

> [!IMPORTANT]
> **對於 Insyra 文檔中未明確列出的任何函數或方法，表示該功能仍在積極開發中。這些實驗性功能可能會提供不穩定的結果。**    
>
> 請參閱我們 **[文檔](https://github.com/HazelnutParadise/insyra/tree/main/Docs)** 資料夾中的最新更新以獲取更多詳細資訊。

## [Idensyra](https://github.com/HazelnutParadise/idensyra)

我們提供了一個迷你 Go IDE，`Idensyra`，旨在使數據分析變得更簡單（儘管 Insyra 已經使其非常簡單）。

`Idensyra` 預裝了 Insyra，不需要安裝 Go 環境即可運行 Go 程式碼！

**[了解更多關於 Idensyra](https://github.com/HazelnutParadise/idensyra)**

## 開始使用

### 致初學者

> [!TIP]
> 如果您已熟悉 Go，請跳至 [安裝](#安裝) 或 [快速範例](#快速範例)。

1. 從 [這裡](https://golang.org/dl/) 下載並安裝 Golang。
2. 設置您的編輯器，我們推薦使用 [VSCode](https://code.visualstudio.com/)。
3. 在您的專案中開啟或建立一個資料夾，並在編輯器中開啟它。

4. 使用以下命令建立新專案：

    ```sh
    go mod init your_project_name
    ```

5. 在您的專案中安裝 **Insyra**：

    ```sh
    go get github.com/HazelnutParadise/insyra
    ```

6. 建立一個新文件，例如 `main.go`，並寫入以下代碼：

    ```go
    package main

    import (
        "fmt"
        "github.com/HazelnutParadise/insyra"
    )

    func main() {
        // 將您的代碼寫在這裡
    }
    ```

7. 運行您的專案：

    ```sh
    go run main.go
    ```

### 安裝

- 要開始使用 **Insyra**，請使用以下命令進行安裝：

    ```sh
    go get github.com/HazelnutParadise/insyra
    ```

- 更新 **Insyra** 到最新版本：

    ```sh
    go get -u github.com/HazelnutParadise/insyra
    ```

    或者

    ```sh
    go get github.com/HazelnutParadise/insyra@latest
    ```


### 快速範例

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

`DataList` 是 **Insyra** 的核心結構，能夠存儲、管理和分析動態數據集合。它提供了各種用於數據操作和統計分析的方法。

有關方法和功能的完整列表，請參閱 **[DataList 文檔](https://github.com/HazelnutParadise/insyra/tree/main/Docs/DataList.md)**。

## [DataTable](/Docs/DataTable.md)

`DataTable` 結構提供了表格數據的表示方式，允許以結構化格式存儲和操作數據。它提供了數據過濾、排序和聚合的方法，使其成為數據分析的強大工具。

**您還可以僅用一行代碼在 DataTables 和 CSV 文件之間進行轉換，實現與外部數據源的無縫整合。**

有關方法和功能的完整列表，請參閱 **[DataTable 文檔](https://github.com/HazelnutParadise/insyra/tree/main/Docs/DataTable.md)**。

## 套件

**Insyra** 還提供了多個擴展套件，每個都專注於數據分析的特定方面。

### **[stats](/Docs/stats.md)**

提供數據分析的統計函數，包括偏度、峰度和矩計算。

### **[parallel](/Docs/parallel.md)**

為數據操作和分析提供平行處理能力。可用於執行任何函數，並自動等待所有 goroutine 完成。

### **[plot](/Docs/plot.md)**

強大的 [github.com/go-echarts/go-echarts](https://github.com/go-echarts/go-echarts) 庫的封裝，用於簡化資料視覺化。

### **[gplot](/Docs/gplot.md)**

基於 [github.com/gonum/plot](https://github.com/gonum/plot) 的視覺化套件。快速且不需要 Chrome。甚至支援函數繪圖。

### **[lpgen](/Docs/lpgen.md)**

提供一個 **超級簡單** 且直觀的方式來生成線性規劃（LP）模型並將其保存為 `.lp` 檔。支援設置目標、添加約束、定義變量邊界，並指定二進制或整數變量。

### **[lp](/Docs/lp.md)**

使用 [GLPK](https://www.gnu.org/software/glpk/) 的全自動線性規劃（LP）包。

### **[csvxl](/Docs/csvxl.md)**

處理 Excel 和 CSV 文件。例如將 CSV 轉換為 Excel。

### **[py](/Docs/py.md)**

在 Go 中執行 Python 程式碼，無需手動安裝 Python 環境和依賴庫。允許在 Go 和 Python 之間傳遞變數。

## 進階使用

除了基本用法外，**Insyra** 還提供了處理不同數據類型和執行複雜統計操作的強大功能。請在 **[詳細文檔](https://github.com/HazelnutParadise/insyra/tree/main/Docs)** 中探索更多內容。

## 貢獻

歡迎各種形式的貢獻！您可以通過以下方式貢獻 **Insyra**：
- **[Issues](https://github.com/HazelnutParadise/insyra/issues):** 提出問題、建議或功能請求。
- **[Pull Requests](https://github.com/HazelnutParadise/insyra/pulls):** 提交代碼更改或新功能。
- **[Discussions](https://github.com/HazelnutParadise/insyra/discussions):** 參與討論，分享您的想法和建議。
<!-- 有關詳細信息，請參閱 [貢獻指南](https://github.com/HazelnutParadise/insyra/blob/main/CONTRIBUTING.md)。 -->

## 貢獻者
[![contributors](https://contrib.rocks/image?repo=HazelnutParadise/insyra)](https://github.com/HazelnutParadise/insyra/contributors)

## 授權

Insyra 採用 MIT 許可證授權。請參閱 [LICENSE](https://github.com/HazelnutParadise/insyra/blob/main/LICENSE) 文件以獲取更多資訊。
