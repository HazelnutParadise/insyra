# Tasks: lint-catches-leaks-and-lost-errors

## 1. 真 bug 先紅
- [x] 1.1 `state.json`、`history.txt`、`config.json` 存在但讀不到時，不加 `--force` 的 import 要失敗、環境不變。
- [x] 1.2 剛建立、什麼都沒有的環境，不加 `--force` 的 import 要成功。
- [x] 1.3 修 `isEnvironmentEmpty`：檔案不存在算空，其他讀取錯誤回報。

## 2. 開 linter、清告警
- [x] 2.1 `.golangci.yml` 開五個 linter。
- [x] 2.2 `errorlint` 55 處、`rowserrcheck` 4 處、`bodyclose` 1 處誤報。

## 3. 文件與紀錄
- [x] 3.1 兩份 CHANGELOG（CLI 修正、錯誤可被 `errors.Is` 認出）。
- [x] 3.2 全套驗證；`api-review.md` RP-8、`delivery-status.md`；歸檔、寫 Purpose；在 #280 留言（RP-7 仍開著）。
