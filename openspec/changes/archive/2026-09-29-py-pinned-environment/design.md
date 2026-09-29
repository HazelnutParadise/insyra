## Context

`py` builds its Python environment the first time a function needs it (`pyEnvInit`), in `.insyra_env/py25c_<os>_<arch>` under the working directory. Today that means: find or install uv (via its install script), `uv init --bare -p ==3.12.*`, `uv sync`, then `uv pip install` each of twelve packages at its newest version. The directory name's `py25c` is a version code the project bumped by hand when the environment changed (`py25b` before 2025-10-19).

## Goals / Non-Goals

**Goals:**

- The same uv, Python and package versions on every machine, each verified against a published or recorded checksum.
- One directory holds every pin, and a test catches pins that disagree.
- An installed environment follows the pins after an upgrade without the user doing anything, and keeps the packages they added.
- A failed or cancelled setup leaves nothing marked ready.

**Non-Goals:**

- Deciding whether the setup should stay implicit (#254).
- Moving to a newer Python minor.
- Pinning the packages a user installs with `PipInstall`.

## Decisions

**uv comes from the release archive, not the install script.** The script downloads the same archive and checks the same checksum, but it runs a remote script, installs uv for the whole account and edits the shell profile. Downloading the archive directly lets the checksum be checked against a copy embedded in the module, so the trust anchor is the reviewed source tree rather than whatever the server returns at run time; fetching the `.sha256` file at run time would only catch corruption. The embedded file is the release's `sha256.sum`, byte for byte, so anyone can confirm it with `cmp`.

**A `uv` on `PATH` is ignored.** Using it would bring back a version that differs per machine. The cost is one download of about 20 MB per uv version.

**Linux uses the static musl build on every host.** uv's own installer picks the glibc build when the host's glibc is new enough and falls back to musl, which needs `ldd` parsing. uv decides which Python build to download from the host's C library, found through the ELF interpreter of `/bin/sh` (`crates/uv-platform/src/libc.rs` at tag 0.12.20), not from how uv itself was built, so the musl build behaves the same on a glibc host and needs no detection in Go.

**`uv sync --frozen --inexact --managed-python --python <pinned>` builds the environment.** `--frozen` installs exactly what the embedded lock records, from the URLs it records. `--locked` was the first choice and the review of this change measured it failing: it re-checks the lock against the user's own uv settings, so a mirror index, `exclude-newer`, `prerelease` or `resolution` in the user's configuration made uv call the lock stale on every run. The pyproject and the lock are embedded together and a test keeps them consistent, so the run-time check adds nothing. `--inexact` keeps packages added with `PipInstall`. `--managed-python` with the exact version passed as `--python` makes uv use its own verified CPython 3.12.14 even when a `.python-version` file or another Python is around. uv runs in the caller's environment minus three variables: `UV_PROJECT_ENVIRONMENT` is replaced with the directory's `.venv`, so a user setting cannot move the environment away from where `py` looks for its interpreter; `UV_PYTHON_PREFERENCE` is dropped, because uv refuses it next to `--managed-python` whatever its value; and `UV_PYTHON_DOWNLOADS` is set to `automatic`, because `never` would stop uv fetching the pinned Python.

**The pins are read from `pyproject.toml`.** The uv version is `[tool.uv] required-version = "==X"`, which uv also enforces itself, and the Python version is `requires-python = "==X"`. Go reads both with an anchored regular expression over the embedded file rather than a TOML parser, which would be a new dependency for two lines of a file the project writes; the pins test fails if either stops matching.

**A marker file, not the directory name, tracks the pin set.** After a successful sync the setup writes the SHA-256 of the embedded `pyproject.toml` and `uv.lock` to `.insyra-env.sha256` in the environment directory. A later run whose marker matches and whose interpreter exists calls no uv command. Any other state writes the files and syncs, which is also how an environment built by an older insyra is converted. Bumping the directory name instead, as `py25b` to `py25c` did, would leave the old environment on disk and drop the user's added packages on every bump. The marker is removed before a sync starts, so an interrupted sync is redone.

**The context reaches the setup.** Setup runs under `pyInitMu`, a lock a caller can stop waiting for when its context ends (a one-slot channel rather than a `sync.Mutex`), so a call with a short deadline does not wait out another call's download. A cancelled caller aborts only its own attempt: the next caller takes the lock, finds the environment not ready and starts again. A download is written to a temporary file and renamed into place, and the marker is written last, so an aborted attempt leaves nothing that looks finished.

**Tests use the test binary as a stand-in uv.** `TestMain` turns the binary into a fake uv when an environment variable asks for it; a test copies the binary to where the setup expects uv, and the fake records its arguments, creates the interpreter file, fails, or sleeps. This runs on all three CI systems and needs no hook in production code. The download is tested through `installUV` with an `httptest` server and archives built in the test. The one test that uses the network builds the real environment and runs only when `INSYRA_PY_E2E=1`.

**`.gitattributes` marks `py/environment/*` as binary.** Git on the Windows CI runners converts text files to CRLF on checkout, which would change the embedded bytes and so the marker fingerprint per platform. The regular expressions also accept a trailing `\r`.

## Risks / Trade-offs

- [The first run after upgrading can downgrade a package the user had upgraded themselves] → Stated in the changelog as a breaking behaviour change. It is what pinning means; `PipInstall` after the sync still installs another version on top.
- [GitHub or PyPI unavailable] → The setup returns the download error; nothing is marked ready, and the next call retries. Same as today.
- [Air-gapped users with their own uv on `PATH` now need the uv download] → Accepted for this change. A way to point `py` at a local uv could be added if someone needs it.
- [A pinned package gets a security fix] → Bumping is documented and checked by the pins test; it is a release-time task like the Go dependency refresh.
- [`uv sync` still reads the rest of the user's uv configuration] → Kept on purpose, so a proxy, certificates or cache location still apply. With `--frozen`, the files and their hashes come from the lock whatever index the configuration names.
- [Windows on arm64 has no wheels of blis or statsmodels at the pinned versions] → uv builds them from source there, which needs a C compiler; their build dependencies are not pinned or hash-checked. A test fails if any other platform loses a wheel, or if these two gain one, so the exception list stays true. The docs say so.
- [A sync killed part-way] → The marker is not written, so the next call syncs again. Whether uv then repairs a package it had half installed was not tested.
- [A working directory on a `noexec` file system] → The pinned uv lives under the working directory, so it cannot run there, where uv installed in the home directory could. Not tested; the error from `exec` names the path.
- [Two programs in the same working directory] → uv takes its own lock on the project environment, so their syncs queue. One program rewriting `pyproject.toml` while the other's uv reads it can fail that run, which then leaves no marker and is retried. Not observed.
- [A crash during the uv download] → The temporary `.uv-download-*` file stays beside the uv directory. The executable is flushed before it is renamed into place, so the final name never holds a partial file.
