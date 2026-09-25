# Tasks: correctly-rounded-tanh

## 1. Oracle and measurement
- [ ] 1.1 `math/big` 的 tanh 參考答案，以 Python `decimal` 算出的 50 位數錨點驗證
- [ ] 1.2 閘控的 2^32 全面比對測試；先對舊的 `nn.Tanh` 跑一次，記下不是正確捨入的輸入數量

## 2. Implementation
- [ ] 2.1 正確捨入的 `tanh`：解析區間、float64 快速路徑加寬安全距離、256 位元後備
- [ ] 2.2 全面比對通過（0 個不一致）
- [ ] 2.3 `tanhVJP` 每一步明確捨入，不合併；對逐步參考逐位元相同

## 3. Records
- [ ] 3.1 `Docs/nn.md`、兩份 CHANGELOG（含改變的輸入數量）、`delivery-status.md`
- [ ] 3.2 全套驗證與 `openspec validate correctly-rounded-tanh --strict`
