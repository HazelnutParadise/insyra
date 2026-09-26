# cli-failure-reporting Specification

## Purpose
CLI 指令做不到被交代的事時，必須以非零狀態結束並說明原因。
## Requirements
### Requirement: A command that fails exits non-zero

`sort`、`dropcol`、`droprow`、`swap`、`ccl`、`addcolccl`、`setcolnames`、`sample` 在目標不存在或程式庫記錄錯誤時 SHALL 回傳錯誤，SHALL NOT 印出成功訊息。錯誤訊息 SHALL 指出缺少的目標。

#### Scenario: Sorting by a column that is not there
- **WHEN** 對沒有 `nonexistent` 欄的表執行 `sort dt nonexistent`
- **THEN** 回傳錯誤，且輸出不含 "sorted"

#### Scenario: A CCL expression that cannot compile
- **WHEN** 執行 `ccl dt "bogus((("`
- **THEN** 回傳錯誤，且輸出不含 "ccl executed"

### Requirement: An error names the command and the argument

指令回報的錯誤 SHALL 指出是哪個指令、哪個引數，SHALL NOT 直接把標準函式庫的錯誤文字（`strconv.ParseFloat: parsing …`）交給使用者。

#### Scenario: A non-numeric argument
- **WHEN** `ttest single x abc`
- **THEN** 錯誤訊息含 `ttest` 與 `mu`，不含 `strconv.`

### Requirement: An unknown option says what is supported

選項的鍵 SHALL 不分大小寫比對。未知的選項或值 SHALL 在訊息中列出支援的清單。列舉型別的值 SHALL 在進入函式庫之前驗證。

#### Scenario: An uppercase option key
- **WHEN** `kmeans dt 2 NSTART 3`
- **THEN** 與 `nstart 3` 效果相同

#### Scenario: A misspelled enum value
- **WHEN** `knn` 的 `weighting` 給了 `inverse`
- **THEN** 回報錯誤並列出 `uniform, distance`

### Requirement: A wrong type is not reported as a missing variable

同時接受 DataTable 與 DataList 的指令，在變數存在但型別不符時 SHALL 說明型別不符，SHALL NOT 回報「variable not found」。

#### Scenario: A variable holding a number
- **WHEN** `clone x as y`，而 `x` 是一個整數
- **THEN** 錯誤說明 `x` 不是 DataTable 也不是 DataList，而不是說找不到 `x`

