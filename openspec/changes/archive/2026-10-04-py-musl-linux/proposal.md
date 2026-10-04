# Proposal: py-musl-linux

## Why

The pinned uv is the static musl build so that it runs on any Linux, and on a musl host such as Alpine it installs a musl CPython, which installs only musllinux wheels. The lock has no musllinux wheel of `scikit-learn` 1.9.1, nor on aarch64 of `matplotlib` 3.11.2 or `statsmodels` 0.15.0, so uv builds them from source there. Nothing checked that. The wheel test's Linux patterns accepted a manylinux wheel for a musl host too. And the build constraints `py-pin-source-build-tools` added, resolved for `blis` and `statsmodels` alone, pin `cython` 3.3.0 and `meson-python` 0.22.1, which `scikit-learn`'s build requirements refuse (`cython>=3.1.2,<3.2.6`, `meson-python>=0.17.1,<0.22.0`). Measured on 2026-10-04, forcing a `scikit-learn` build with them fails with "your requirements are unsatisfiable", so the setup on Alpine could not succeed.

The owner decided on 2026-10-04 that musl Linux is a supported host, answering the `AGENTS.md` follow-up the review of `py-pin-source-build-tools` recorded.

## What Changes

- The build constraints are resolved again over the build requirements of all four packages some platform builds from source, `blis`, `statsmodels`, `scikit-learn` and `matplotlib`, plus `ninja` and `patchelf`, which meson-python asks for when they are not on `PATH` (`patchelf` only on Linux). The result is twelve tools: `cython` 3.2.5, `meson` 1.12.1, `meson-python` 0.21.1, `ninja` 1.13.2, `numpy` 2.5.3, `packaging` 26.3, `patchelf` 0.19.1.0, `pybind11` 3.1.0, `pyproject-metadata` 0.12.1, `scipy` 1.18.1, `setuptools` 84.0.0 and `setuptools-scm` 9.2.2, each with the SHA-256 of every file, 208 in all. The resolution is the same for musl Linux on x86_64 and aarch64, for Windows on arm64 and for macOS, and every tool has a musllinux wheel on both architectures, so none is itself built from source. The environment's own resolution does not change.
- The wheel test tells the two Linux libraries apart: a glibc host needs a manylinux wheel, a musl host (`linux-musl/amd64`, `linux-musl/arm64`) a musllinux one, and `knownSourceBuilds` lists the three musl source builds.
- `matplotlib`'s build downloads the sources of FreeType, HarfBuzz, libraqm, Qhull and SheenBidi through meson; each wrap in its source archive pins a SHA-256 that meson checks, so those downloads are verified as well.
- `TestSourceBuildsUseOnlyPinnedTools` builds the packages `knownSourceBuilds` lists for the platform it runs on, as the setup there does, rather than every listed package; it tells a musl Linux from a glibc one by its `/lib/ld-musl-*.so.1` loader, and skips on a platform that builds nothing, such as macOS. Forcing every package built things no setup builds: `blis`, which a musl host installs from a wheel, failed in Alpine under gcc, which rejects the clang flags the musl CPython passes to a setuptools build, and under clang, which rejects `blis`'s armv8a assembly.
- A new workflow, `Python on musl Linux`, runs `TestPinnedEnvironmentEndToEnd` and `TestSourceBuildsUseOnlyPinnedTools` in a `golang:1.26.8-alpine` container on amd64 and arm64 with only `build-base` installed, whenever a file that decides the build changes, and fails unless each test is seen to pass. It replaces the macOS `Python Build Tools` workflow, where the build check now has nothing to build.
- The end-to-end test and the build check allow 75 minutes instead of 30 and 45: an Alpine arm64 setup compiled three packages in 17 minutes on an M3.
- `Docs/py.md` says what a musl host builds, that it needs a C and C++ compiler (`apk add build-base`), and how long the first setup takes; the bump steps resolve the build constraints for every platform in `knownSourceBuilds`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `py-environment`: musl Linux is a supported host, built and checked in CI, and the build check runs there.

## Impact

- Pins: `py/environment/pyproject.toml`, `py/environment/uv.lock`.
- Tests: `py/environment_pins_test.go`, `py/environment_build_test.go`, `py/environment_setup_test.go`.
- CI: `.github/workflows/py-musl.yml` (new), `.github/workflows/py-build-tools.yml` (removed).
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`, `AGENTS.md` (the follow-up removed).
