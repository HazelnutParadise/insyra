# cli-argument-validation Specification

## Purpose
CLI 的選項值只接受文件所列的寫法，不認識或用不到的引數必須回報並停止執行，不能默默忽略，因為被忽略的引數會讓打錯字看起來像是成功了。
## Requirements
### Requirement: Option values come from a closed set

`sort` 的方向、`ttest` 的變異數假設、`ztest` 的對立假設、`clean` 的標準差 SHALL 只接受文件所列的寫法；拼錯 SHALL 回傳錯誤，SHALL NOT 退回預設值。

#### Scenario: Misspelled sort direction
- **WHEN** 執行 `sort dt price dsc`
- **THEN** 回傳錯誤而不是以升冪排序

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

### Requirement: config validates keys and values

`config` 對未知的 key SHALL 回傳錯誤並列出支援的 key；`log-level`、`no-color`、`accel-mode` SHALL 只接受各自的合法值；無效的設定 SHALL NOT 被寫入設定檔。

#### Scenario: Unknown key
- **WHEN** 執行 `config bogus-key 123`
- **THEN** 回傳錯誤，設定檔不變

### Requirement: Ranges are checked

`sample` 的數量 SHALL 大於 0，且在不放回抽樣時 SHALL NOT 超過來源長度；`setcolnames` SHALL 要求名稱數量與欄數相同。

#### Scenario: Sample larger than the list
- **WHEN** 對 3 個元素的 list 執行 `sample x 99`
- **THEN** 回傳錯誤

### Requirement: accel's usage matches the command

`accel` 的 Usage SHALL NOT 宣告不存在的 `run` 子命令，也 SHALL NOT 宣告沒有任何動作會讀的 `--precision`。除了動作之外，Usage SHALL 只列 `--mode`，Cobra 命令上也 SHALL 只註冊 `--mode`。

#### Scenario: Registered flags
- **WHEN** 建立 `accel` 的 Cobra 命令
- **THEN** `mode` 旗標存在，`precision` 旗標不存在

### Requirement: Every command declares its arguments

每個註冊的指令 SHALL 在註冊時宣告它接受的引數數量：固定上限、依形式而定的上限，或由指令自行逐項檢查。數量檢查 SHALL 由註冊統一套用在指令執行之前，SHALL NOT 由各指令各自實作。沒有宣告的指令 SHALL 讓測試失敗。

#### Scenario: A command registered without a declaration
- **WHEN** 一個指令註冊時沒有 `Args`
- **THEN** `TestEveryCommandDeclaresItsArguments` 失敗並指出該指令

#### Scenario: A form the declaration does not know
- **WHEN** 執行 `ttest bogus x`
- **THEN** 數量檢查放行，由 `ttest` 自己回報未知的形式

