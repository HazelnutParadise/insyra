# Tasks: ml-classes-never-nil

## 1. Tests first

- [x] 1.1 `ml/classes_nil_test.go`：未 fit 的分類器 `Classes()` 不為 nil、長度 0、`Err()` 非 nil、`.Data()` 不 panic
- [x] 1.2 `ml/mltest`：第三方回 nil 時一致性檢查回報失敗而不是 panic；空 list 要被接受

## 2. Implementation

- [x] 2.1 `ml` 的 9 個 `Classes()`：回空 list + `SetErr`
- [x] 2.2 `nn/protocol.go` 的 1 個：同上（形式一致，分支實務上不可達）
- [x] 2.3 `ml/helpers.go`：新增共用的 `noClasses`；`classes != nil` 經實測仍會觸發（`PCATransformer.Transform` 刻意傳 nil、三個呼叫端傳的是 `m.classes` 欄位而非方法），保留並註明理由
- [x] 2.4 `ml/model_selection.go:1121`、`:1270`：保留 `classes != nil`（值可能來自第三方 `ProbaModel`），拿掉多餘的 `isNilPointer(classes)`
- [x] 2.5 `ml/mltest/conformance.go`：新增 `Classes()` 非 nil 的檢查

## 3. Docs, changelog

- [x] 3.1 `Docs/ml.md`：新增 `Classes()` 說明與失敗時的行為
- [x] 3.2 `skills/insyra/SKILL.md`：示範改成檢查 `Err()`
- [x] 3.3 `CHANGELOG.md`／`CHANGELOG_TW.md`：標 BREAKING，明寫 `== nil` 的替代寫法

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`golangci-lint run` 0 issues
- [x] 4.2 `openspec validate ml-classes-never-nil --strict` 通過
