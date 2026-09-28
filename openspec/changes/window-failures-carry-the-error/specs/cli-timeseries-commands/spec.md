## ADDED Requirements

### Requirement: A failed window reducer is an error

`rolling`、`ewm` 與 `expanding` 的 reducer 結果帶有錯誤時，命令 SHALL 回傳含命令名稱前綴與該錯誤的錯誤，SHALL NOT 存任何變數，也 SHALL NOT 印出 `saved as`。`ewm` SHALL 依結果的錯誤判斷失敗，SHALL NOT 以結果長度推斷。

#### Scenario: A window of zero
- **WHEN** 變數 `x` 為 `[1, 2, 3]`，執行 `rolling x 0 mean as m`
- **THEN** 回傳以 `rolling:` 開頭的錯誤，`m` 不存在

#### Scenario: MinObs above the window
- **WHEN** 執行 `rolling x 2 mean minobs 3 as m`
- **THEN** 回傳錯誤，`m` 不存在
