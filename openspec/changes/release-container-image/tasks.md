# Tasks

- [x] `Dockerfile` and `.dockerignore` at the repository root; the image builds locally from this branch and from `v0.3.5`.
- [x] `.github/workflows/container.yml`: native builds per architecture, push by digest, `insyra version` check, tags attached afterwards by version number.
- [x] `Docs/container.md`, linked from `Docs/README.md`, `README.md` and `README_TW.md`.
- [x] `AGENTS.md`: the Go image is part of the dependency refresh before a release, and the package's visibility is checked after the first push.
- [x] Entries under `## Unreleased` in both CHANGELOGs.
- [x] `openspec validate release-container-image --strict` passes.
- [ ] The workflow runs on the first release after this change, and `ghcr.io/hazelnutparadise/insyra` is set to public.
- [ ] GPU passthrough checked on a Linux host with a GPU.
