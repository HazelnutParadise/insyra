# cli-argument-validation Specification

## Purpose
CLI 的選項值只接受文件所列的寫法，不認識的引數必須回報而非忽略。

## Requirements
### Requirement: Option values come from a closed set

`sort` 的方向、`ttest` 的變異數假設、`ztest` 的對立假設、`clean` 的標準差 SHALL 只接受文件所列的寫法；拼錯 SHALL 回傳錯誤，SHALL NOT 退回預設值。

#### Scenario: Misspelled sort direction
- **WHEN** 執行 `sort dt price dsc`
- **THEN** 回傳錯誤而不是以升冪排序

### Requirement: Unknown arguments are refused

`plot`、`fetch`、`merge`、`clean`、`sort` 對不屬於其文法的引數 SHALL 回傳錯誤，SHALL NOT 忽略。`plot` 的 Usage SHALL NOT 宣告它不接受的選項。

#### Scenario: Extra tokens after a plot
- **WHEN** 執行 `plot line series extra junk`
- **THEN** 回傳錯誤指出 `extra`

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

`accel` 的 Usage SHALL NOT 宣告不存在的 `run` 子命令，SHALL 列出 `--precision`，且 `--mode` 與 `--precision` SHALL 都註冊在 Cobra 命令上。

#### Scenario: One-shot precision flag
- **WHEN** 建立 `accel` 的 Cobra 命令
- **THEN** `mode` 與 `precision` 兩個旗標都存在

