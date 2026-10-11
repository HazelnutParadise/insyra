# Tasks: group-keys-integers-by-value

## 1. Tests first

- [x] 1.1 `groupby_integer_keys_test.go`：將 11 種整數型別、兩種浮點型別與文字 1 進行分群，得到 11、2、1 三組；`OpNUnique` 得到 3。

## 2. Implementation

- [x] 2.1 `datatable_groupby.go` (`encodeGroupKey`, `uniqueKey`) 與 `cell_identity.go` (`encodeCell`)：`uintptr` 加入整數分群處理分支。

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md`：說明 GroupBy 的分群型別鍵值規則。
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md`：Core。
- [x] 3.3 `delivery-status.md`。

## 4. Verification

- [x] 4.1 全套驗證通過。
