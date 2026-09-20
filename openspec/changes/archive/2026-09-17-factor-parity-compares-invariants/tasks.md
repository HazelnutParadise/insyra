# Tasks: factor-parity-compares-invariants

## 1. 先量測

- [x] 1.1 在種子化的參考值上暫時記錄 L·Phi·L'、S·L'、W·L'、F·L'、L·Cov(F)·L' 與我方 S − L·Phi，確認 125 個因素框架欄位失敗組合的 L·Phi·L'、S·L' 都在 6.2e-6 之內，我方 S 等於 L·Phi。
- [x] 1.2 以 GPArotation 2026.8.2 從 40 個隨機起點量同一極小值內的散布（`two_blocks` ML oblimin、`three_blocks` MINRES／PAF bentlerQ），並量 psych 在 8 個種子間的漂移，作為容忍度的依據。

## 2. 實作

- [x] 2.1 `assertFactorAnalysisMatchesR` 加上計分方法參數，六個呼叫處跟著改。
- [x] 2.2 加入 L·Phi·L'、S·L'（2e-5）、迴歸與 Bartlett 的 W·L'（1e-4）與 S = L·Phi（1e-10）的比對，在判定之前對每種旋轉執行。
- [x] 2.3 同一個解改由載荷差距判定，因素框架欄位一律以 `factorRotationTol` 比對，準則值只檢查不比 psych 差；`factorRotationTol` 改為 1e-2，註解寫明量測。
- [x] 2.4 加不需要 R 的單元測試，證明不變量不受旋轉、換序與變號影響，但會抓到 Phi 正負號錯誤與沒乘 Phi 的結構矩陣。

## 3. 量測

- [x] 3.1 跑 `INSYRA_STRICT_FACTOR_R_PARITY=1` 的 R parity 套件，和 691 個失敗葉節點比較，列出換到哪些欄位，確認新增的不變量失敗都落在萃取已經漂移的組合。結果 691 → 512（總共 52,176 個葉節點）：313 個因素框架欄位不再失敗，新增 134 個不變量失敗全部在 near_collinear 萃取已漂移的 52 個組合，剩下 11 個是無旋轉與 Promax 在 narrow_plus_group、mixed_scale 上經 R⁻¹ 放大的分數欄位，99 個是單因子資料的 Anderson-Rubin 組合。

## 4. 文件與收尾

- [x] 4.1 `stats/factor_analysis_test.go` 開頭註解依 3.1 更新，`delivery-status.md` 新增 milestone 與 decision。
- [x] 4.2 `gofmt -l .`、`go vet ./...`、`go test ./stats/...`、`golangci-lint run`、`openspec validate factor-parity-compares-invariants --strict` 都通過。
- [x] 4.3 推上 `0.4`，確認 CI 全綠，再 verify 與 archive。（1ef3e25b：Test 三個 OS、Reference Verification、Lint、Govulncheck、KNN／Clustering Parity 全綠）
