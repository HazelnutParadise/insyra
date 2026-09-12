# Tasks: factor-parity-compares-what-it-means

## 1. 先重現

- [ ] 1.1 用兩個小測試釘對齊與準則規則：只差欄序與正負號的兩組載荷比對要通過，準則值較高的解要失敗、較低的要通過。修正前是紅的。

## 2. 修正

- [ ] 2.1 `stats/internal/fa/criterion.go`：匯出 `Criterion(method, L, gamma, delta)`，varimax 用 Kaiser 正規化。
- [ ] 2.2 `assertFactorAnalysisMatchesR` 從載荷求一次對齊，套到所有以因子為索引的欄位；對齊後仍超出容忍度的 GPA 準則旋轉改比準則值。呼叫端傳入旋轉方法。
- [ ] 2.3 快取鍵加入工具鏈版本，每個測試程序只探測一次。
- [ ] 2.4 `INSYRA_STRICT_FACTOR_R_PARITY=1` 全跑一次，把剩下的失敗數與原因寫進 `factorParityTol` 的註解。

## 3. 紀錄

- [ ] 3.1 `delivery-status.md` Milestones 補一條，`AGENTS.md` 刪掉這條 follow-up。

## 4. 收尾

- [ ] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [ ] 4.2 `openspec validate factor-parity-compares-what-it-means --strict`。
