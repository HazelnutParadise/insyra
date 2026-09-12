# Tasks: harden-limits-and-permissions

- [x] 1.1 `ipc.WriteMessage` 的上限檢查，含「寫入端接受的長度都在讀取端範圍內」的測試。
- [x] 1.2 `PipInstall`／`PipUninstall` 的名稱檢查與 `--`，含拒絕與接受兩組樣本。
- [x] 2.1 CLI 資料庫連線改用 `gormlogger.Discard`。
- [x] 2.2 `SavePNG` 線上備援加 timeout 與 `io.LimitReader`。
- [x] 2.3 六處 `os.ModePerm` 改成 `0o755`。
- [x] 2.4 Excel 讀取加 `UnzipSizeLimit`，`csvxl` 共用同一份設定。
- [x] 2.5 IPC accept 迴圈、連線期限、socket 檔清理。
- [x] 3.1 `go test ./...`、`go vet ./...`、`golangci-lint run`。
- [x] 3.2 兩份 CHANGELOG、`api-review.md`、`delivery-status.md`。
- [ ] 3.3 關閉 #291、#292、#293、#295、#297、#339；#294 留言說明只做了 UnzipSizeLimit。
