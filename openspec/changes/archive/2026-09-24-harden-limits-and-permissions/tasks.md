# Tasks: harden-limits-and-permissions

- [x] 1.1 `ipc.WriteMessage` 的上限檢查，含「寫入端接受的長度都在讀取端範圍內」的測試。
- [x] 2.1 CLI 資料庫連線改用 `gormlogger.Discard`。
- [x] 2.2 `SavePNG` 線上備援加 timeout 與 `io.LimitReader`。
- [x] 2.3 六處 `os.ModePerm` 改成 `0o755`。
- [x] 3.1 `go test ./...`、`go vet ./...`、`golangci-lint run`。
- [x] 3.2 兩份 CHANGELOG。
- [x] 3.3 關閉 #291、#292、#293、#295、#339。
