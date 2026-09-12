# Tasks: test-unpinned-behaviour

## 1. Documented DataTable behaviour (#304)

- [x] 1.1 `datatable_merge_leftright_test.go`: 左右合併，鍵欄位版本，逐格值與列順序照 `Docs/DataTable.md` 的範例。
- [x] 1.2 同檔案：以列名合併的左右版本（`colIdx1 == -1` 的那條路徑）。
- [x] 1.3 `datatable_sort_stability_test.go`: 升序、降序、多層全平手、列名隨列移動，四個穩定性測試。
- [x] 1.4 對照 `Docs/DataTable.md` 的敘述確認測試釘的就是文件寫的；文件若有錯，改文件而不是改測試。

## 2. internal/core (#305)

- [x] 2.1 `ring_test.go`: 空 ring 的 `Get`/`PopFront`/`PopBack`、`NewRing(0)` 與負容量、成長、`PopFront` 後 `Push` 的環繞、`ToSlice` 順序、`Clear` 後重用。
- [x] 2.2 同檔案：`DeleteAt` 的頭、中、尾與越界。
- [x] 2.3 `biindex_more_test.go`: `Get`、`Has`、`Len`、`IDs`、`DeleteByName`、`Clear`；空名稱與負 id 被拒絕；`Set` 把名字從舊 id 搬走。
- [x] 2.4 同檔案：`Clone` 與原本互不影響，`Clone` 在 nil receiver 上回傳空的 BiIndex。
- [x] 2.5 `atomic_test.go`: 序列化、同 actor 重入、跨 actor 重入走 inline 並觸發 `TrustZoneFallbackHook`。
- [x] 2.6 同檔案：`AtomicDoN` 對重複與 nil actor 的處理、重入時走 inline；`Close` 後 `AtomicDo` 仍執行 f、`IsClosed` 在 nil receiver 上回傳 true。
- [x] 2.7 `go test -race ./internal/core/` 通過。

## 3. lp 的解析函式 (#306)

- [x] 3.1 `lp/parse_test.go`: `parseGLPKOutputFromFile` 讀一個固定樣本檔，斷言列數、`Rows:`/`Columns:` 的改寫、空行被略過。
- [x] 3.2 同檔案：檔案不存在時的回傳。
- [x] 3.3 `extractIterationNodeCounts` 取最後一筆；沒有匹配時回傳空字串。
- [x] 3.4 `extractWarnings` 串接多筆、沒有時回傳空字串。
- [x] 3.5 `createAdditionalInfoDataTable` 的列名順序固定。
- [x] 3.6 測試不呼叫 `glpsol`，在沒有安裝 GLPK 的機器上也會執行。

## 4. parquet 的 CCL 介接層 (#308)

- [x] 4.1 `parquet/ccl_test.go`: 在 `t.TempDir()` 寫一個 Parquet 檔當共用 fixture。
- [x] 4.2 `FilterWithCCL`：條件成立的列、全部不符、條件語法錯誤、欄位不存在。
- [x] 4.3 `ApplyCCL`：新增欄位後重讀檔案確認寫回；語法錯誤時原檔不被破壞。
- [x] 4.4 `parquetContext` 的存取方法：`GetCol`/`GetColByName`/`GetCell`/`GetCellByName`/`GetRowAt`/`GetColData`/`GetColDataByName`/`GetAllData`/`SetRowIndex`/`GetColIndexByName`/`GetRowIndexByName`，含越界與名稱不存在的回傳。

## 5. 收尾

- [x] 5.1 `go test ./...` 全綠；`go test -race` 跑過本次動到的套件。
- [x] 5.2 `golangci-lint run` 無問題（含 gofmt）。
- [ ] 5.3 重新量測四個套件的覆蓋率，把數字記在 `delivery-status.md`。
- [ ] 5.4 `api-review.md` 的 TS-7、TS-8、TS-9、TS-10、TS-12 標為已修正；關閉 #304、#305、#306、#308 並附證據。
- [ ] 5.5 測試過程中發現、但超出本次範圍的問題記到 `AGENTS.md` 的 `## Follow-ups`。
