# Tasks: device-exact-product-sum

## 1. Library
- [ ] 1.1 `accel/internal/wgpu/exact_sum.go`：WGSL 函式庫（精確暫存器、清空、加入乘積、合併、捨入一次），註解寫明每個位數的界限
- [ ] 1.2 原始碼檢查測試：函式庫不含 `f16`、`f32`、`f64`、`i32`（不需要裝置，CI 會跑）

## 2. Parity on the device
- [ ] 2.1 測試用 kernel 與執行器：每列一個執行緒，單一暫存器與兩個暫存器合併兩種模式
- [ ] 2.2 `math/big` 參考答案，並以 `nn.EdgeSum` 對同一組乘積當第二個參考
- [ ] 2.3 隨機與刻意刁難的列在 M3 上逐位元相同（0 個不一致），兩種模式都是

## 3. Records
- [ ] 3.1 對全核心 CPU `EdgeSum` 量一次吞吐量，記進 `delivery-status.md`
- [ ] 3.2 `ENG.md` 記下暫存器配置與只用無號整數的規則
- [ ] 3.3 `delivery-status.md`：M35 完成，Next Ticket 改成 M36；`openspec validate device-exact-product-sum --strict`
