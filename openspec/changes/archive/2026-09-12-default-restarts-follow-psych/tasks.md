# Tasks: default-restarts-follow-psych

## 1. 先重現

- [x] 1.1 在 `stats/verify_more_test.go` 釘 `DefaultFactorAnalysisOptions().Rotation.Restarts == 20`，確認修正前是紅的。

## 2. 修正

- [x] 2.1 `DefaultFactorAnalysisOptions()` 的 `Restarts` 改為 20，struct 註解與 `Docs/stats.md` 的預設說明同步。
- [x] 2.2 `buildStarts` 接收 `eps`、`maxIter` 並交給資訊起點的 `Varimax`，`FaRotations` 傳入旋轉自己的值，四個測試呼叫點跟著改。
- [x] 2.3 `go test ./stats/...` 全綠，含 R parity 以外的所有 FactorAnalysis 測試。

## 3. 文件與紀錄

- [x] 3.1 `Docs/stats.md` 的 `Restarts` 段落：預設 20 與出處、挑選規則與它跟的是哪個參考實作。
- [x] 3.2 `skills/insyra/references/stats.md` 同步。
- [x] 3.3 兩份 CHANGELOG 的 `stats` 段各補一條 **BREAKING**。
- [x] 3.4 `delivery-status.md`：Milestones 補一條，Decision Delta 把「兩個待決」那條換成兩個決定。
- [x] 3.5 `AGENTS.md` 刪掉挑選規則與 `n.rotations` 的 follow-up，補一條每個起點各記一次未收斂警告的 follow-up。

## 4. 收尾

- [x] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 4.2 `openspec validate default-restarts-follow-psych --strict`。
