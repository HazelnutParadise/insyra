## 1. Tests

- [x] 1.1 Clear options apply to one call; two options are an error
- [x] 1.2 ClipGradNorm with a frozen parameter gives the trainable norm and leaves the frozen gradient alone
- [x] 1.3 A version-3 file with an unknown slot reports the version; Float32 and Float32Rounded share files; an exact tape refuses them; a step above MaxInt64 is refused on save
- [x] 1.4 A failed first resident Adam step leaves no moments (GPU-gated)
- [x] 1.5 Format tests rewritten for version 2; the subprocess continuation still passes bit for bit

## 2. Implementation

- [x] 2.1 `Clear(opts ...ClearOptions)`; remove `SetClearOptions`/`GetClearOptions`
- [x] 2.2 `ClipGradNorm` skips frozen parameters
- [x] 2.3 Format version 2 with metadata, single validation in the reader
- [x] 2.4 `adamRounded` releases moments it created when the first step fails

## 3. Documentation

- [x] 3.1 `Docs/nn.md`, both changelogs, `delivery-status.md`
