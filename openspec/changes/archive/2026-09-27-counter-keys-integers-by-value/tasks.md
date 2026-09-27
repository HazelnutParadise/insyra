# Tasks: counter-keys-integers-by-value

## 1. Tests first

- [x] 1.1 `counter_integers_test.go`：混寬度整數合併成 `int` key 且與 `Count` 一致；CSV 讀入的欄 `counter[5]` 查得到；兩欄不同寬度在 `DataTable.Counter` 合併；`int` 裝不下的值以 `uint64` 合併，負數不與無號數合併；`ToMapKey(uint8(9))` 是 `int`（先紅：四個測試全部失敗，`counter[5]` 在 CSV 欄是 0）

## 2. Implementation

- [x] 2.1 `cell_identity.go`：`integerKey` 依 `integerParts` 把整數轉成 `int`，裝不下時用 `int64`／`uint64`；`ToMapKey` 先走它，`Counter` 經由 `ToMapKey` 一併生效

## 3. Docs, changelog, follow-ups

- [x] 3.1 `Docs/DataList.md`、`Docs/DataTable.md`：Counter 說明整數以數值當 key，查詢用字面值、`ToMapKey` 或 `Count`；修正 `ToToMapKey` 錯字
- [x] 3.2 `skills/insyra/SKILL.md`：整數照數值比對與 `Counter` 的 key 型別
- [x] 3.3 `CHANGELOG.md`／`CHANGELOG_TW.md`：Core 新增 BREAKING；先前 `Counter` 條目與此矛盾的句子移除，錯字修正
- [x] 3.4 `AGENTS.md`：移除 `Counter()` follow-up
- [x] 3.5 `delivery-status.md`

## 4. Verification

- [x] 4.1 業界對照：pandas 2.3.3 的 `value_counts` 與 `collections.Counter` 把 `int64`、`int`、`int8` 的 5 併成一個，`"5"` 分開；R 的 `table` 合併 `5L` 與 `5`
- [x] 4.2 `go test ./...`、`golangci-lint run`、`GOOS=linux GOARCH=386 go vet .`、`openspec validate counter-keys-integers-by-value --strict`
