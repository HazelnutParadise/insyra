# Tasks: oblimin-honours-its-start

## 1. 先重現

- [x] 1.1 在 `stats/internal/fa/rotation_starts_test.go` 釘一組過因子的載荷（60×6 合成表以 ML 抽 4 個因子的未旋轉載荷），寫測試要求 `gamma = 0` 的 oblimin 與 quartimin 在 `Restarts` 1／2／5／20 下準則值相同，並要求 `Restarts` 5 的準則值低於 `Restarts` 1，確認修正前是紅的。
- [x] 1.2 在 `stats/verify_more_test.go` 以公開 API 重現同一件事：`FixedK = 4`、oblimin、`Restarts` 5 的準則值低於 `Restarts` 1，確認修正前是紅的。

## 2. 修正

- [x] 2.1 `FaRotations` 的 `"oblimin"` 分支改成和其他斜交方法一樣：`pre = L·start`，從單位矩陣旋轉，拿掉自建的單位矩陣與 SPSS 註解。
- [x] 2.2 拿掉 `finalRot` 對 oblimin 不與起點相乘的特例。
- [x] 2.3 1.1、1.2 轉綠，`TestRestartsParameter` 與 `TestRotationPreservesTheModelAcrossRestarts` 仍綠。

## 3. 文件與紀錄

- [x] 3.1 兩份 CHANGELOG 的 `stats` 段落各補一條 **BREAKING**，接在 `orthogonal-rotation-starts` 的條目之後。
- [x] 3.2 `Docs/stats.md` 與 `skills/insyra/references/stats.md` 的 `Restarts` 說明補上「準則值更低不等於解更合用，過因子時可能換到因子高度相關的解」。
- [x] 3.3 `api-review.md` 補一列 ST-12。
- [x] 3.4 `delivery-status.md`：Latest Milestones 補一條，Decision Delta 把 oblimin 那條改成本次決定，並記下 psych 2.6.5 把 `n.rotations` 預設改成 20 這件待決事項。
- [x] 3.5 `AGENTS.md` 刪掉 oblimin 的 Follow-up，補一條「挑選規則跟 GPArotation 引擎而不是 psych faRotations 的 hyperplane count」的 Follow-up。

## 4. 收尾

- [x] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 4.2 `openspec validate oblimin-honours-its-start --strict`。
