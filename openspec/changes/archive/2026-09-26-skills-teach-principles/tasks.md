# Tasks: skills-teach-principles

## 1. Audit and migration
- [x] 1.1 逐條盤點兩個 skill（含 references）對 `Docs/` 與 CLI `help` 的涵蓋：COVERED／MISSING／PRINCIPLE／OUTDATED
- [x] 1.2 CLI skill 的 MISSING 條目補進 `Docs/cli-dsl.md`（與 `Docs/accel.md`），OUTDATED 條目不搬；兩條 UNVERIFIED 先實測，確認才寫
- [x] 1.3 Go skill 的 MISSING 條目補進對應的 `Docs/*.md`

## 2. Skills
- [x] 2.1 重寫 `skills/use-insyra-cli/SKILL.md`：使用時機、心智模型、可重現性原則、從執行檔查指令、文件位置；刪除 `references/`
- [x] 2.2 重寫 `skills/insyra/SKILL.md`：使用時機、心智模型、跨套件慣例、查證與驗證流程、版本正確的查詢路徑與 Docs 地圖；刪除 `references/`

## 3. Contracts and records
- [x] 3.1 `cli/commands/docs_sync_test.go` 只比對 `Docs/cli-dsl.md`；證明缺一個指令時測試會失敗
- [x] 3.2 `AGENTS.md` 的同步規則與 Agent Skills 段、`cli/AGENTS.md`、`README.md`、`README_TW.md`
- [x] 3.3 `openspec validate skills-teach-principles --strict`、`go test ./...`、`golangci-lint run`
