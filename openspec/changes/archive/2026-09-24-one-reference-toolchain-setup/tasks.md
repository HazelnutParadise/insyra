# Tasks: one-reference-toolchain-setup

## 1. 實作
- [x] 1.1 新增 composite action：Python 3.12、R release、共用的 Python 與 R 套件，外加額外 Python 套件的輸入。
- [x] 1.2 三個 workflow 改用它；`reference-verification.yml` 傳入額外套件並保留 PyTorch 步驟。

## 2. 驗證與紀錄
- [x] 2.1 推上 `0.4` 後三個 workflow 都實跑通過，log 顯示各自的測試真的執行。
- [x] 2.2 `api-review.md` RP-7 與對照列、`delivery-status.md`；歸檔；關 #280。
