## ADDED Requirements

### Requirement: A function returning a DataTable returns a usable one

回傳 `*insyra.DataTable` 的函式 SHALL NOT 在失敗時回傳 nil，因為呼叫端的下一行通常就會呼叫它的方法而 panic。失敗時 SHALL 回傳空但可用的表格，並把原因記在 `Err()` 上。`lp.SolveFromFile` 與 `lp.SolveModel` 的兩個回傳值 SHALL 在所有路徑上都不是 nil。

#### Scenario: A solve that times out
- **WHEN** `result, info := lp.SolveFromFile(path, 1)` 逾時
- **THEN** `result.Show()` 印出空表格而不是 panic，`result.Err()` 說明逾時，`info` 也不是 nil

#### Scenario: Too many timeout arguments
- **WHEN** `lp.SolveFromFile(path, 1, 2)`
- **THEN** 兩個回傳值都是可用的表格，而不是兩個 nil

### Requirement: A reported success means there is a result

求解成功但結果檔讀不到時，附加資訊表的 `Status` SHALL NOT 是 `Success`。該情況 SHALL 回報為錯誤，並把原因放進警告欄位。

#### Scenario: The solution file cannot be read
- **WHEN** `glpsol` 正常結束，但結果檔不存在
- **THEN** `Status` 是 `Error`，警告欄位說明讀不到結果檔，結果表帶著同一個原因
