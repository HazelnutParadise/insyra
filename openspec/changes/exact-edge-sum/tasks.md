# Tasks: exact-edge-sum

## 1. Accumulator
- [x] 1.1 先寫會失敗的測試：有限值的精確加總對 `math/big`，含相消、平手取偶、次正規數結果、溢位成無限大與剛好不溢位
- [x] 1.2 精確累加器（32 位元 digit、延後進位、一次捨入）
- [ ] 1.3 先寫會失敗的測試：NaN、`0·∞`、正負無限大、正負零與空加總
- [ ] 1.4 特殊值規則

## 2. EdgeSum
- [ ] 2.1 先寫會失敗的測試：前向與兩個梯度對 `math/big`、邊順序打亂結果不變
- [ ] 2.2 `EdgeSum` 與兩個梯度改用累加器，換掉舊的固定順序測試

## 3. Records
- [ ] 3.1 `BenchmarkEdgeSum` 前後對照，記進 `delivery-status.md`
- [ ] 3.2 `Docs/nn.md`、兩份 CHANGELOG 的 `EdgeSum` 條目改寫
- [ ] 3.3 全套驗證與 `openspec validate exact-edge-sum --strict`
