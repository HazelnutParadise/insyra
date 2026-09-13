# Tasks: lp-pure-go-default

## 1. 查證
- [x] 1.1 以 GLPK 5.0 實際執行 `glpsol --output --write`，記錄最佳解、整數最佳解、LP 無解、LP 無界、整數無解、鬆弛問題無界、時間上限有解、時間上限無解、長變數名稱與語法錯誤的報告、結果檔、stdout 與結束代碼，存成 `lp/testdata/glpk/` 測試樣本。
- [x] 1.2 讀 GLPK 5.0 `cplex.c`，整理 CPLEX LP 語法規則供讀取器對齊。
- [x] 1.3 確認 `lpgen` 與舊 `SolveModel` 兩種寫法 GLPK 都讀得進去且答案相同。

## 2. 測試（先寫、先看它失敗）
- [x] 2.1 LP 讀取器：`lpgen` 寫出的模型、各段關鍵字的拼法、邊界寫法、整數與二元宣告、變數編號順序、語法錯誤與不支援寫法回報行號。
- [x] 2.2 go-milp 引擎：最佳解、無解、無界、整數規劃、時間上限，`Status` 與 error 不重疊，nil 模型與空目標。
- [x] 2.3 GLPK 輸出解析：以 1.1 的樣本驗證全精度數值、長名稱、狀態字母與 `u` 時的訊息判斷。
- [x] 2.4 GLPK 引擎：`GLPK_PATH` 指向檔案或目錄、找不到時 `ErrEngineUnavailable`；有 `glpsol` 時端對端求解，沒有時跳過。
- [x] 2.5 兩個引擎對同一批模型的狀態與目標值一致（需要 `glpsol`，否則跳過）。
- [x] 2.6 `ToDataTable`：欄名、列序、零列時不記錄錯誤。
- [x] 2.7 `lpgen.WriteLP` 與 `GenerateLPFile` 輸出相同。

## 3. 實作
- [x] 3.1 `lpgen.(*LPModel).WriteLP(io.Writer) error`，`GenerateLPFile` 改用它。
- [x] 3.2 `lp` 的 API：`Engine`、`Options`、`Status`、`Solution`、`ToDataTable`、三個哨兵錯誤、`Solve`、`SolveFile`。
- [x] 3.3 CPLEX LP 讀取器。
- [x] 3.4 go-milp 引擎，含時間上限時把「解加 `ErrTimeLimit`」轉成 `StatusFeasible`。
- [x] 3.5 GLPK 引擎：尋找 `glpsol`、執行、解析結果檔與報告、判斷狀態。
- [x] 3.6 刪除自動安裝、`SolveFromFile`、`SolveModel`、報告逐行解析與附加資訊表，以及只測它們的測試。

## 4. CI、文件與紀錄
- [x] 4.1 CI 的 Ubuntu 與 macOS 測試安裝 GLPK，讓端對端與雙引擎比對測試實際執行。
- [x] 4.2 `Docs/lp.md`：新 API、狀態與錯誤、引擎選擇、各系統安裝 GLPK 的方式、go-milp#2 的已知限制。
- [x] 4.3 容量規劃教學改用新 API；`Docs/lpgen.md` 補上 `WriteLP`。
- [x] 4.4 兩份 CHANGELOG 的 BREAKING 條目（`lp`）與新增條目（`lpgen`）。
- [x] 4.5 `api-review.md` 的 LP-1、LP-2、SEC-9、SEC-19，`delivery-status.md`。
- [x] 4.6 `gofmt -l`、`go build ./...`、`go vet ./...`、`go test ./...`（含有 `glpsol` 的端對端）、`golangci-lint run`、`govulncheck ./...`。
- [x] 4.7 歸檔後補寫 `lp-solve` 的 Purpose，更新 `deterministic-and-atomic-output` 的 Purpose；在 #257、#372 留言附證據並關閉。
