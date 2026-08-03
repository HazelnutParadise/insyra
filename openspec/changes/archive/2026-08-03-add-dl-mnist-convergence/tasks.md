# Tasks

## 1. Ingredients

- [x] 1.1 IDX reader (test-side): magic/dims validation, images to f32 scaled [0,1], labels to int64; malformed files refused with names
- [x] 1.2 Seeded He-style init helper (test-side), deterministic under a fixed seed

## 2. Convergence

- [x] 2.1 Ungated micro-convergence: tiny two-class problem to 100% in bounded steps
- [x] 2.2 Gated MNIST run behind `INSYRA_DL_MNIST_DIR`: 784→128→10 MLP, Adam, shuffled minibatches, bounded epochs, ≥95% test accuracy, loss-curve sanity asserted; clean skip without data
- [x] 2.3 Existing suites pass unchanged

## 3. Sync

- [x] 3.1 Docs/dl.md training section shows the convergence loop; changelogs both languages; skills
