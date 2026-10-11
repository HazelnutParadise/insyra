# Tasks: cli-test-nan-equality

## 1. Tests first

- [x] 1.1 `TestApproxEqualAnyTreatsNaNOnlyAsNaN`（驗證 NaN 對數字被判定為不相等，NaN 對 NaN 為相等）

## 2. Implementation

- [x] 2.1 `approxEqualAny`：NaN 只等於 NaN
- [x] 2.2 重跑 CLI 測試，確認既有測試通過

## 3. Verification

- [x] 3.1 驗證通過：`go test ./cli/commands/...` 通過
