## ADDED Requirements

### Requirement: Every release publishes a container image

When a release is published, the `Container image` workflow SHALL build the image from the release's tag for `linux/amd64` and `linux/arm64` and SHALL push it to `ghcr.io/hazelnutparadise/insyra` under the release's version without the leading `v`. A rebuild of an existing tag SHALL be possible by hand.

#### Scenario: A release is published

- **WHEN** release `v0.3.6` is published
- **THEN** `ghcr.io/hazelnutparadise/insyra:0.3.6` exists for both architectures and its `insyra version` prints `insyra v0.3.6`

### Requirement: The image holds the release it is named after

The image SHALL contain the `insyra` CLI built from the tag it is tagged with and a complete Go toolchain from the official Go image, pinned by digest. The build SHALL fail before anything is pushed when `version.go` does not state the tag's version or the image's Go is older than the tag's `go` directive.

#### Scenario: The tag and version.go disagree

- **WHEN** the tag is `v0.3.6` and `version.go` says `0.3.5`
- **THEN** the workflow fails and pushes nothing

### Requirement: A tag moves only after both architectures pass

Each architecture SHALL be built on its own native runner, pushed by digest and checked by running `insyra version` from that digest. The tags SHALL be attached to the multi-architecture image only after both checks pass.

#### Scenario: One architecture fails

- **WHEN** the arm64 image reports a different version
- **THEN** no tag is created or moved

### Requirement: Floating tags follow version numbers

The minor-line tag (`0.3`) SHALL point to the highest stable release of that line and `latest` SHALL point to the highest stable release overall, both decided by version number among published releases. A prerelease SHALL get only its own tag.

#### Scenario: A patch to an older line

- **WHEN** `0.4.0` is released and then `0.3.7`
- **THEN** `0.3` points to `0.3.7` and `latest` still points to `0.4.0`

#### Scenario: A prerelease

- **WHEN** `v0.4.0-rc1` is published as a prerelease
- **THEN** the image is tagged `0.4.0-rc1` only

### Requirement: The image can reach the host's GPU through Vulkan

The image SHALL contain the Vulkan loader and Mesa's Vulkan drivers, and SHALL ask the NVIDIA Container Toolkit for the `graphics` capability alongside `compute` and `utility`, so that `--gpus all` on an NVIDIA host and `--device /dev/dri` on an AMD or Intel host expose a hardware adapter to `accel`.

#### Scenario: No GPU is reachable

- **WHEN** the image runs without a GPU, or on Docker Desktop for Windows or macOS
- **THEN** `insyra accel devices` reports that no accelerator was detected and the CLI works on the CPU

### Requirement: The image works for a non-root user

The image SHALL set `HOME` and `GOCACHE` to directories any user can write, so that a container started with `--user` can run the CLI, keep its environments and build Go programs.

#### Scenario: Running as the host user

- **WHEN** the image runs with `--user 1000:1000`
- **THEN** `insyra newdl 1 2 3 as x` and `go run` both succeed
