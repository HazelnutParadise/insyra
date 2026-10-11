# lp-solve Specification

## Purpose
How `lp.Solve` and `lp.SolveFile` solve a linear or integer program, with go-milp by default or an installed glpsol, what a Solution reports, and how the deprecated GLPK functions relate to them.

## Requirements

### Requirement: Solve and SolveFile return a solution or an error, never both

`lp.Solve` 與 `lp.SolveFile` SHALL 回傳 `(*Solution, error)`，error 不為 nil 時 `Solution` 為 nil。

#### Scenario: A model that solves
- **WHEN** 以 `lp.Solve(model, lp.Options{})` 求解一個有最佳解的 `lpgen` 模型
- **THEN** error 為 nil，`Solution` 不為 nil 且 `Status` 為 `StatusOptimal`

#### Scenario: A model that cannot be solved as written
- **WHEN** 傳入 nil 模型、沒有目標函數的模型，或 `SolveFile` 讀到含語法錯誤的 LP 檔
- **THEN** `Solution` 為 nil，`errors.Is(err, lp.ErrInvalidModel)` 為真，LP 檔的錯誤訊息指出出錯的行號

#### Scenario: A file that cannot be read
- **WHEN** `SolveFile` 讀取不存在的檔案
- **THEN** `Solution` 為 nil，`errors.Is(err, fs.ErrNotExist)` 為真

### Requirement: Solver outcomes are statuses, not errors

無可行解、無界、時間上限前找到解、時間上限前沒有找到解，SHALL 分別以 `StatusInfeasible`、`StatusUnbounded`、`StatusFeasible`、`StatusStopped` 表示，且 error 為 nil。

#### Scenario: An infeasible model
- **WHEN** 求解一個限制式互相矛盾的模型
- **THEN** error 為 nil，`Status` 為 `StatusInfeasible`，`Values` 為 nil

#### Scenario: A time limit with a solution in hand
- **WHEN** 整數規劃在 `TimeLimit` 到達時已有可行解
- **THEN** error 為 nil，`Status` 為 `StatusFeasible`，`Values` 含每個變數的值

#### Scenario: A time limit before any solution
- **WHEN** 整數規劃在 `TimeLimit` 到達時還沒有任何解
- **THEN** error 為 nil，`Status` 為 `StatusStopped`，`Values` 為 nil

### Requirement: The time limit is best effort

`Options.TimeLimit` SHALL 是盡力而為的搜尋上限，不保證總求解時間不超過它。

#### Scenario: go-milp's limit does not cover the first relaxation
- **WHEN** 以 go-milp 引擎設定 `TimeLimit`
- **THEN** 上限只計算分支搜尋，不包括最初的連續鬆弛，總求解時間可能超過上限

#### Scenario: GLPK's limit is rounded up to whole seconds
- **WHEN** 以 `EngineGLPK` 設定 `TimeLimit` 為 1.5 秒
- **THEN** 傳給 `glpsol` 的 `--tmlim` 為 2

### Requirement: The default engine needs no external program

`Options` 的零值 SHALL 使用 go-milp 求解，且 `Solve` 與 `SolveFile` SHALL NOT 修改 `PATH`、`GLPK_PATH` 等行程環境變數。

#### Scenario: A machine without GLPK
- **WHEN** 在沒有 `glpsol` 的機器上以 `lp.Options{}` 求解
- **THEN** 求解正常完成，`PATH` 與 `GLPK_PATH` 維持原值，不下載或安裝任何東西

### Requirement: The go-milp engine works around two defects of go-milp v0.2.0

go-milp 引擎 SHALL 在交給 go-milp 前關閉其預處理並放大係數最大值小於 1 的列，且遇到無界的整數模型時 SHALL 先確認它是否有整數可行點。

#### Scenario: Distinct rows with small coefficients
- **WHEN** 兩條係數約為 1e-7 的限制式彼此不同，例如 `1e-7 x + 1e-7 y = 1` 與 `2e-7 x + 1e-7 y = 1.5`
- **THEN** 模型求得最佳解，目標值約為 1e7，而不是被判為無解

#### Scenario: An unbounded relaxation with no integer point
- **WHEN** 整數模型的鬆弛無界，但等式 `2 x - 2 y = 1` 的整數係數無法整除右側
- **THEN** `Status` 為 `StatusInfeasible`，error 為 nil

#### Scenario: The feasibility check cannot decide
- **WHEN** 無界的整數模型，其整數可行性檢查在 `TimeLimit` 內無法判定（未設定時的上限為 30 秒）
- **THEN** `Status` 為 `StatusStopped`，error 為 nil

### Requirement: The GLPK engine uses an installed glpsol and never installs one

`Engine: lp.EngineGLPK` SHALL 依 `GLPK_PATH`（可指向 `glpsol` 執行檔或其所在目錄）、再依 `PATH` 尋找 `glpsol`，兩處都沒有時 SHALL 回傳 `ErrEngineUnavailable` 且不安裝任何東西。

#### Scenario: glpsol is missing
- **WHEN** `GLPK_PATH` 未指向有效的檔案或目錄，`PATH` 裡也沒有 `glpsol`，以 `EngineGLPK` 求解
- **THEN** `Solution` 為 nil，`errors.Is(err, lp.ErrEngineUnavailable)` 為真

#### Scenario: GLPK_PATH names the directory holding glpsol
- **WHEN** `GLPK_PATH` 指向含有 `glpsol` 的目錄，`PATH` 裡沒有它
- **THEN** 使用該目錄的 `glpsol` 求解，結果的 `Engine` 為 `EngineGLPK`（沒有 `glpsol` 的機器上此測試跳過）

#### Scenario: An unknown engine name
- **WHEN** `Options.Engine` 是不認得的名稱
- **THEN** `Solution` 為 nil，`errors.Is(err, lp.ErrEngineUnavailable)` 為真

### Requirement: GLPK results come from glpsol's solution file at full precision

GLPK 引擎 SHALL 從 `--write` 產生的結果檔讀取目標值與變數值，並依結果檔的狀態字母決定 `Status`。

#### Scenario: A value the report rounds
- **WHEN** 結果檔中某變數的值是 123456.789，而報告只印出 `123457`
- **THEN** `Values` 裡該變數的值是 123456.789

#### Scenario: A variable name longer than 12 characters
- **WHEN** 報告把超過 12 個字元的變數名稱與其數值分成兩行
- **THEN** 該名稱仍對應到正確的值

#### Scenario: An undefined status
- **WHEN** 結果檔的狀態字母為 `u`，`glpsol` 輸出含 `LP HAS NO PRIMAL FEASIBLE SOLUTION`
- **THEN** `Status` 為 `StatusInfeasible`，error 為 nil；輸出含 `LP HAS UNBOUNDED PRIMAL SOLUTION` 時為 `StatusUnbounded`，含 `TIME LIMIT EXCEEDED` 時為 `StatusStopped`

#### Scenario: The log
- **WHEN** GLPK 引擎求解完成
- **THEN** `Solution.Log` 含 `glpsol` 的輸出與報告原文，`Nodes` 為 0

### Requirement: The LP reader reads the supported CPLEX LP syntax as GLPK does

`lp` 的 CPLEX LP 讀取器 SHALL 接受 `lpgen` 寫出的所有模型，並對支援的語法採用與 GLPK 5.0 讀取器相同的規則。

#### Scenario: What lpgen writes
- **WHEN** 讀取 `lpgen` 寫出的模型
- **THEN** 讀取成功，變數、限制式、邊界與整數、二元宣告都與模型一致

#### Scenario: A syntax error
- **WHEN** LP 檔含有語法錯誤
- **THEN** 回傳 `ErrInvalidModel`，錯誤訊息含檔名與出錯的行號

#### Scenario: Variables are numbered by first appearance
- **WHEN** 變數第一次出現的順序與字母順序不同
- **THEN** 變數依第一次出現的順序編號，與 GLPK 相同

#### Scenario: Bounds that GLPK refuses
- **WHEN** 下界大於上界，或整數變數的邊界不是整數
- **THEN** 回傳 `ErrInvalidModel`，兩個引擎的結果一致

#### Scenario: Both engines on the same model
- **WHEN** 機器上有 `glpsol`，以兩個引擎求解同一個有唯一最佳目標值的模型
- **THEN** 兩者的 `Status` 相同，目標值相差不超過 1e-6（沒有 `glpsol` 時跳過）

### Requirement: A solution converts to a DataTable

`Solution.ToDataTable()` SHALL 回傳 `Variable` 與 `Value` 兩欄，列的順序與模型中變數的順序相同，且 SHALL NOT 回傳 nil。

#### Scenario: An infeasible solution as a table
- **WHEN** 對 `StatusInfeasible` 的 `Solution` 呼叫 `ToDataTable()`
- **THEN** 得到兩欄零列的表格，`Err()` 為 nil

### Requirement: Solution reports the engine, the time and the log

`Solution` SHALL 記錄求解所用的 `Engine`、`Elapsed`、`Nodes` 與 `Log`，且 `Nodes` 在 GLPK 引擎下為 0。

#### Scenario: A go-milp solve
- **WHEN** 以預設引擎求解
- **THEN** `Engine` 為 `EngineMILP`，`Elapsed` 為實際經過的時間，`Log` 為讀取 LP 時的警告（沒有警告時為空字串）

### Requirement: The deprecated solve functions keep their GLPK behaviour until 0.4

`SolveFromFile` 與 `SolveModel` SHALL 保留 0.3.x 的 GLPK 行為（包括找不到 `glpsol` 時自動安裝 GLPK），並標為 `Deprecated`，在 0.4 移除。

#### Scenario: A caller of SolveModel
- **WHEN** 呼叫 `lp.SolveModel(model, 10)`
- **THEN** 回傳與 0.3.x 相同的兩張 DataTable，行為不因新增的 `Solve` 而改變

#### Scenario: The deprecation notices
- **WHEN** 讀取 `SolveFromFile` 與 `SolveModel` 的 doc comment
- **THEN** 各自有 `Deprecated:` 段落，指名 `Solve` 或 `SolveFile` 為替代，並寫明在 0.4 移除
