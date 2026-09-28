# Proposal

## Why

`Tape.Custom` accepts an output that an operation recorded earlier on the tape has already read. The reverse pass visits operations in reverse order, so it reaches the custom operation before that reader, and the reader's gradient never reaches the custom operation's inputs: in the `0.4` review's probe, `dz/da` came back 0 instead of 20, with no error. The `nn-training` spec says a malformed custom operation is an error, never a silent gradient. `Custom` is unreleased on this line (added after v0.3.3), and `0.4` fixed the same code in 183c3508.

## What Changes

- `Tape.Custom` refuses an output that an operation already on the tape read as an input, with an error naming that operation and telling the caller to record the custom operation first, and records nothing.
- `Docs/nn.md` and the changelog entry for `Custom` say so.

## Capabilities

### New Capabilities

### Modified Capabilities
- `nn-training`: the malformed-custom-operation requirement also refuses an output an earlier operation read.

## Impact

- `nn/autodiff.go` (`Custom`), `nn/autodiff_custom_test.go`.
- `Docs/nn.md`, both changelogs.
