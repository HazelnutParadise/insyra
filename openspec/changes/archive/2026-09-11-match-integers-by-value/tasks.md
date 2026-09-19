# Tasks: match-integers-by-value

## 1. Tests first

- [x] 1.1 根套件：int64 資料配 Go int 字面值，涵蓋 DataList 與 DataTable 的每個依值查找、計數、取代、刪除方法；無號與負數、小數與整數不相等、NaN 仍能找到；IsEqualTo 維持型別嚴格（先紅：18 個方法與無號、int32 兩例全部失敗）
- [x] 1.2 根套件：`OrdinalEncode` 的 `Order: []any{1, 2, 3}` 對 int64 欄；`LabelEncoder` 在 int 上 fit、在 int64 上 Transform；同欄 int 與 int64 是同一個類別（先紅：`value 1 is not in Order`）
- [x] 1.3 `cli/commands`：`count`、`find`、`replace`、`encode ordinal` 對 int64 資料（先紅：印出 0、`[]`、沒取代、全部 nil）

## 2. Implementation

- [x] 2.1 一個比對規則：`valueMatcher` 依要找的值挑一次比較方式，整數依值相等，其餘沿用原本的 `==` 與 NaN 規則
- [x] 2.2 所有依值查找、計數、取代、刪除的方法都經過這個規則
- [x] 2.3 編碼器的類別鍵依整數值產生（`labelKey`）
- [x] 2.4 效能：第一版逐格判斷讓一百萬格的小數、字串 `Count` 慢了三倍，改成每次呼叫挑一次後，整數持平，小數 2.3→1.5 ms、字串 2.7→2.0 ms；基準測試留在 `value_lookup_bench_test.go`

## 3. Docs, changelog, follow-ups

- [x] 3.1 `Docs/DataList.md`、`Docs/DataTable.md`：寫明值怎麼比對；順手把誤放在「Searching」底下的 ExecuteCCL 全有全無說明移回原位
- [x] 3.2 CLI usage 參考的字面值說明
- [x] 3.3 `CHANGELOG.md`／`CHANGELOG_TW.md`
- [x] 3.4 `AGENTS.md`：刪除已解決的 follow-up，新增 `Counter` 的混型鍵
- [x] 3.5 `delivery-status.md`

## 4. Verification

- [x] 4.1 `go test ./...` 全綠、`-race` 跑根套件與 `cli/commands`、`golangci-lint run` 0 issues
- [x] 4.2 CLI 在隔離的 HOME 實跑：one-shot 下 `count t 2` 得 2、`find t 2` 得 `[1 3]`、`replace` 生效、`encode ordinal order 1,2,3 unknown error` 得到序號；同一個腳本裡 `groupby` 產生的 int 欄仍能用 `find` 找到
- [x] 4.3 `openspec validate match-integers-by-value --strict`
