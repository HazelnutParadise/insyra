## MODIFIED Requirements

### Requirement: Unknown arguments are refused

每個 CLI 指令對不屬於其文法的引數 SHALL 回傳錯誤，錯誤 SHALL 指出該引數並附上指令的 Usage；指令 SHALL NOT 忽略該引數，也 SHALL NOT 執行動作。不儲存結果的指令 SHALL 拒絕 `as <var>`。`plot` 的 Usage SHALL NOT 宣告它不接受的選項。

#### Scenario: Extra tokens after a plot
- **WHEN** 執行 `plot line series extra junk`
- **THEN** 回傳錯誤指出 `extra`

#### Scenario: An alias on a command that stores nothing
- **WHEN** 執行 `mean x as m`
- **THEN** 回傳指出 `mean` 與 `"as"` 的錯誤，而且不會建立變數 `m`

#### Scenario: A removed accel flag
- **WHEN** 執行 `accel plan --precision float32`
- **THEN** 回傳錯誤，不產生規劃報告

#### Scenario: A trailing argument on a fixed-shape command
- **WHEN** 執行 `iqr x junk`
- **THEN** 回傳 `iqr: unexpected argument "junk"; usage: iqr <var>`，不印出結果

#### Scenario: The count depends on the form
- **WHEN** 執行 `ttest single x 0 junk`
- **THEN** 回傳指出 `"junk"` 的錯誤；`ttest single x 0` 照常執行

#### Scenario: A script line with an extra argument
- **WHEN** `.isr` 腳本的一行是 `iqr x junk`
- **THEN** 這一行回報同樣的錯誤，`run` 接著執行下一行

## ADDED Requirements

### Requirement: Every command declares its arguments

每個註冊的指令 SHALL 在註冊時宣告它接受的引數數量：固定上限、依形式而定的上限，或由指令自行逐項檢查。數量檢查 SHALL 由註冊統一套用在指令執行之前，SHALL NOT 由各指令各自實作。沒有宣告的指令 SHALL 讓測試失敗。

#### Scenario: A command registered without a declaration
- **WHEN** 一個指令註冊時沒有 `Args`
- **THEN** `TestEveryCommandDeclaresItsArguments` 失敗並指出該指令

#### Scenario: A form the declaration does not know
- **WHEN** 執行 `ttest bogus x`
- **THEN** 數量檢查放行，由 `ttest` 自己回報未知的形式
