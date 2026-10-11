# Tasks: lp-pure-go-default

## 1. 查證
- [x] 1.1 對照 GLPK 5.0 `cplex.c` 的 CPLEX LP 語法，讀取器依此實作（`lp/reader.go` 檔頭註明對照來源）

## 2. 測試
- [x] 2.1 LP 讀取器：`lp/reader_test.go`，涵蓋關鍵字拼法、邊界、整數與二元宣告、變數編號順序、語法錯誤回報行號
- [x] 2.2 go-milp 引擎：`lp/solve_test.go`，涵蓋最佳解、無解、無界、整數規劃、時間上限、`Status` 與 error 不重疊、nil 模型
- [x] 2.3 go-milp 的兩個缺陷：`TestMILPPresolveKeepsDistinctCloseConstraints`、`TestMILPUnboundedRelaxationWithNoIntegerFeasiblePointIsInfeasible`
- [x] 2.4 GLPK 引擎：`GLPK_PATH` 指向檔案或目錄、找不到時 `ErrEngineUnavailable`；有 `glpsol` 時端對端求解，沒有時跳過
- [x] 2.5 兩個引擎對同一批模型的狀態與目標值比對（需要 `glpsol`，否則跳過）
- [x] 2.6 `Solution.ToDataTable`：欄名、列序、零列時不記錄錯誤

## 3. 實作
- [x] 3.1 `lp` 的 API：`Engine`、`Options`、`Status`、`Solution`、`ToDataTable`、三個哨兵錯誤、`Solve`、`SolveFile`
- [x] 3.2 CPLEX LP 讀取器（`lp/reader.go`）
- [x] 3.3 go-milp 引擎（`lp/milp.go`），含時間上限時把「解加 `ErrTimeLimit`」轉成 `StatusFeasible`
- [x] 3.4 go-milp 的兩個缺陷的規避：關閉預處理、放大係數小的列、整數模型無界時先確認可行性
- [x] 3.5 GLPK 引擎（`lp/glpk.go`）：尋找 `glpsol`、執行、解析結果檔與報告、判斷狀態
- [x] 3.6 `SolveFromFile`、`SolveModel` 保留 GLPK 行為並標為 `Deprecated`，在 0.4 移除

## 4. 文件與紀錄
- [x] 4.1 `Docs/lp.md`：新 API、`Options`、狀態與錯誤、引擎選擇
- [x] 4.2 兩份 CHANGELOG 的 `lp` 條目，說明 `SolveFromFile`、`SolveModel` 為 Deprecated 而非 BREAKING
