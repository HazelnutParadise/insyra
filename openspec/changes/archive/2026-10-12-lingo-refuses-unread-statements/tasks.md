# Tasks: lingo-refuses-unread-statements

## 1. Tests first

- [x] 1.1 `lpgen/lingo_strict_test.go`：未加分號的最後一句、認不得的敘述、非線性的關係與目標函數、讀不出變數的宣告都回傳含行號與敘述的錯誤；註解被略過；行內註解後的敘述照讀；`@GIN`、`@FREE`、`@BND` 讀進對應欄位；舊名稱仍照舊略過
- [x] 1.2 `lp/lingo_free_test.go`：含 `@FREE` 的 LINGO 模型經 `lp.Solve` 得到 -3

## 2. Implementation

- [x] 2.1 `lpgen/lingo.go`：解析器分嚴格與寬鬆兩種，`ParseLingo`、`ParseLingoFile` 用嚴格版，兩個舊名用寬鬆版（與過去相同）；嚴格版讀 `@GIN`、`@FREE`、`@BND`、略過註解、拒絕其餘敘述與未結束的最後一句

## 3. Docs and changelog

- [x] 3.1 `Docs/lpgen.md`：`ParseLingo`／`ParseLingoFile` 讀什麼、拒絕什麼
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`lpgen` 條目
