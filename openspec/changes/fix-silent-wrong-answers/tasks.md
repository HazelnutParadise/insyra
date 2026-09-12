# Tasks: fix-silent-wrong-answers

## 1. 先重現

- [x] 1.1 `FormatValue` 在 `2^63` 與更大值上的測試，原生與 `GOARCH=amd64` 都要跑。
- [x] 1.2 毫秒時間戳超過 2262-04-11 的測試。
- [x] 1.3 `isr.DT.From(map[int]any{...})` 的測試。

## 2. 修正

- [x] 2.1 `internal/utils/utils.go`：`FormatValue` 轉換前先檢查範圍，並改用 `int64`。
- [x] 2.2 同檔案：毫秒分支改用 `time.UnixMilli`。
- [x] 2.3 `isr/dt.go`：`map[int]any` 的鍵改用 `numberToColIndex`。

## 3. 收尾

- [x] 3.1 1.1 到 1.3 轉綠；原生與 `GOARCH=amd64` 都跑過 `go test ./...`。
- [x] 3.2 `golangci-lint run`。
- [x] 3.3 兩份 CHANGELOG 的 Core 與 `isr` 條目。
- [x] 3.4 `delivery-status.md` 里程碑。
