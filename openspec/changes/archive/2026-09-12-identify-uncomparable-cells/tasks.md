# Tasks: identify-uncomparable-cells

## 1. 先重現
- [x] 1.1 端到端：sqlite BLOB 欄讀進來，`Counter()` panic、`Count()` 回 0。
- [x] 1.2 `[]any{1}` 與 `[]any{"1"}` 在 GroupBy 被併成一組。
- [x] 1.3 `[2]any{[]int{1},2}` 與 `struct{X any}` 用 `reflect.TypeOf(...).Comparable()` 判斷會漏掉。

## 2. 編碼
- [x] 2.1 `encodeCell(v any) string`：沿用既有的 `n:`／`s:`／`b:`／`i:`／`f:` 純量規則，遞迴進入 slice／array／map／struct。
- [x] 2.2 map 依編碼後的 key 排序輸出，與迭代順序無關。
- [x] 2.3 pointer／chan／func 以位址編碼，與 `==` 的語意一致。
- [x] 2.4 深度上限 64，超過退回位址，避免自參照值造成 fatal stack overflow。
- [x] 2.5 `encodeGroupKey` 與 `uniqueKey` 的 default arm 改呼叫它；純量輸出逐位元組不變。

## 3. 替身與查詢
- [x] 3.1 `UncomparableKey{Type string; content string}` 與 value receiver 的 `String()`（型別 + 截斷內容，bytes 用十六進位）。
- [x] 3.2 `KeyOf(v any) any`，用 `reflect.ValueOf(v).Comparable()`，純量先走 type switch 快路。
- [x] 3.3 `DataList.Counter` 與 `DataTable.Counter` 改用 `KeyOf`。

## 4. 相等
- [x] 4.1 `equalCell` 對不可比較的值改比編碼。
- [x] 4.2 `valueMatcher` 的 fallback 同上，讓 `Count`／`FindAll`／`Replace`／`DropAll` 與 `Counter` 一致。

## 5. 測試
- [x] 5.1 BLOB 端到端：`Counter` 不 panic 且為 2、`Count` 為 2。
- [x] 5.2 `[2]any`、`struct{X any}` 兩種 TypeOf 判不出來的情況。
- [x] 5.3 `[]any{1}` vs `[]any{"1"}` 在 GroupBy 與 Counter 都分開。
- [x] 5.4 map 編碼與迭代順序無關（重複跑多次）。
- [x] 5.5 自參照切片不崩。
- [x] 5.6 純量的 group key 與變更前相同（釘住不退步）。
- [x] 5.7 `counter[KeyOf(v)]` 查得到；`counter[1]`、`counter["a"]` 照舊。
- [x] 5.8 列印含大型二進位值的計數結果仍可讀。

## 6. 收尾
- [x] 6.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 6.2 兩份 CHANGELOG。
- [x] 6.3 `Docs/DataList.md`、`Docs/DataTable.md` 補 `KeyOf` 與 `UncomparableKey`；skill 若提到 Counter 一併更新。
- [x] 6.4 `AGENTS.md`：`labelKey` 的同病記成 follow-up；`Counter` 的 `int(1)`／`int64(1)` 那條註明本次未裁決。
- [x] 6.5 `delivery-status.md`。
