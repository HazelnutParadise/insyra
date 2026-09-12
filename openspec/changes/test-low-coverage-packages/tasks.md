# Tasks: test-low-coverage-packages

## 1. cli/repl

- [x] 1.1 `completer_test.go`：`Do` 的回傳契約（剩下要打的字 + 已打的長度）。
- [x] 1.2 指令、變數、檔案三種補全，以及 `shouldCompletePath` 的整張表。
- [x] 1.3 檔案補全：前綴過濾、目錄補上分隔符、子目錄、讀不到的目錄。

## 2. stats/internal/fa

- [x] 2.1 七個 `vgQ*` 的解析梯度對數值微分。
- [x] 2.2 `GPForth`：正交性、共同性不變、目標函式不變差、單因子被拒絕、Kaiser 正規化後仍保共同性。
- [x] 2.3 `GPFoblq`：各欄單位長度、支援的方法名稱、單因子被拒絕。
- [x] 2.4 `SymmetricEigenDescendingDsyevr`：A·v = λ·v、降冪排序、特徵向量正交。
- [x] 2.5 `KaiserVarimaxWithRotationMatrix`、`Rotate`、`NormalizingWeight`、`frobNorm`、`obliqueCriterion`。

## 3. datafetch

- [x] 3.1 `RateLimitError` 的兩種訊息形式與 `errors.Is`／`errors.As`。
- [x] 3.2 `normalizeDateColumns` 的「取最後一個詞」判斷，含 `notadate` 這種不該轉換的名稱。
- [x] 3.3 `sleepBackoff`。

## 4. 收尾

- [x] 4.1 `go test ./...`、`go vet ./...`、`golangci-lint run` 全綠。
- [x] 4.2 `api-review.md` 的 TS-18 更新；`delivery-status.md` 里程碑。
- [ ] 4.3 #309 留言說明 TS-18 的進度與刻意不做的兩個套件。
- [x] 4.4 旋轉測試發現的 `fa.Rotate` 缺陷另外回報。
