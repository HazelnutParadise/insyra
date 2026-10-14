# Tasks: datalist-transforms-fail-like-the-rest

## 1. Tests first

- [x] 1.1 新測試 `datalist_failure_shape_test.go`：`Normalize` 失敗回傳接收者且內容不變；`Sample`、`SampleFrac` 的各種失敗、`Rank` 與 `Shift` 收到兩個選填值時回傳帶錯誤的空 list、接收者也有錯誤且內容不變；空 list 抽樣不是錯誤（舊程式碼：十個子測試紅、空 list 測試綠）

## 2. Implementation

- [x] 2.1 `datalist.go`：`Normalize` 失敗時回傳 `dl`；`Rank` 收到兩個旗標時回傳 `failedResult()`
- [x] 2.2 `datalist_window.go`：`Shift` 收到兩個填補值時回傳 `failedResult()`
- [x] 2.3 `datalist_sampling.go`：`Sample`、`SampleFrac` 的失敗回傳 `failedResult()`
- [x] 2.4 釘住舊結果的 `TestChainableTransformsNeverReturnNil`（拿掉 `Normalize`）與 `TestDataListNormalize` 改成新規則

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataList.md`：Error Handling 一節寫明兩種失敗形狀；`Normalize` 的 Returns；`Rank` 的失敗說明（原本寫回傳 nil，與程式不符）；抽樣錯誤帶著錯誤
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：Core，BREAKING
- [x] 3.3 `AGENTS.md`：刪除「five `DataList` transforms fail in a shape the others do not」follow-up
- [x] 3.4 `api-review.md`：D-7 一列記下本 change；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate datalist-transforms-fail-like-the-rest --strict`
