## ADDED Requirements

### Requirement: A chi-square result holds observed and expected counts as two tables

`ChiSquareTestResult` SHALL 以 `Observed *insyra.DataTable` 與 `Expected *insyra.DataTable` 兩個欄位回報觀察次數與期望次數，兩張表形狀相同，每格都是 `float64`。SHALL NOT 再有把兩個數字包成一格的 `ContingencyTable` 欄位。

- 適合度檢定：兩張表各有一欄，`Observed` 表的欄名為 `Observed`，`Expected` 表的欄名為 `Expected`；每個出現在 `input` 的類別一列，依類別標籤的字串排序，列名為類別標籤。
- 獨立性檢定：每個列類別一列、每個欄類別一欄，都依標籤的字串排序，列名與欄名為類別標籤，與變更前 `ContingencyTable` 的列與欄相同。

兩個檢定的 `Statistic`、`PValue` 與 `DF` SHALL 與變更前逐位元相同。`Show()` SHALL 印出統計量、p 值、自由度以及這兩張表。

#### Scenario: Expected counts can be summed
- **WHEN** 對 `ChiSquareIndependenceTest` 的結果 `res` 呼叫 `res.Expected.GetColByNumber(0).Sum()`
- **THEN** 得到該欄期望次數的總和，不是 `NaN`，也不記錄錯誤

#### Scenario: Goodness-of-fit tables are named by category
- **WHEN** 以 `red, red, red, red, red, blue, blue, blue, green, green` 呼叫 `ChiSquareGoodnessOfFit(input, nil, false)`
- **THEN** `res.Observed.RowNames()` 為 `blue, green, red`，`Observed` 欄為 `3, 2, 5`，`Expected` 表的 `Expected` 欄為 `10/3, 10/3, 10/3`

### Requirement: Goodness-of-fit probabilities are keyed by category

`ChiSquareGoodnessOfFit` SHALL 以 `p map[string]float64` 接收期望機率，鍵為類別標籤，也就是該值轉成文字並去掉前後空白後的字串，與結果表的列名相同。`p` 為 nil 或空的 map 時 SHALL 使用均等機率。

- `p` 有某個鍵不是 `input` 中出現的類別時，SHALL 回傳錯誤 `p names category "<鍵>", which does not occur in input`，不回傳結果。多個這樣的鍵時，SHALL 回報排序後的第一個。
- `input` 中有某個類別在 `p` 中沒有鍵時，SHALL 回傳錯誤 `p has no probability for category "<類別>"`，不回傳結果。多個這樣的類別時，SHALL 回報排序後的第一個。
- 負值、`NaN`、`±Inf`、總和不為正，以及 `rescaleP` 為 false 時總和不為 1，SHALL 照舊回傳錯誤。`p` SHALL NOT 被修改。

`input` 中各值的出現順序 SHALL NOT 影響結果。

#### Scenario: Probabilities follow their names
- **WHEN** 以 `red` 5 次、`blue` 3 次、`green` 2 次（任意順序）呼叫 `ChiSquareGoodnessOfFit(input, map[string]float64{"red": 0.5, "green": 0.3, "blue": 0.2}, false)`
- **THEN** `Expected` 依列名 `blue, green, red` 為 `2, 3, 5`，`Statistic` 為 `(3-2)²/2 + (2-3)²/3 = 0.8333…`，`DF` 為 2

#### Scenario: A misspelled category is refused
- **WHEN** 同樣的輸入以 `map[string]float64{"red": 0.5, "green": 0.3, "Blue": 0.2}` 呼叫
- **THEN** 回傳錯誤 `p names category "Blue", which does not occur in input`，結果為 nil

#### Scenario: A category without a probability is refused
- **WHEN** 同樣的輸入以 `map[string]float64{"red": 0.5, "green": 0.5}` 呼叫
- **THEN** 回傳錯誤 `p has no probability for category "blue"`，結果為 nil

### Requirement: The chi-square tests agree with R's chisq.test

適合度檢定 SHALL 與 R `chisq.test(x = counts, p = p, rescale.p = rescaleP)` 相符，獨立性檢定 SHALL 與 R `chisq.test(table(rows, cols), correct = FALSE)` 相符；比對的項目為統計量、p 值、自由度、觀察次數與期望次數。參考值 SHALL 由呼叫 `chisq.test` 本身的 R 腳本產生並提交到 `stats/testdata`，測試讀取提交的結果，執行時不需要 R。

#### Scenario: Reference cases
- **WHEN** 執行 `stats` 的卡方測試
- **THEN** 至少三個適合度案例（均等機率、指定機率、需要 `rescale.p`）與至少三個獨立性案例（含 2×2 與大於 2×2）的每個比對項目都與 `chisq.test` 的輸出相符

### Requirement: The CLI names each goodness-of-fit proportion's category

CLI 的 `chisq gof` SHALL 接受 `chisq gof <var> [label=p ...]`，每個比例都寫成 `類別=比例`，在最後一個 `=` 分開，比例照舊重新縮放為總和 1。沒有 `=` 的參數 SHALL 回傳錯誤 `chisq gof: expected label=proportion, got "<參數>"`；同一個類別寫兩次 SHALL 回傳錯誤。沒有比例時 SHALL 使用均等機率。`Forms`、`Examples` 與 `Docs/cli-dsl.md` SHALL 使用新形式。

#### Scenario: Named proportions
- **WHEN** 變數 `colors` 為 `red` 5 次、`blue` 3 次、`green` 2 次，執行 `chisq gof colors red=0.5 green=0.3 blue=0.2`
- **THEN** 輸出的 `chi2` 與 `p` 與程式庫以同樣 map 呼叫的結果相同

#### Scenario: A bare number is refused
- **WHEN** 執行 `chisq gof colors 0.5 0.3 0.2`
- **THEN** 指令回傳錯誤 `chisq gof: expected label=proportion, got "0.5"`，不輸出結果
