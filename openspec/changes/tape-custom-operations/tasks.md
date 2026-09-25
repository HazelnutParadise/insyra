# Tasks: tape-custom-operations

## 1. Custom operations
- [x] 1.1 先寫會失敗的測試：自訂乘法與內建 `Mul` 逐位元相同、夾在內建運算中間、surrogate 梯度、宣告被拒、`vjp` 回傳錯誤形狀／數量／型別／錯誤、失敗的 pass 不發布梯度
- [x] 1.2 `Tape.Custom` 與反向傳播時的檢查；失敗的 pass 保留上一次成功的梯度

## 2. Explicit upstream gradient
- [x] 2.1 先寫會失敗的測試：非純量輸出、`Backward(loss)` 與 `BackwardFrom(loss, 1)` 逐位元相同、不是這條 tape 產生的輸出、upstream 形狀不符
- [x] 2.2 `Tape.BackwardFrom`；`Backward` 改用同一段反向走訪

## 3. Docs and records
- [x] 3.1 `Docs/nn.md` 新增自訂運算一節，`skills/insyra/SKILL.md` 補上 `Custom` 與 `BackwardFrom`
- [x] 3.2 兩份 CHANGELOG 的 `` ### `nn` ``
- [x] 3.3 全套驗證：`go build`、`go vet`、`go test ./...`、`golangci-lint run`
- [x] 3.4 `openspec validate tape-custom-operations --strict`
