# Tasks: rotation-starts-same-on-every-platform

## 1. 先重現

- [x] 1.1 以 `GOARCH=amd64 go test` 在 Rosetta 下重現 `TestObliminRestartsSearchTheCriterion` 失敗，arm64 通過。
- [x] 1.2 逐起點記錄種子與準則值，確認 amd64 的額外起點有被使用，差別在種子：amd64 從 `Restarts` 7 開始到達低谷，arm64 從 5 開始。
- [x] 1.3 在 `stats/internal/fa/rotation_starts_test.go` 寫測試：把 `overFactoredStructure()` 的 `[1,1]` 換成 amd64 抽出的值，要求隨機起點逐位元相同、`Restarts` 5 的準則值相同，確認修正前在 arm64 與 amd64 都是紅的。

## 2. 修正

- [x] 2.1 `buildStarts` 改用固定種子，刪掉 `seedFromMatrix`。
- [x] 2.2 1.3 轉綠。`TestObliminRestartsSearchTheCriterion` 不改，在 amd64 與 arm64 都綠。
- [x] 2.3 改動前後各跑一次 `INSYRA_STRICT_FACTOR_R_PARITY=1` 的 R parity 套件，比較失敗的葉節點數。

## 3. 文件與紀錄

- [x] 3.1 兩份 CHANGELOG 的 `stats` 段落末尾各補一條 **BREAKING**。
- [x] 3.2 `Docs/stats.md` 與 `skills/insyra/references/stats.md` 的 `Restarts` 說明補上：隨機起點來自固定種子，各平台相同。
- [x] 3.3 `overFactoredStructure` 註解裡「第三個隨機起點」改成修正後的實測。
- [x] 3.4 `api-review.md` 補一列 ST-13。
- [x] 3.5 `delivery-status.md` 的 Latest Milestones 與 Decision Delta 各補一條。

## 4. 收尾

- [ ] 4.1 `gofmt -l .`、`go vet ./...`、`go test ./...`、`GOARCH=amd64 go test ./stats/...`、`golangci-lint run`。
- [ ] 4.2 `openspec validate rotation-starts-same-on-every-platform --strict`。
- [ ] 4.3 推到 `0.4`，確認 Test workflow 在三個 OS 都綠，再 archive。
