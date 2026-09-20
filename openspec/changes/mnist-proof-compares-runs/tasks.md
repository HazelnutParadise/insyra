# Tasks: mnist-proof-compares-runs

## 1. 實作
- [x] 1.1 `TestSequentialMNISTConvergence` 在同一個行程跑一次手寫 tape 迴圈，逐輪比較兩邊的平均損失與準確率。
- [x] 1.2 移除轉錄下來的常數，保留跨平台仍成立的界線（第二輪低於第一輪的一半、準確率至少 95%）。

## 2. 驗證與紀錄
- [x] 2.1 本機跑 `TestSequentialMNISTConvergence` 通過；先確認改前的版本在 amd64 會失敗（CI run 35519818488 已經是證據）。
- [ ] 2.2 推上 `0.4` 後 Neural Network Data Gates 通過，五個測試都執行。
- [x] 2.3 `AGENTS.md` follow-up 記下 `TestSequentialFitMNISTConvergence` 仍釘死常數。
- [ ] 2.4 歸檔。
