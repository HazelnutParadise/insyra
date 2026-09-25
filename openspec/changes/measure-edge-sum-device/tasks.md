# Tasks: measure-edge-sum-device

## 1. Prototype and parity
- [ ] 1.1 原型 WGSL 前向 kernel（每個輸出一個 invocation，依目標升冪加總）兩種累加寫法，放在 `accel/internal/wgpu` 的測試檔
- [ ] 1.2 小圖上兩種寫法各自對「不合併」與「合併」兩種 CPU 參考做逐位元與 ULP 比對

## 2. Timing
- [ ] 2.1 與 `BenchmarkEdgeSum` 相同的大小，最佳五次，兩種情境（每次全上傳、連線表只上傳一次）對全核心 CPU
- [ ] 2.2 在 Apple M3（Metal）上實跑，結果與判定記進 `delivery-status.md`

## 3. Records
- [ ] 3.1 `openspec validate measure-edge-sum-device --strict`
