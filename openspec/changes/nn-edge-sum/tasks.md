# Tasks: nn-edge-sum

## 1. Topology and forward
- [x] 1.1 先寫會失敗的測試：拓撲驗證與複製、#379 的三條邊例子、重複目標與負權重對 float64 稠密參考、固定加總順序逐位元相同、批次形狀、形狀與型別錯誤、百萬節點少數邊
- [x] 1.2 `NewEdgeTopology`、`EdgeTopology`、`EdgeSum`

## 2. Reverse rule
- [x] 2.1 先寫會失敗的測試：對有限差分、對升冪參考迴圈逐位元相同、批次
- [x] 2.2 `Tape.EdgeSum`

## 3. Docs and records
- [ ] 3.1 `Docs/nn.md`、`skills/insyra/SKILL.md`
- [ ] 3.2 兩份 CHANGELOG 的 `` ### `ml` and `nn` ``
- [ ] 3.3 全套驗證：`go build`、`go vet`、`go test ./...`、`golangci-lint run`
- [ ] 3.4 `openspec validate nn-edge-sum --strict`
