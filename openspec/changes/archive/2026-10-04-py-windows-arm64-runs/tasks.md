# Tasks: py-windows-arm64-runs

## 1. 測試先行

- [x] 1.1 `py/polars_cpu_check_test.go`：用本機 Python 執行跳過檢查的那段程式，六種情況（x64 Python 在 ARM64 Windows、大小寫不同、原生 arm64 Python、真的 x64 機器、macOS、使用者已設值）；另一個測試確認它在 `import polars as pl` 之前。在沒有常數的程式上確認失敗
- [x] 1.2 `TestPinnedEnvironmentEndToEnd` 在 `windows-11-arm` 上失敗於 `import polars`（run 37141744239），作為修正前的紅燈

## 2. 實作

- [x] 2.1 `py/py.go`：新增 `skipPolarsCPUCheckUnderEmulation`，放在 `generateDefaultPyCode` 所有 import 之前；1.1 通過
- [x] 2.2 `.github/workflows/py-windows-arm64.yml`：`py/` 底下有變動就在 `windows-11-arm` 上跑 `TestPinnedEnvironmentEndToEnd`，最後確認測試真的有跑

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：Windows arm64 的說明加上 Python 在模擬下執行、為什麼跳過 polars 的檢查
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 新增條目
- [x] 3.3 `delivery-status.md`：Latest Milestones 最上方新增條目，記下擁有者的選擇

## 4. 驗證

- [x] 4.1 本機在 macOS 上跑 `TestPinnedEnvironmentEndToEnd`（`INSYRA_PY_E2E=1`），確認跳過檢查的程式不影響其他平台
- [x] 4.2 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate py-windows-arm64-runs --strict`
- [x] 4.3 push 後 `Python on Windows arm64` workflow 通過
