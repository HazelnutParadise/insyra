# Change: Fold the GPU backend into the core module

## Why
Getting GPU execution today takes a second `go get` of `github.com/HazelnutParadise/insyra/accel/backend/wgpu` plus a blank import that does nothing visible. Users reasonably expect `go get github.com/HazelnutParadise/insyra` to be enough, and `go get .../insyra/accel` does not help because `accel` is a package inside the core module rather than a module of its own.

The separation was justified on three grounds, and measurement knocked two of them down:

- "It would raise everyone's minimum Go version." Not true. A `require` on a module with a higher `go` directive has no effect on consumers who never import it — verified by building a module that requires a `go 1.29` dependency without importing it. Only importers are affected, and with the default `GOTOOLCHAIN=auto` they get a newer toolchain rather than an error.
- "442,188 lines of gogpu would enter everyone's build." Not true either. Go compiles only imported packages.
- "insyra's own build and CI become dependent on a 0.x project shipping 175 releases in seven months." This one stands, and is now an accepted cost.

## What Changes
- Move the gogpu mechanics into `accel/internal/wgpu`, a leaf package that does not import `accel`
- Add an adapter inside `accel` that registers the backend during package initialisation
- Delete the `accel/backend/wgpu` module, and add its requirements to the core `go.mod`
- Remove the second `go get` and the blank import from the documented setup

## Impact
- Affected specs: `accel-gpu-execution`
- Affected code: new `accel/internal/wgpu/`, new `accel/backend_wgpu.go`, deleted `accel/backend/wgpu/`, `go.mod`, `Docs/accel.md`, README and README_TW, CHANGELOG and CHANGELOG_TW, `skills/use-insyra-cli/`
