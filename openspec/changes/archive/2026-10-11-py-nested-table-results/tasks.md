# Tasks: py-nested-table-results

## 1. 測試先行

- [x] 1.1 `py/nested_result_test.go`：dict 結果的表格欄位與分數都解碼；`map[string]*insyra.DataTable` 與 `[]*insyra.DataList` 逐一解碼；內嵌 `*insyra.DataTable` 的包裝型別照舊拿到整張表；沒有表格的 struct 照舊走 JSON；自己解 JSON 的型別照舊；空 DataFrame 保留欄名。在舊程式上確認失敗
- [x] 1.2 `py/nested_result_rules_test.go`：`encoding/json` 的欄位規則（支配、`-`、`-,`、`,string`、同層同名遮蔽、自我內嵌不再 stack overflow）、map key 支援整數與 `UnmarshalText`、解進既有值保留結果沒給的部分、沒有 tag 的內嵌表格或清單從整個值解碼並在收到另一種值時回錯。在舊程式上 8 個失敗（含 stack overflow），三個突變都會被抓到

## 2. 實作

- [x] 2.1 `py/pyresult_decode.go`：空表不再重設欄名
- [x] 2.2 `py/pyresult_decode.go`：結果型別在 struct 欄位、map 值、slice 或 array 元素裡有表格或清單時，逐一解碼；1.1、1.2 通過
- [x] 2.3 `py/builtin.go`：`insyra.Return` 以 `json.dumps` 的 `default=` 把 dict、list、tuple 裡的 DataFrame 與 Series 轉成格式，一般值不多做任何處理

- [x] 2.4 對抗式審查後：欄位規則改為移植 `encoding/json` 的 `typeFields`（支配、`-`、`-,`、`,string`、只在 key 用到時配置內嵌指標），map key 支援整數與 `UnmarshalText`，解進既有值時保留結果沒給的部分；沒有 tag 的內嵌表格或清單從整個值解碼，收到另一種值回傳錯誤
- [x] 2.5 第二輪審查後：只有 `*DataTable`／`*DataList` 指標內嵌才拿整個值，旁邊有其他欄位時只接受有型別標記的格式或陣列；內嵌 `IDataTable`／`IDataList` 回到以型別命名的具名欄位；同層兩個同型別內嵌互相遮蔽；`,string` 解失敗不留半寫的值；slice 解進既有元素
- [x] 2.6 複審後：結果本身是表或清單時，名叫 `DataTable`／`DataList` 的欄位（不論是否內嵌）拿整個結果，只有 dict 結果才逐欄解碼；欄名用 `strings.EqualFold` 比對；`py/port_review_test.go` 在前一版上失敗

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：Return Type Binding 說明巢狀的表格與清單、包裝型別的條件、空 DataFrame
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 段落末尾新增條目

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）
