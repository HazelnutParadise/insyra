# Proposal: py-pinned-environment

## Why

SEC-10 of the second review round ([#289](https://github.com/HazelnutParadise/insyra/issues/289)), and the `ReinstallPyEnv` point of PY-1 ([#254](https://github.com/HazelnutParadise/insyra/issues/254)). The first time a `py` function runs, it builds a Python environment in `.insyra_env/py25c_<os>_<arch>` under the working directory, and nothing in that chain is pinned or verified:

- When `uv` is not on `PATH`, the setup runs `curl -LsSf https://astral.sh/uv/install.sh | sh` (falling back to `wget`), or `irm https://astral.sh/uv/install.ps1 | iex` on Windows: a remote script executed unread, installing whichever uv is current into the user's account. The script also edits the shell profile to put uv on `PATH` unless `UV_NO_MODIFY_PATH` is set (checked in the 0.12.20 installer).
- When `uv` is on `PATH`, whatever version it is gets used.
- Python is `3.12.*`, so the patch depends on the day, and uv may pick a Python already on the machine instead of a build it downloads.
- The twelve packages are installed one by one with `uv pip install <name>`, each at its newest version, and their dependencies float too. Two machines set up a week apart run different numpy and pandas.
- The setup marks the environment ready before its first step runs, so a setup that fails part-way is never retried: the next call skips it and runs whatever was half built.
- `ReinstallPyEnv` deletes the whole install directory, packages added with `PipInstall` included, while `Docs/py.md` says it removes "the existing virtual environment". It also resets the ready flag without the lock the setup holds.

## What Changes

- **uv is pinned and verified.** The setup downloads the uv 0.12.20 release archive for the platform from `github.com/astral-sh/uv/releases`, checks its SHA-256 against the checksum the release publishes, and keeps the executable in `.insyra_env/uv-0.12.20_<os>_<arch>/`. The checksums are a byte-for-byte copy of the release's `sha256.sum` (`https://github.com/astral-sh/uv/releases/download/0.12.20/sha256.sum`), embedded in the module, so a download that does not match is refused. No install script runs and a `uv` on `PATH` is not used. uv keeps the CPython build and its cache in its usual per-user directories, as before. Linux takes the static musl build, which runs on any Linux; uv detects the host's C library on its own when it picks a Python build. macOS, Linux and Windows on amd64 and arm64 are supported; anything else is an error naming the platform.
- **Python is pinned.** The environment runs CPython 3.12.14, the newest 3.12 patch, which uv 0.12.20 can download for all six platforms. It is always a uv-managed build (`--managed-python`), whose download uv checks against the checksum it carries, never a Python already on the machine. The 3.12 line is kept: moving to 3.13 or 3.14 changes the language user code runs under, and is a separate decision.
- **Every package is pinned, with its dependencies.** The twelve packages are pinned `name==version` at their newest releases as of 2026-09-30, and `uv.lock` records the resolution of all 73 packages, with the SHA-256 of every file. The setup installs them with `uv sync --frozen`, from the PyPI URLs and with the hashes the lock records, whatever index the user's uv configuration names. On Windows on arm64, blis and statsmodels have no wheels at these versions and are built from source.
- **The pins live in one place**, `py/environment/`: `pyproject.toml` holds the uv version (`[tool.uv] required-version`), the Python version (`requires-python`) and the packages; `uv.lock` the resolution; `uv-sha256.sum` the uv checksums. `Docs/py.md` says how to bump them, and a test checks that the files agree with each other and with the packages the Python preamble imports.
- **BREAKING (behaviour)**: an environment is brought to the pinned set on first use. After an upgrade, the first call writes the pinned project into the existing directory and runs `uv sync`, which moves each pinned package to its pinned version, downgrading one that was newer. Packages added with `PipInstall` stay (`--inexact`), unless the environment's Python is not 3.12.14, in which case uv rebuilds the virtual environment. A marker file records the pin set the environment was synced to, so later runs start without calling uv.
- A setup that fails is not marked ready, and the next call runs it again.
- The context given to `RunCodeContext`, `RunCodefContext`, `RunFileContext` and `RunFilefContext` also bounds the first-use setup: cancelling it stops the download or the sync, and the call returns `ctx.Err()`.
- `ReinstallPyEnv` holds the setup lock, deletes `.insyra_env/py25c_<os>_<arch>` with everything in it, and builds it again from the pins. The uv executable, kept beside it, is not downloaded again. Its doc comment and `Docs/py.md` say so.
- The setup no longer prints a progress bar to standard output; it logs through insyra's logger as before.

Whether `RunCode` should keep setting up on first use at all is #254's open question and is not part of this change.

## Capabilities

### New Capabilities

- `py-environment`: how the Python environment is pinned, verified, built, kept in sync with its pins and rebuilt.

### Modified Capabilities

(none)

## Impact

- New files: `py/environment/pyproject.toml`, `py/environment/uv.lock`, `py/environment/uv-sha256.sum` (embedded), `py/environment.go`, and `.gitattributes`, which keeps the three files byte-identical on Windows checkouts.
- Code: `py/init.go` (the setup), `py/const.go` (`pythonVersion` goes, `uvPath` joins), `py/py.go` (`ReinstallPyEnv`, the `Pip…` functions call the pinned uv, the runners pass their context to the setup).
- Tests: the pins agree; each supported platform has an archive and a checksum; a download whose checksum does not match is refused and writes nothing; the executable is extracted from both archive kinds; the setup runs `uv sync` once per pin set, retries after a failure, stops when its context ends; `ReinstallPyEnv` removes the whole directory. They run against a stand-in uv, the test binary itself, so they need no network. One more test builds the real environment from scratch in a temporary directory and runs Python in it; it needs the network and runs only when `INSYRA_PY_E2E=1`.
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md` (SEC-10; a note on PY-1), `delivery-status.md`. `skills/insyra/SKILL.md` teaches nothing about the environment and does not change.
