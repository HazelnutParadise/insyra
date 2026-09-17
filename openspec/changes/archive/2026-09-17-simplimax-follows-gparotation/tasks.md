# Tasks: simplimax-follows-gparotation

## 1. 先寫會失敗的測試

- [x] 1.1 從 GPArotation 2026.8.2 的 `vgQ.simplimax` 取四組載荷在預設 `k` 與 `k = nrow` 下的準則值與梯度，釘進測試，並測 `Criterion("simplimax", …)` 用預設 `k`。在現行程式上確認 6×3（有相等值）與 6×4 的案例是紅的。
- [x] 1.2 `gparotation_bb_test.go` 的 simplimax 參考案例改用 `GPArotation::simplimax(A, Tmat)` 重新產生（預設 `k`），補回 6×3 的案例，拿掉跳過它的註解。在現行程式上確認是紅的。
- [x] 1.3 檢查既有的 `TestVgQSimplimax`，和 GPArotation 算法不同的地方改成 GPArotation 的值並寫明原因。（它只檢查方法名稱、有限值與維度，不需要改。）

## 2. 修正

- [x] 2.1 `vgQSimplimax` 改成依欄優先位置穩定排序、恰好取 `k` 個、依欄優先順序加總，NaN 排在最後。確認 1.1 轉綠。
- [x] 2.2 新增預設 `k` 的共用函式，`obliqueCriterion`、`criterion.go`、`FaRotations` 都改用它，移除 `Simplimax` 包裝函式沒用到的 `k` 參數。確認 1.2 轉綠。
- [x] 2.3 `go test ./stats/...` 在 arm64 與 `GOARCH=amd64` 都通過。

## 3. 量測

- [x] 3.1 改動後跑 `INSYRA_STRICT_FACTOR_R_PARITY=1` 的 R parity 套件，和改動前的 709 個失敗葉節點（26 個 simplimax）比較。結果仍是 709，26 個 simplimax 葉節點完全相同：`three_blocks` 與 `cross_loading` 的 simplimax 組合都以「和 R 不同、而且更低的極小值」通過，照 GPArotation 的準則我們到 0.0068 至 0.236，psych 的解在 0.101 至 0.668，因為 psych 用 hyperplane count 挑起點。所以這個 change 的證據是參考值測試，不是 parity 數字。

## 4. 文件

- [x] 4.1 兩份 CHANGELOG 的 `stats` 段落末尾各補一條 **BREAKING**。
- [x] 4.2 `Docs/stats.md` 補一句 Simplimax 的準則與預設 `k`。
- [x] 4.3 `stats/factor_analysis_test.go` 開頭註解改成 3.1 的量測。
- [x] 4.4 `delivery-status.md` 新增 milestone，`api-review.md` 補一列 ST-15，刪掉 `AGENTS.md` 的 simplimax follow-up。

## 5. 收尾

- [x] 5.1 `gofmt -l .`、`go vet ./...`、`go test ./...`、`golangci-lint run` 都通過。
- [x] 5.2 `openspec validate simplimax-follows-gparotation --strict` 通過。
- [x] 5.3 推上 `0.4`，確認 CI 全綠，再 verify 與 archive。（466fd52f：Test 三個 OS、Reference Verification、Lint、Govulncheck、KNN／Clustering Parity 全綠）
