# Tasks: rotation-starts-same-on-every-platform

## 1. 先重現

- [x] 1.1 以 `GOARCH=amd64 go test` 在 Rosetta 下跑整個因子分析矩陣，與原生 arm64 比對，確認 `Restarts >= 2` 的 2400 組組合有 197 組差超過 1e-5，最大 2.1。
- [x] 1.2 確認差別在種子，不在抽取：`Restarts: 1` 的 800 組只有 8 組差超過 1e-5，且都在 1.5e-5 以內。
- [x] 1.3 在 `stats/internal/fa/rotation_starts_test.go` 寫測試：把 `overFactoredStructure()` 的 `[1,1]` 換成 amd64 抽出的值，要求隨機起點逐位元相同、`Restarts` 5 的 quartimin 準則值相同，確認修正前在 arm64 與 amd64 都是紅的。

## 2. 修正

- [x] 2.1 `buildStarts` 改用固定種子 `rotationStartSeed`，刪掉 `seedFromMatrix`。
- [x] 2.2 1.3 轉綠，並重跑 1.1 的比對：不一致從 197 組降到 24 組，全部是 `Restarts: 1` 本來就有的那 8 組。
- [x] 2.3 改動前後各跑一次 `INSYRA_STRICT_FACTOR_R_PARITY=1` 的 R parity 套件，比較失敗的葉節點數：前後都是 1,842，葉節點與數值完全相同。

## 3. 文件與紀錄

- [x] 3.1 兩份 CHANGELOG 的 `stats` 段落末尾各補一條。
- [x] 3.2 `Docs/stats.md` 與 `skills/insyra/references/stats.md` 的 `Restarts` 說明補上：隨機起點來自固定種子，各平台相同。
- [x] 3.3 `stats/factor_analysis_test.go` 的 strict suite 註解補上這條線自己的葉節點數，並說明為何兩條線不同。
- [x] 3.4 `overFactoredStructure` 註解寫成 quartimin 的實測，並說明 oblimin 在這條線上的行為。

## 4. 收尾

- [x] 4.1 `gofmt -l .`、`go build ./...`、`go vet ./...`、`go test -count=1 ./...`、`go test -race -count=1 ./stats/...`、`GOARCH=amd64 go test ./stats/...`、`golangci-lint run`。
- [x] 4.2 `openspec validate stats-factor-rotation --type spec --strict`。
