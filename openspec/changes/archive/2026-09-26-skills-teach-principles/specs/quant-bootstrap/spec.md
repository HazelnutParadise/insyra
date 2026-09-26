## REMOVED Requirements

### Requirement: Docs, skills and changelogs describe the bootstrap API
**Reason**: The agent skills no longer carry API examples (`agent-skills`). The skill clause is dropped; the rest is kept unchanged under the renamed requirement below.
**Migration**: `Docs/quant.md` already holds the bootstrap section and fan-chart example the skill repeated.

## ADDED Requirements

### Requirement: Docs and changelogs describe the bootstrap API

`Docs/quant.md` SHALL 涵蓋 `BootstrapConfig`、`BlockBootstrap`、`BootstrapResult`、`PercentileBands` 的章節與扇形圖用法範例，並說明 seed 永遠生效、`BlockSize` 在兩種方法下的意義、以及區塊長度與樣本數的統計建議。`Docs/README.md`、`README.md`、`README_TW.md` 的 quant 列 SHALL 提及 bootstrap 路徑模擬。`CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` SHALL 各新增 `` ### `quant` `` 條目並連結 issue #199。

#### Scenario: Documentation set is complete

- **WHEN** 變更完成
- **THEN** 上述每個檔案都包含對應的新內容，英文與繁體中文版本同步
