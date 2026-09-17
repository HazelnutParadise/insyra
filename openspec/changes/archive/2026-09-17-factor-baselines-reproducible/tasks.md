# Tasks: factor-baselines-reproducible

## 1. 先寫會失敗的測試

- [x] 1.1 在 `stats/factor_analysis_test.go` 加測試：不經快取、以同一組輸入執行參考值腳本兩次，要求輸出逐位元組相同。在現行腳本上確認是紅的。

## 2. 修正

- [x] 2.1 `crosslang_baseline.R` 的 `factor_analysis_stats` 在呼叫 psych 前設定固定種子，註解寫明原因與 psych `faRotations` 平手判斷的缺陷。確認 1.1 轉綠。

## 3. 量測

- [x] 3.1 重新產生參考值後跑 `INSYRA_STRICT_FACTOR_R_PARITY=1` 的 R parity 套件，和 709 個失敗葉節點比較。結果 709 → 691：64 個不再失敗、46 個新失敗，全部落在因素框架欄位（326 → 308），其他類別不變，證實哪些葉節點失敗原本會隨 psych 的隨機起點移動。整個快取從零重新產生兩次，2411 個參考值檔案逐位元組相同，失敗的葉節點也相同。

## 4. 文件與收尾

- [x] 4.1 `stats/factor_analysis_test.go` 開頭註解依 3.1 更新，`delivery-status.md` 新增 milestone。
- [x] 4.2 `gofmt -l .`、`go vet ./...`、`go test ./stats/...`、`golangci-lint run`、`openspec validate factor-baselines-reproducible --strict` 都通過。
- [x] 4.3 推上 `0.4`，確認 CI 全綠，再 verify 與 archive。（b2a72c8b：Test 三個 OS、Reference Verification、Lint、Govulncheck、KNN／Clustering Parity 全綠）
