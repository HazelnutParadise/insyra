# Tasks: run-cvxpy-reference-in-ci

## 1. 查證
- [x] 1.1 本機以嚴格模式跑 `TestPortfolioAgreesWithCVXPY`，有 cvxpy 時通過；找不到帶 cvxpy 的 Python 時失敗而不是跳過。

## 2. 實作
- [x] 2.1 `reference-verification.yml` 安裝 cvxpy、直譯器檢查匯入 cvxpy、新增跑 quant 對照測試的步驟。
- [x] 2.2 `ENG.md` 的參考工具清單加上 cvxpy。

## 3. 驗證與紀錄
- [x] 3.1 推上 `0.4` 後 Reference Verification 通過，且 log 顯示 `TestPortfolioAgreesWithCVXPY` 實際執行並 PASS。
- [x] 3.2 `api-review.md` TS-4、`delivery-status.md`。
- [ ] 3.3 歸檔；在 #302 留言附證據並關閉。
