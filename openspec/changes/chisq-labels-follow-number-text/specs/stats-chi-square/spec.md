## MODIFIED Requirements

### Requirement: Goodness-of-fit probabilities are keyed by category

`ChiSquareGoodnessOfFit` SHALL 以 `p map[string]float64` 接收期望機率，鍵為類別標籤，與結果表的列名相同。類別標籤 SHALL 是該值依程式庫輸出文字的規則（與 `ToStringSlice` 相同）寫成的文字，再去掉前後空白：絕對值從 0.000001 到（不含）1e21 的浮點數寫成一般小數，其外為指數形式，其他型別照 `fmt` 的寫法，`nil` 為 `<nil>`。`ChiSquareIndependenceTest` 的列與欄類別 SHALL 依同一規則命名。`p` 為 nil 或空的 map 時 SHALL 使用均等機率。

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

#### Scenario: A large float category is a plain decimal
- **WHEN** 輸入為 `1500000.0`、`2.0`、`1500000.0`，呼叫 `ChiSquareGoodnessOfFit(input, map[string]float64{"1500000": 0.5, "2": 0.5}, false)`
- **THEN** 呼叫成功，`Observed` 的列名為 `1500000, 2`；以 `map[string]float64{"1.5e+06": 0.5, "2": 0.5}` 呼叫則回傳錯誤 `p names category "1.5e+06", which does not occur in input`

## ADDED Requirements

### Requirement: The independence test reads both lists at one moment

`ChiSquareIndependenceTest` SHALL 在同一個 `insyra.AtomicDoAll` 中讀取 `rowData` 與 `colData`，讓另一個以 `AtomicDoAll` 同時改變兩個 list 長度的 goroutine 無法落在兩次讀取之間。

#### Scenario: A writer resizes both lists together
- **WHEN** 另一個 goroutine 在 `insyra.AtomicDoAll` 內把兩個 list 一起在 10 與 20 個值之間切換，同時反覆呼叫 `ChiSquareIndependenceTest(rows, cols)`
- **THEN** 沒有任何一次回傳 `both DataLists must have the same length`
