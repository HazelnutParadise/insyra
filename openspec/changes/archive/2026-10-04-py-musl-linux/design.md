# Design: py-musl-linux

## Context

Measured on 2026-10-04 with the pinned uv 0.12.20 in `alpine:3.22` containers, with only `build-base` installed:

- uv installed `cpython-3.12.14-linux-aarch64-musl`, and on aarch64 built `scikit-learn`, `statsmodels` and `matplotlib` from source in 1006 seconds on an M3. Every tool the three builds installed came from the new constraints, `ninja` and `patchelf` included, which meson-python asked for because neither was on `PATH`. All twelve packages the preamble imports then imported.
- With the constraints `py-pin-source-build-tools` left, forcing a `scikit-learn` build failed at once: `cython>=3.1.2,<3.2.6` against `cython==3.3.0`.
- `scikit-learn` treats OpenMP as optional, and Alpine's `gcc` brings `libgomp`, so the build uses it there.

## Decisions

### One list of build constraints for every source build

A constraint applies to every build that installs that tool, so the list has to satisfy all four packages at once. It is resolved from their `[build-system] requires` together, each range written as given, with `numpy`, `scipy` and `packaging` at the environment's versions, plus `ninja` and `patchelf` for meson-python. The resolution gives the same twelve versions for `x86_64-unknown-linux-musl`, `aarch64-unknown-linux-musl`, `aarch64-pc-windows-msvc` and both macOS targets, so a single list serves every platform. `patchelf` has no Windows wheel, but meson-python asks for it only on Linux.

### The wheel test knows glibc from musl

The old Linux patterns, `linux[0-9_]*_x86_64.whl`, matched `musllinux_1_2_x86_64.whl` as well as manylinux, so a package with only manylinux wheels passed for musl. The patterns now require `manylinux…` for `linux/amd64` and `linux/arm64` and `musllinux_<major>_<minor>_…` for the two new `linux-musl` platforms, and `knownSourceBuilds` lists `scikit-learn` for `linux-musl/amd64` and `matplotlib`, `scikit-learn` and `statsmodels` for `linux-musl/arm64`. The Go side needs no change: the environment directory stays `py26a_linux_<arch>`, and uv chooses the musl CPython itself.

### Checked in an Alpine container on both architectures

The workflow runs the tests through `docker run` on GitHub's `ubuntu-latest` and `ubuntu-24.04-arm` runners, in `golang:1.26.8-alpine`, the Go release `go.mod` names. Running docker from a step instead of a job container keeps `actions/checkout` on the host, where it works on both architectures. The container installs only `build-base`, the compiler `Docs/py.md` asks a musl user for, and leaves `ninja` and `patchelf` out on purpose, so meson-python asks for them and the constraints have to cover them.

Both tests run on both architectures, because each builds a different set of packages. The build check builds the packages `knownSourceBuilds` lists for the platform it runs on, as the setup there does, where it used to force every listed package so that it could run anywhere. Forcing built things no setup builds: measured on 2026-10-04, `blis`, which a musl host installs from a wheel, failed in Alpine under gcc with `unrecognized command-line option '--rtlib=compiler-rt'`, because setuptools passes on the flags the musl CPython was built with, which are clang's, and under clang with `inline assembly requires more registers than available` in `blis`'s armv8a kernel. The meson builds of the other three ignore those flags. The check tells a musl Linux from a glibc one by its dynamic loader, `/lib/ld-musl-<arch>.so.1`, and skips on a platform that builds nothing, such as macOS, so the macOS workflow goes. The two `windows/arm64` entries are not built anywhere, since no supported setup builds them; the resolution over every listed package's build requirements is what keeps the constraints acceptable to them. Both tests skip themselves without `INSYRA_PY_E2E`, so each job's last step fails unless its test is seen to pass. The workflow is triggered by the files that decide the build, not by every change to `py/`, because each run compiles several packages.

### A longer ceiling for the tests

The end-to-end test allowed 30 minutes for the setup and the checks together, and the build check 45; an Alpine arm64 setup alone took 17 minutes on an M3, and a four-core runner is slower. Both now allow 75 minutes, under the 85-minute `go test` timeout and the 100-minute job limit.

## Risks

- A musl user without a C and C++ compiler gets a setup that fails while building the first of these packages; `Docs/py.md` says to install one first.
- The first setup on musl takes many minutes, on aarch64 most of all. uv keeps the wheels it builds in its cache, so a second environment on the same machine does not compile them again.
- `matplotlib`'s build needs network access to the FreeType, HarfBuzz, libraqm, Qhull and SheenBidi download sites; the hashes its wraps pin keep what it downloads fixed.
