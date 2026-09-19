# Tasks: test-untested-packages

## 1. 重新量測 (#307, #309)

- [x] 1.1 量出目前真實的覆蓋率與「完全沒有測試檔」的套件清單，不沿用審查當時的數字。
- [x] 1.2 判斷哪些套件不值得或無法測，寫下理由（`allpkgs`、`engine`、`cmd/insyra`、`py`、`tools/gendocs`）。

## 2. 繪圖套件 (#307 與 #309 的 plot)

- [x] 2.1 `gplot/charts_test.go`：七個 `CreateXxxChart` 各自建圖並存檔，含 png／svg／pdf／jpg／tif。
- [x] 2.2 同檔案：文件寫明的 nil 回傳（不支援的型別、空資料、NaN／Inf、空 grid）。
- [x] 2.3 `plot/charts_test.go`：十三個 `CreateXxx` 各自建圖並存成 HTML，斷言標題出現在輸出裡。
- [x] 2.4 同檔案：每個 `CreateXxx` 的空輸入 nil 回傳、`SaveHTML` 的成功與失敗、標題的 HTML 轉義、`SavePNG` 的引數檢查（不實際啟動瀏覽器）。
- [x] 2.5 `plot/internal/internal_test.go`：調色盤循環、`ApplyYAxis` 的類別軸判斷與回寫標籤。

## 3. 其餘沒有測試的套件 (#309 TS-13)

- [x] 3.1 `lpgen/lpgen_test.go`：模型建構、LP 檔逐段內容、LINGO 解析、`lingoIsBound` 的四個文件案例。
- [x] 3.2 `engine/ccl`、`engine/atomic`、`engine/ring`、`engine/biindex`、`engine/algorithms` 各一個測試檔。
- [x] 3.3 `stats/internal/parutil`：`ChunkBounds` 完整覆蓋 `[0, n)`，且與 `Run` 的切法一致。
- [x] 3.4 `datafetch/internal/limiter`：間隔、並行排隊、取消回滾，以及「兩次放行不會比間隔更近」的不變量。
- [x] 3.5 `cli/style`：開關彩色輸出兩種狀態。

## 4. 覆蓋率低於 50% 的套件 (#309 TS-18)

- [x] 4.1 `isr`：`DL` 全部、`DT.From` 未測到的型別、`Col`/`Row`/`At` 選擇器、五個 window wrapper、`Name()`、`CCL`。
- [x] 4.2 `internal/utils`：`ToFloat64` 家族、`TruncateString`、`FormatValue` 全部分支、時間戳的四個區間與邊界。
- [x] 4.3 `internal/algorithms` 已在審查期間升到 55.4%，不在本次範圍。
- [x] 4.4 記下 TS-18 還沒處理的五個套件與各自的原因。

## 5. 收尾

- [x] 5.1 `go test ./...`、`go vet ./...`、`golangci-lint run` 全綠。
- [x] 5.2 修正 `engine/ccl.EvalError` 的錯誤註解，並以測試釘住實際行為。
- [x] 5.3 `api-review.md` 的 TS-11、TS-13、TS-18 標注；`delivery-status.md` 里程碑。
- [ ] 5.4 #307 關閉；#309 附進度留言，TS-18 未完成的部分保持開啟。
- [ ] 5.5 測試過程中發現的缺陷記到 `AGENTS.md` 的 `## Follow-ups`，並在各自的 change 修正。
