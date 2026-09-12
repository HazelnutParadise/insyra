# Tasks: promax-matches-psych

## 1. 先重現

- [x] 1.1 在 `stats/internal/fa/promax_psych_test.go` 釘 psych 2.6.5 在 parity 十列表上的 `Promax(weighted, m = 4)` 載荷與 `Phi`，容忍 5e-4，確認修正前是紅的（載荷差 8e-2）。

## 2. 修正

- [x] 2.1 `fa.Promax` 的前置旋轉改為 `Varimax`（梯度投影、單位矩陣起點、不正規化），Th 取其旋轉矩陣。
- [x] 2.2 1.1 轉綠，`go test ./stats/...` 全綠。
- [x] 2.3 `INSYRA_STRICT_FACTOR_R_PARITY=1` 全跑一次，記下 Promax 剩餘失敗數，更新 `factorParityTol` 註解。

## 3. 文件與紀錄

- [x] 3.1 兩份 CHANGELOG 的 `stats` 段各補一條 **BREAKING**。
- [x] 3.2 `delivery-status.md` Milestones 補一條，`AGENTS.md` 刪掉 Promax 的 follow-up。

## 4. 收尾

- [x] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 4.2 `openspec validate promax-matches-psych --strict`。
