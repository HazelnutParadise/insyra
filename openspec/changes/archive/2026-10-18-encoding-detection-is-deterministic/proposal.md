# Proposal: encoding-detection-is-deterministic

## Why

chardet runs its recognizers concurrently and sorts their answers by confidence only, so recognizers that tie come back in completion order. `DetectBest` returned the first, and a 24-byte Big5 CSV was read as Shift_JIS, GB18030 or Big5 from one call to the next (AGENTS.md follow-up 2026-10-13). Reproduced: two calls in one process gave `gb-18030` and `big5`.

## What Changes

- `detectEncodingOfSample` takes every answer from `DetectAll` and, among those with the highest confidence, picks the first by charset name. One input always gets one answer.
- Not changed: whether a low-confidence guess should be an error. That alters what a read does with such a file and stays a decision for the owner.

## Capabilities

### Modified Capabilities

- `core-encoding-detection`: the answer for a given input is stable.

## Impact

`utils.go`, `detect_encoding_stable_test.go`, both CHANGELOGs, the follow-up entry in `AGENTS.md` (narrowed to the confidence question).
