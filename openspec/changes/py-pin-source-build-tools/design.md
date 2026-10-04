# Design: py-pin-source-build-tools

## Context

Measured on 2026-10-03 with uv 0.12.20 on macOS arm64, forcing both packages to build from source with `--no-binary-package blis --no-binary-package statsmodels`:

- `blis` 1.3.3 builds with `setuptools.build_meta` and installs `cython`, `numpy` and `setuptools`, its `[build-system] requires`; its `setup_requires` names the same two packages again.
- `statsmodels` 0.15.0 builds with `mesonpy` and installs `meson-python`, `cython`, `numpy`, `scipy` and `setuptools-scm`, their dependencies `meson`, `packaging`, `pyproject-metadata` and `setuptools`, and then `ninja`, which `mesonpy.get_requires_for_build_wheel()` adds when no `ninja` is on `PATH`.
- With the ten constrained, uv's log shows every one of them selected at its pinned version, `ninja` included, and both wheels build (about two and a half minutes on an M3).
- Replacing `cython`'s hashes in `uv.lock` with a wrong one fails the setup with `Hash mismatch for cython==3.3.0`. Replacing them in `pyproject.toml` alone is not noticed, because `uv sync --frozen` reads the build constraints from the lock.

On GitHub's `windows-11-arm` runner the same day (run 37141744239), the setup finished in about 20 seconds and Python then failed at `import polars` with `unknown feature flag: 'sse3'`: uv had installed the x86-64 CPython, which it does by default on Windows on arm64, and every package from its `win_amd64` wheel. Nothing was built from source, so no supported platform builds `blis` or `statsmodels` today.

## Decisions

### Pin the tools although nothing builds today

A source build happens when a platform runs a Python with no wheel for some package: a native arm64 Python on Windows, if uv's default changes or insyra asks for one, or any platform a later pin leaves without a wheel. Pinning the tools now costs nothing at setup, where no build runs, and closes the one way the environment could still install something unpinned and unverified.

### Every file's hash, not only Windows arm64's

Each constraint lists the SHA-256 of every file PyPI has for that version, as `uv pip compile --generate-hashes` writes them and as `uv.lock` already does for the environment's packages. Listing only the Windows arm64 wheels would keep `pyproject.toml` shorter, but a source build anywhere else, such as the check below on macOS, would then fail on a hash the list does not have. The cost is length: 197 hashes, one per line.

### Resolved for Windows on arm64

The list is resolved with `--python-platform aarch64-pc-windows-msvc` and the pinned Python, because a native arm64 Python on Windows is where the two packages lack a wheel; a marker that differs by platform resolves as it would there. Every resolved package has a Windows arm64 wheel or a pure-Python wheel, checked against PyPI, so no build tool needs a build of its own.

### Shared tools at the environment's versions

`numpy`, `scipy` and `packaging` are both build tools and packages the environment installs. They are pinned at the environment's versions: `statsmodels` and `blis` are then compiled against the `numpy` and `scipy` they run with, and nothing new is downloaded for them. A test keeps the two in step.

### The lock is checked against `pyproject.toml`

Because the lock is what the setup enforces, the test that already compares the package pins with the lock also compares the build constraints, hashes included. A constraint edited in `pyproject.toml` without running `uv lock` fails the tests instead of being silently ignored.

### A real build checks that the list is complete

The pin tests cannot tell whether the list is complete: what a build installs is decided by the source archive and its build backend at build time, and uv installs a tool no constraint names without an error. Measured by leaving `ninja` out: uv installed the newest `ninja` and the setup succeeded. So `TestSourceBuildsUseOnlyPinnedTools` does a real build. It downloads the pinned uv the way the setup does, writes the embedded `pyproject.toml` and `uv.lock` into a temporary project, and runs the setup's `uv sync` with `--no-binary-package` for every package in `knownSourceBuilds`, an empty cache and `-v`. Each tool uv's log says it installed into a build environment must be in the lock's build constraints at that version. On macOS it took five to eleven minutes, and with `ninja` left out of the lock it failed naming `ninja==1.13.2`.

The check reads uv's debug log, which is not a stable interface. uv is pinned, so the format changes only with a uv bump, and the check fails rather than passes when it finds no build requirement in the log or no `Built` line for a package it forced. It keeps its cache under a short temporary directory, because the builds run inside the cache and Windows tools that are not long-path aware fail past 260 characters.

### Checked on macOS when the pins change

Forcing the builds makes the check independent of the platform it runs on, so the `Python Build Tools` workflow runs it on `macos-latest`, whose compiler builds both packages as the local measurement did. It does not run on `windows-11-arm`: there `--python 3.12.14` gives the x86-64 CPython, so the forced builds would need an x86-64 toolchain under emulation, which is a question about the runner, not about whether the list is complete. The workflow is triggered by the paths that decide the result, not on every push, because the check downloads the whole environment and compiles two packages. The test skips itself without `INSYRA_PY_E2E`, and a `-run` pattern that matches nothing passes, so the workflow's last step fails unless the test is seen to pass, as `nn-data-gates.yml` does.

## Risks

- A future `blis` or `statsmodels` may need a tool the list does not name. The bump steps rebuild the list from the source archives' build requirements, and the build check fails until the list covers it.
- uv reuses a wheel it built from the same source archive, whatever tools built it, so a machine that built `blis` or `statsmodels` earlier would keep that build if it ever ran a native arm64 Python. No supported setup builds them today, so `Docs/py.md` does not ask anyone to clear the cache.
- On musl Linux, the lock has no musllinux wheel of `scikit-learn`, nor on arm64 of `matplotlib` or `statsmodels`, so uv would build them from source there with tools no constraint names. Whether musl hosts are supported is not decided; it is recorded as an `AGENTS.md` follow-up.
- If PyPI adds a Windows arm64 wheel of either package, the existing test that guards `knownSourceBuilds` fails and says so; the constraints can then shrink.
