## MODIFIED Requirements

### Requirement: Unknown arguments are refused

`plot`、`fetch`、`merge`、`clean`、`sort`、`accel` 與九個 DataList 統計指令（`sum`、`mean`、`median`、`mode`、`stdev`、`var`、`min`、`max`、`range`）對不屬於其文法的引數 SHALL 回傳錯誤並指出該引數，SHALL NOT 忽略，也 SHALL NOT 執行動作。`plot` 的 Usage SHALL NOT 宣告它不接受的選項。

#### Scenario: Extra tokens after a plot
- **WHEN** 執行 `plot line series extra junk`
- **THEN** 回傳錯誤指出 `extra`

#### Scenario: An alias on a command that stores nothing
- **WHEN** 執行 `mean x as m`
- **THEN** 回傳指出 `mean` 與 `"as"` 的錯誤，而且不會建立變數 `m`

#### Scenario: A removed accel flag
- **WHEN** 執行 `accel plan --precision float32`
- **THEN** 回傳錯誤，不產生規劃報告

### Requirement: accel's usage matches the command

`accel` 的 Usage SHALL NOT 宣告不存在的 `run` 子命令，也 SHALL NOT 宣告沒有任何動作會讀的 `--precision`。除了動作之外，Usage SHALL 只列 `--mode`，Cobra 命令上也 SHALL 只註冊 `--mode`。

#### Scenario: Registered flags
- **WHEN** 建立 `accel` 的 Cobra 命令
- **THEN** `mode` 旗標存在，`precision` 旗標不存在
