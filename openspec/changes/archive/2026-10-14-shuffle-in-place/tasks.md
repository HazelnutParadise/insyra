# Tasks: shuffle-in-place

## 1. Tests first

- [x] 1.1 新測試 `shuffle_in_place_test.go`：`Shuffle` 原地打亂並回傳接收者、保留名稱；`Shuffled` 回傳副本並保留接收者；同一 seed（0 到 49）兩者順序相同；兩個 options 時 `Shuffle` 不動接收者、`Shuffled` 回傳帶錯誤的空 list；空 list 不記錄錯誤；DataTable 的對應情況與欄長不一的表格（舊程式碼沒有 `Shuffled`，編譯失敗即為紅）
- [x] 1.2 CLI 測試：`shuffle … as` 對 DataList 與 DataTable 都存成副本、原變數不變（舊程式碼為綠；只改核心、還沒改 CLI 時為紅，已確認）

## 2. Implementation

- [x] 2.1 `datalist_sampling.go`：`Shuffle` 改成原地，新增 `Shuffled`
- [x] 2.2 `datatable_sampling.go`：`Shuffle` 改成原地（欄位與列名一起移動，短欄補 nil），新增 `Shuffled`
- [x] 2.3 `interfaces.go`：`IDataList`、`IDataTable` 加入 `Shuffled`
- [x] 2.4 `internal/dsl/commands/sample.go`：`shuffle` 改呼叫 `Shuffled`；`ml` 的 fold 切分維持呼叫 `Shuffle`（作用在它自己建的表上，結果相同）
- [x] 2.5 既有的 seed 重現測試改用 `Shuffled`（原地版本對同一個 list 打亂兩次，原本的比較會變成恆真）

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataList.md`：「In Place or a New List」把 `Shuffle` 移到原地組、`Shuffled` 放進新 list 組，說明動詞與 `-ed` 的成對規則；Sample / SampleFrac / Shuffle / Shuffled 一節
- [x] 3.2 `Docs/DataTable.md`：Sample / SampleFrac / Shuffle / Shuffled / TrainTestSplit 一節
- [x] 3.3 `CHANGELOG.md`／`CHANGELOG_TW.md`：Core，BREAKING
- [x] 3.4 `skills/insyra/SKILL.md`：方法要嘛改變自己、要嘛回傳新值的原則，以及文件位置
- [x] 3.5 `api-review.md`：D-7 標為已修正；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate shuffle-in-place --strict`
