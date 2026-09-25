# Tasks: nn-edge-sum-all-cores

- [x] 1.1 先寫會失敗的測試：一個 worker 與所有 worker 的前向、兩個梯度逐位元相同；小圖只用一個 worker
- [x] 1.2 前向與兩個梯度改用 `parallelFor`
- [ ] 2.1 一核心對全核心的 benchmark，量測結果記進 `delivery-status.md`
- [ ] 2.2 CHANGELOG 的 `EdgeSum` 條目補一句大圖會用滿所有核心
- [ ] 2.3 全套驗證與 `openspec validate nn-edge-sum-all-cores --strict`
