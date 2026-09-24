# Tasks: gmaps-reviews-restored

## 1. 查證
- [x] 1.1 未登入時比對各條路：`listugcposts` 403、Maps 評論頁只給 5 則、`qv9Egd` 每種排序 5 則且代碼不穩、`reviewDialog` 404、Google 搜尋的評論視窗可捲動載入。
- [x] 1.2 找出搜尋評論視窗翻頁用的 `GetLocalBoqProxy` 請求，以純 HTTP、不帶 cookie 重送並連翻 6 頁，確認每頁 10 則不重複且都有下一頁代碼。
- [x] 1.3 確認排序值 1 到 4 的意義與另一家店也能取得，並從存下的回應對出每個欄位的位置。

## 2. 測試（先寫、先看它失敗）
- [x] 2.1 本地伺服器測試：兩頁翻頁與請求內容、欄位對應、`pageCount` 停止、最小等待間隔、零值選項、缺少評論區塊時的警告。
- [x] 2.2 對現行實作跑一次，確認新測試失敗。
- [x] 2.3 `gmaps_live` 實測兩頁最新評論。

## 3. 實作
- [x] 3.1 `GetReviews` 改打 `GetLocalBoqProxy`，組 `reqpld`，依代碼翻頁。
- [x] 3.2 解析評論頁，對應欄位，`<br>` 換行、解碼 HTML 實體、時間戳轉 UTC 日期。
- [x] 3.3 移除舊的 `listugcposts` 解析與「目前無法使用」的說明。

## 4. 收尾
- [x] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`，以及 live test。
- [x] 4.2 兩份 CHANGELOG：改寫尚未發布的 `datafetch` 條目。
- [x] 4.3 `Docs/datafetch.md`：評論恢復、每頁 10 則、`ReviewDate` 格式、`ReviewerState` 與 `ReviewerLevel` 不再提供。
- [x] 4.4 `api-review.md` DF-1、`delivery-status.md`，在 #249 留言附證據並關閉。
