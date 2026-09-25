## Why

`TestSequentialMNISTConvergence` checks that the sugar changes nothing by comparing its loss curve against two numbers transcribed from one machine (`0.350281`, `0.163855`) and an accuracy of exactly `95.84%`. The first CI run of the new data-gates workflow failed on it: on `ubuntu-latest` the second epoch's mean loss is `0.163840` and the accuracy `95.90%`. The hand-written tape run in the same job produced `0.163840` too, so the two paths still agree digit for digit — what drifted is f32 arithmetic between arm64 and amd64, which the recorded pair cannot survive.

The property the spec states is that Sequential reproduces **the hand-written tape run**. Comparing against that run instead of against a transcription keeps the proof exact and makes it true on every platform.

## What Changes

- `TestSequentialMNISTConvergence` runs the documented hand-written tape loop in the same process, under the same seed and hyperparameters, and requires both epochs' mean losses and accuracies to be identical between the two paths.
- The transcribed constants go. What is left of the absolute claim is platform-independent: the second epoch is below half the first, and the accuracy is at least 95%.

No library code changes, and nothing user-visible, so there is no changelog entry.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `nn-training`: the sugar-changes-nothing scenario compares the two runs against each other rather than against recorded values.

## Impact

- `nn/sequential_convergence_test.go`.
- The Neural Network Data Gates workflow goes green; `TestSequentialFitMNISTConvergence` keeps transcribed constants and is recorded as a follow-up.
