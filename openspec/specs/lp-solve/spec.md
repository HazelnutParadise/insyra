# lp-solve Specification

## Purpose
How `lp` solves a linear or mixed-integer model: what `Solve` and `SolveFile` return, which outcomes are statuses and which are errors, how the default go-milp engine and the optional installed GLPK engine run, and how both engines read the same LP text as the same model.
## Requirements
### Requirement: A solve returns a solution or an error, never both

`lp.Solve` 與 `lp.SolveFile` SHALL 回傳 `(*Solution, error)`。error 不為 nil 時 `Solution` SHALL 為 nil；error 為 nil 時 `Solution` SHALL 不為 nil。模型無法求解（語法錯誤、不支援的寫法、沒有目標函數、nil 模型）SHALL 回傳包住原因的 `ErrInvalidModel`；選定的求解器沒有安裝 SHALL 回傳 `ErrEngineUnavailable`；求解器本身失敗且沒有任何解 SHALL 回傳 `ErrSolverFailed`。

#### Scenario: A model that solves
- **WHEN** 以 `lp.Solve(model, lp.Options{})` 求解一個有最佳解的 `lpgen` 模型
- **THEN** 回傳的 error 為 nil，`Solution` 不為 nil 且 `Status` 為 `StatusOptimal`

#### Scenario: A model that cannot be read
- **WHEN** 以 `lp.SolveFile` 讀取含有語法錯誤的 LP 檔
- **THEN** `Solution` 為 nil，`errors.Is(err, lp.ErrInvalidModel)` 為真，訊息指出出錯的行號

### Requirement: Solver outcomes are statuses, not errors

無可行解、無界、在時間上限前找到解但未證明最佳、以及在時間上限前沒有找到任何解，SHALL 分別以 `StatusInfeasible`、`StatusUnbounded`、`StatusFeasible`、`StatusStopped` 表示，並回傳 nil error。`Objective` 與 `Values` SHALL 只在 `StatusOptimal` 與 `StatusFeasible` 時有值；其他狀態下 `Values` SHALL 為 nil。

#### Scenario: An infeasible model
- **WHEN** 求解一個限制式互相矛盾的模型
- **THEN** error 為 nil，`Status` 為 `StatusInfeasible`，`Values` 為 nil

#### Scenario: A time limit with a solution in hand
- **WHEN** 整數規劃在 `TimeLimit` 到達時已有可行解
- **THEN** error 為 nil，`Status` 為 `StatusFeasible`，`Values` 含每個變數的值

### Requirement: The default engine needs no external program

`Options.Engine` 的零值 SHALL 使用 go-milp 求解，SHALL NOT 需要安裝任何外部程式，也 SHALL NOT 下載、編譯或安裝任何東西。`lp` 套件 SHALL NOT 修改行程的環境變數。

#### Scenario: A machine without GLPK
- **WHEN** 在沒有 `glpsol` 的機器上以 `lp.Options{}` 求解
- **THEN** 求解正常完成，不發出任何網路請求，`PATH` 與 `GLPK_PATH` 維持原值

### Requirement: The GLPK engine uses an installed glpsol

`Engine: lp.EngineGLPK` SHALL 先依 `GLPK_PATH`（可指向 `glpsol` 執行檔或其所在目錄）尋找 `glpsol`，找不到再從 `PATH` 尋找。兩處都沒有時 SHALL 回傳 `ErrEngineUnavailable`，錯誤訊息 SHALL 指向安裝說明。`TimeLimit` SHALL 以無條件進位的整數秒傳給 `--tmlim`。

#### Scenario: glpsol is missing
- **WHEN** `GLPK_PATH` 未設定、`PATH` 裡也沒有 `glpsol`，以 `EngineGLPK` 求解
- **THEN** `Solution` 為 nil，`errors.Is(err, lp.ErrEngineUnavailable)` 為真

### Requirement: GLPK results are read at full precision with a status from its solution file

GLPK 引擎 SHALL 從 `glpsol --write` 的結果檔讀取目標值與變數值，SHALL 從 `--output` 報告的欄編號對應變數名稱。狀態 SHALL 依結果檔的狀態字母判斷；字母為 `u` 時 SHALL 依 `glpsol` 輸出中的固定訊息分辨無可行解（`HAS NO PRIMAL FEASIBLE SOLUTION`）、無界（`HAS UNBOUNDED PRIMAL SOLUTION`）與時間上限（`TIME LIMIT EXCEEDED`）。報告與輸出原文 SHALL 放進 `Solution.Log`。

#### Scenario: A value the report rounds
- **WHEN** 解出的變數值是 123456.789，而報告只印出 `123457`
- **THEN** `Values` 裡該變數的值是 123456.789

#### Scenario: A variable name longer than 12 characters
- **WHEN** 報告把超過 12 個字元的變數名稱與其數值分成兩行
- **THEN** 該名稱仍對應到正確的值

#### Scenario: An infeasible LP under presolve
- **WHEN** 結果檔狀態為 `u`，`glpsol` 輸出含 `LP HAS NO PRIMAL FEASIBLE SOLUTION`
- **THEN** `Status` 為 `StatusInfeasible`，error 為 nil

### Requirement: The LP reader matches GLPK on the supported format

`lp` 的 CPLEX LP 讀取器 SHALL 接受 `lpgen` 寫出的所有模型，並對支援的語法（目標函數、限制式、Bounds、General／Integer、Binary、End）採用與 GLPK 5.0 讀取器相同的規則。不支援或不合法的寫法 SHALL 回傳 `ErrInvalidModel`，訊息 SHALL 含行號。變數 SHALL 依第一次出現的順序編號，與 GLPK 相同。

#### Scenario: What lpgen writes
- **WHEN** 讀取 `lpgen.GenerateLPFile` 寫出的模型
- **THEN** 讀取成功，變數、限制式、邊界與整數、二元宣告都與模型一致

#### Scenario: Both engines on the same model
- **WHEN** 機器上有 `glpsol`，以兩個引擎求解同一個有唯一最佳目標值的模型
- **THEN** 兩者的 `Status` 相同，目標值相差不超過 1e-6

### Requirement: A solution converts to a DataTable

`Solution.ToDataTable()` SHALL 回傳 `Variable` 與 `Value` 兩欄，列的順序 SHALL 與模型中變數的順序相同，SHALL NOT 回傳 nil。`Values` 為 nil 時 SHALL 回傳沒有列的表格，且 SHALL NOT 在 `Err()` 記錄錯誤。

#### Scenario: An infeasible solution as a table
- **WHEN** 對 `StatusInfeasible` 的 `Solution` 呼叫 `ToDataTable()`
- **THEN** 得到兩欄零列的表格，`Err()` 為 nil

