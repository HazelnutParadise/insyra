# Tasks

## 1. Refuse an output an earlier operation read

- [x] 1.1 `TestCustomRefusesAnOutputAnEarlierOperationRead` in `nn/autodiff_custom_test.go`, ported from `0.4` 183c3508: `Custom` given an output that `Tape.Mul` already read returns an error naming `square` and `Mul` and records nothing; it fails before the change (`go test -run TestCustomRefusesAnOutputAnEarlierOperationRead ./nn/`)
- [x] 1.2 `Tape.Custom` in `nn/autodiff.go` refuses such an output with `0.4`'s check and message, and its doc comment says why; 1.1 and `go test -run 'Custom|BackwardFrom|Tape' ./nn/` pass
- [x] 1.3 `Docs/nn.md` lists the new refusal and the order it sets, and the `Tape.Custom` entry in both changelogs mentions it

## 2. Verification

- [x] 2.1 `go test ./...`, `golangci-lint run` and `openspec validate nn-custom-refuses-a-read-output --strict` pass
