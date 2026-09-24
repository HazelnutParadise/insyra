# Tasks: one-warning-per-unconverged-rotation

## 1. 先重現

- [x] 1.1 在 `stats/internal/fa/rotation_starts_test.go` 攔截 logger，5 個全部不收斂的起點要求恰好一則警告與一筆錯誤緩衝，20 個收斂的起點要求零則，確認修正前是紅的。

## 2. 修正

- [x] 2.1 `GPForth`、`GPFoblq` 的每次未收斂改為 `LogDebug`。
- [x] 2.2 `FaRotations` 在選中的解未收斂時記錄一次警告，指出方法、起點數、迭代上限。
- [x] 2.3 1.1 轉綠，`go test ./stats/...` 全綠。

## 3. 文件與紀錄

- [x] 3.1 `Docs/stats.md` 的 `Restarts` 段落補一句警告行為。
- [x] 3.2 兩份 CHANGELOG 的 `stats` 段各補一條。
- [x] 3.3 `delivery-status.md` Milestones 補一條，`AGENTS.md` 刪掉這條 follow-up。

## 4. 收尾

- [x] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 4.2 `openspec validate one-warning-per-unconverged-rotation --strict`。
