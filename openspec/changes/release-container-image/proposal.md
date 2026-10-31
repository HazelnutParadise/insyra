# Proposal: release-container-image

## Why

Someone who wants to analyse data with Insyra, or run their own Go analysis program, has to install Go, install the CLI and get a GPU driver working before they can start. A container image published with every release removes those steps, and pinning the image to the release it came from keeps an analysis reproducible.

## What Changes

- A `Dockerfile` at the repository root builds the image: the official Go image pinned by digest, the release's `insyra` CLI, the Vulkan loader and Mesa's Vulkan drivers, and the NVIDIA settings that make the toolkit mount the host's Vulkan driver.
- A `Container image` workflow builds `linux/amd64` and `linux/arm64` on native runners when a release is published, tests each image, and only then tags it on `ghcr.io/hazelnutparadise/insyra`.
- Tags: the exact version on every build; the minor line and `latest` only for the highest stable release by version number.
- `Docs/container.md` explains running the image, reaching the GPU and which hosts can.
- `AGENTS.md` adds the Go image to the dependency refresh before a release.

## Capabilities

### New Capabilities

- `release-container-image`: the image every release publishes, what it holds and how its tags move.

## Impact

`Dockerfile`, `.dockerignore`, `.github/workflows/container.yml`, `Docs/container.md`, `Docs/README.md`, `README.md`, `README_TW.md`, `AGENTS.md`, both CHANGELOGs. No Go code changes.
