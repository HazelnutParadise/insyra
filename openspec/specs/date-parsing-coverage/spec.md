# date-parsing-coverage Specification

## Purpose
常見時間字串版面的辨識範圍，以及不得誤判非日期。

## Requirements
### Requirement: Common timestamp layouts parse

`TryParseTime` SHALL 接受 `2006-01-02 15:04:05`、`2006-01-02T15:04:05`、`2006-01-02 15:04` 與以 `/` 分隔的等價寫法；無時區的版面 SHALL 視為 UTC。非日期字串（純數字、單字、空字串）SHALL NOT 被視為日期。

#### Scenario: Zone-less timestamp
- **WHEN** 解析 `"2024-01-02 03:04:05"`
- **THEN** 成功，且等於 2024-01-02T03:04:05Z

