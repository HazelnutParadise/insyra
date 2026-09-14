# Tasks: rotations-use-gparotation-bb

## 1. 先寫會失敗的測試

- [x] 1.1 用 GPArotation 2026.8.2 的 `GPForth`／`GPFoblq`（`eps = 1e-5`、`maxit = 2000`、預設演算法）產生參考值：九種準則、四組載荷從單位矩陣出發，外加固定隨機起點與 Kaiser 正規化 Varimax，共 49 組。釘進 `stats/internal/fa/gparotation_bb_test.go`，要求前 6 次迭代的準則值與步長相對誤差 1e-10、收斂與否相同，收斂的案例最終準則值 1e-8、載荷 1e-4 之內。在現行程式上確認是紅的（50 組中 48 組失敗）。量測後改成這個比對方式的理由寫在 design；threeFactor 的 simplimax 因準則本身的平手處理不同而拿掉，另記 follow-up。
- [x] 1.2 寫測試：沒指定迭代上限、沒有起點收斂時，警告指出上限 2000。在現行程式上確認是紅的（警告寫 1000）。

## 2. 移植

- [x] 2.1 `GPFoblq` 改用參考演算法的步長、最近 10 次迭代的接受條件，以及奇異值分解的擬反矩陣。確認 1.1 的斜交案例轉綠。
- [x] 2.2 `GPForth` 改用同樣的步長與接受條件。確認 1.1 的正交案例轉綠。
- [x] 2.3 `FaRotations`、`fa.Rotate`、Promax 的前置 Varimax 預設上限改為 2000，並更新 `fa.go`、`stats/factor_analysis.go` 裡說明上限的註解。確認 1.2 轉綠。
- [x] 2.4 `go test ./stats/...` 在 arm64 與 `GOARCH=amd64` 都通過，沒有既有測試需要調整。

## 3. 量測

- [x] 3.1 改動後跑 `INSYRA_STRICT_FACTOR_R_PARITY=1` 的 R parity 套件，和 2026-09-13 的 922 個失敗葉節點比較，並分出 `rotation_converged`、Promax 等各類變化。結果 922 → 709：`rotation_converged` 99 → 0、Promax 246 → 76、simplimax 更差極小值 0。新增 196 個失敗都是載荷已在 2e-5 內、Phi 與分數沒有的欄位，R 自己在同一極小值的收斂起點間 `|Phi12|` 就差到 3.3e-3。

## 4. 文件與更正

- [x] 4.1 兩份 CHANGELOG 的 `stats` 段落末尾各補一條 **BREAKING**。
- [x] 4.2 `Docs/stats.md` 改成旋轉採用 GPArotation 預設演算法、上限 2000，順便把之前接成一長行的段落重新斷行。`skills/insyra/references/stats.md` 補一句演算法與上限。
- [x] 4.3 `stats/factor_analysis_test.go` 開頭註解改成 3.1 的新量測，並寫出正確的不收斂原因。
- [x] 4.4 已 archive 的 `rotation-starts-same-on-every-platform` proposal 與 design 各補一段更正說明。
- [x] 4.5 `delivery-status.md`：更正 `rotation-starts-same-on-every-platform` 那條 milestone 的原因，新增本 change 的 milestone 與 decision。
- [x] 4.6 `api-review.md` 補一列 ST-14。

## 5. 收尾

- [x] 5.1 `gofmt -l .`、`go vet ./...`、`go test ./...`、`golangci-lint run` 都通過。
- [x] 5.2 `openspec validate rotations-use-gparotation-bb --strict` 通過。
- [ ] 5.3 推上 `0.4`，確認 Test workflow 在三個 OS 都綠，再 verify 與 archive。
