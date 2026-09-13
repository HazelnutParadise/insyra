# Tasks: orthogonal-rotation-starts

## 1. 先重現
- [x] 1.1 在 `stats/internal/fa` 寫測試，對每個方法檢查旋轉不變量（正交 `L·L' = Lu·Lu'`、斜交 `L·Φ·L' = Lu·Lu'`）與 `R'R = I`，`Restarts` 取 1／2／5／20，確認修正前是紅的。
- [x] 1.2 在 `stats` 層以公開 API 重現 #373 的數字，確認修正前是紅的。

## 2. 起點（#373 本體）
- [x] 2.1 起點清單只收正交矩陣：單位矩陣、Varimax 旋轉矩陣、QR 隨機正交矩陣，拿掉 Promax 與 TargetRot。
- [x] 2.2 使用前驗證起點正交，驗不過就略過並補上隨機起點。
- [x] 2.3 `Restarts` 等於起點總數。

## 3. 收斂
- [x] 3.1 每個旋轉包裝函式都把 `convergence` 傳出來。
- [x] 3.2 candidate 帶著 `convergence`，挑選時優先收斂的解；全部未收斂就回傳最佳者並回報未收斂。
- [x] 3.3 `fa.Rotate` 與 `FactorAnalysis.RotationConverged` 回報實際情況，並補測試。

## 4. 文件
- [x] 4.1 `stats/factor_analysis.go:93` 的註解寫 default 10，實際是 1。
- [x] 4.2 `skills/insyra/references/stats.md` 拿掉 #373 的警告。
- [x] 4.3 `Docs/stats.md`／`Docs/stats_TW.md` 說明 `Restarts` 的意義與 `RotationConverged`。

## 5. 收尾
- [x] 5.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 5.2 兩份 CHANGELOG。
- [x] 5.3 `api-review.md` 補一列、`delivery-status.md`、`AGENTS.md` 的 oblimin 後續項。
- [x] 5.4 關閉 #373 並附證據。
