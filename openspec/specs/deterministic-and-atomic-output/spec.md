# deterministic-and-atomic-output Specification

## Purpose
The file geocode cache is written atomically, through a temporary file renamed into place, so a reader never sees a half-written cache.
## Requirements
### Requirement: File geocode cache is written atomically

`fileGeocodeCache.Set` SHALL 先寫入暫存檔再 rename 到目標路徑。

#### Scenario: Cache file after Set
- **WHEN** `Set` 之後讀取目錄
- **THEN** 只有目標檔，無殘留 `.tmp`，內容可解析

