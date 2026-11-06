## 1. Default files appear whole

- [x] 1.1 `createWhole` writes to a temporary file and links it into place, falling back to an exclusive create
- [x] 1.2 `writeDefaultFiles` uses it, keeping its handling of a file that is already there
- [x] 1.3 Changelog entries; drop the #669 entries that recorded an unchanged type
