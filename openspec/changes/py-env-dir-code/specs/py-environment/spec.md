## ADDED Requirements

### Requirement: The environment directory is named for its Python

The environment directory SHALL be `.insyra_env/<code>_<os>_<arch>` under the working directory, where the code is `py<two-digit year><letter>` and SHALL change whenever the pinned Python version changes: the letter is `a` for the year's first new code, `b` for the second. `py/const.go` SHALL hold the code (`envDirCode`) and the Python version it was given for (`envDirPython`), and a test SHALL fail when the pinned Python differs from `envDirPython`. A change of package versions alone SHALL keep the code. The code for CPython 3.12.14 SHALL be `py26a`. insyra SHALL NOT delete the directory of an earlier code.

#### Scenario: The current directory
- **WHEN** the environment is prepared with CPython 3.12.14 pinned
- **THEN** it is built in `.insyra_env/py26a_<os>_<arch>`

#### Scenario: A Python bump without a new code
- **WHEN** `requires-python` in `py/environment/pyproject.toml` is changed and `envDirPython` is not
- **THEN** `go test ./py/` fails, naming both versions and asking for a new directory code

## MODIFIED Requirements

### Requirement: The environment follows the pins

The setup SHALL write the embedded `pyproject.toml` and `uv.lock` into the environment directory and run `uv sync --frozen --inexact --managed-python --python <pinned version>` there, with `UV_PROJECT_ENVIRONMENT` set to the directory's `.venv`, `UV_PYTHON_DOWNLOADS` set to `automatic`, and any `UV_PYTHON_PREFERENCE` of the caller removed. After a successful sync it SHALL record the pin set's fingerprint in a marker file; a later setup whose marker matches and whose interpreter exists SHALL run no uv command. A setup whose marker is missing or different SHALL sync again, so an environment synced to other package pins of the same Python is brought to the pins while keeping packages installed with `PipInstall`. A setup that fails SHALL NOT mark the environment ready or write the marker, and the next call SHALL run it again.

#### Scenario: First setup
- **WHEN** the environment directory is empty
- **THEN** the setup writes both files, runs `uv sync` once with those flags, and writes the marker

#### Scenario: Already in sync
- **WHEN** the marker matches the embedded pin set and the interpreter exists
- **THEN** the setup runs no uv command

#### Scenario: Pins changed
- **WHEN** the marker holds another pin set's fingerprint
- **THEN** the setup runs `uv sync` again and rewrites the marker

#### Scenario: The user's uv settings
- **WHEN** the caller's environment sets `UV_PYTHON_PREFERENCE`, `UV_PYTHON_DOWNLOADS=never` or `UV_PROJECT_ENVIRONMENT`
- **THEN** none of them reaches `uv sync`, and the virtual environment is built in the environment directory

#### Scenario: Sync fails
- **WHEN** `uv sync` exits with an error
- **THEN** the setup returns an error carrying uv's error output, writes no marker, and the next call runs the setup again

### Requirement: ReinstallPyEnv deletes the environment directory

`ReinstallPyEnv` SHALL hold the setup lock, mark the environment not ready, delete the environment directory, `.insyra_env/py26a_<os>_<arch>` under the working directory, with everything in it, packages installed with `PipInstall` included, and build it again from the pins. It SHALL keep the uv executable, which lives outside that directory. Its doc comment and `Docs/py.md` SHALL say what it deletes.

#### Scenario: The delete fails part-way
- **WHEN** part of the environment directory cannot be deleted
- **THEN** `ReinstallPyEnv` returns an error and the environment is not marked ready

#### Scenario: A file added to the environment
- **WHEN** a file was added to the environment directory and `ReinstallPyEnv` runs
- **THEN** the file is gone, the environment is synced and marked, and the uv executable is still in place
