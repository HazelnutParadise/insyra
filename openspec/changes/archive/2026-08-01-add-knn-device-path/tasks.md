# Tasks

## 1. The socket
- [x] 1.1 Batch device-searcher hook in `stats/internal/knn`, race-safe registration, consulted only for `auto`
- [x] 1.2 Defensive validation of the answer: row count, entries per row, index range — malformed falls back
- [x] 1.3 When the device answers, the CPU searcher is never built
- [x] 1.4 `stats.RegisterKNNDeviceSearcher` forwarding

## 2. The bridge
- [x] 2.1 `accel/knnbridge` registering on blank import: pre-gates (device present, k ≤ shortlist − 1, per-row work ≥ floor), roles mapped test→dataset train→queries, un-accelerated answers discarded
- [x] 2.2 Squared distances forwarded so weighting semantics match the CPU path

## 3. Verification
- [x] 3.1 Hook tests with a fake searcher: auto-only, malformed rejection, flow-through
- [x] 3.2 Real-device parity, gated: classify, regress and neighbors via the bridge equal brute force exactly
- [x] 3.3 No-device behaviour: with the bridge imported but the accelerator disabled, results equal brute force and nothing fails
- [x] 3.4 True-direction benchmark arm — and it overturned half the original claim. Transposed (dataset=train): device wins everywhere. True direction (dataset=test, the side the kernel parallelises over): device time is flat in test rows (~467ms at 1k/2k/4k on 100k×32), so it LOSES at 1k test rows (469ms vs CPU 324ms) and wins 1.4x/2.9x/3.7x at 2k/4k/10k. The bridge gained a second floor (test rows ≥ 2048) because of this measurement

## 4. Documentation
- [x] 4.1 `Docs/stats.md` KNN section and `Docs/accel.md`; `skills/insyra/`; changelogs in both languages
- [x] 4.2 `delivery-status.md`: the ticket closes, the numbers land in the decision log
