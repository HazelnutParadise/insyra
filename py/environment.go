package py

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
)

// The pinned environment. Every version the py package installs lives in
// py/environment: pyproject.toml pins uv, Python and each package, uv.lock
// records the whole resolution with the hash of every file, and
// uv-sha256.sum is the sha256.sum the uv release publishes. Docs/py.md,
// "Pinned versions", says how to bump them.

//go:embed environment/pyproject.toml
var envPyproject []byte

//go:embed environment/uv.lock
var envLock []byte

//go:embed environment/uv-sha256.sum
var uvSHA256Sums []byte

var (
	requiredUVVersion     = regexp.MustCompile(`(?m)^required-version = "==([0-9]+\.[0-9]+\.[0-9]+)"\r?$`)
	requiredPythonVersion = regexp.MustCompile(`(?m)^requires-python = "==([0-9]+\.[0-9]+\.[0-9]+)"\r?$`)
)

// pinnedUVVersion returns the uv version environment/pyproject.toml requires.
func pinnedUVVersion() (string, error) {
	m := requiredUVVersion.FindSubmatch(envPyproject)
	if m == nil {
		return "", errors.New("environment/pyproject.toml has no exact [tool.uv] required-version")
	}
	return string(m[1]), nil
}

// pinnedPythonVersion returns the Python version environment/pyproject.toml
// requires.
func pinnedPythonVersion() (string, error) {
	m := requiredPythonVersion.FindSubmatch(envPyproject)
	if m == nil {
		return "", errors.New("environment/pyproject.toml has no exact requires-python")
	}
	return string(m[1]), nil
}

// uvArchives names the uv release archive for each supported platform. Linux
// takes the static musl build, which runs on any Linux: uv finds the host's C
// library itself when it picks a Python build.
var uvArchives = map[string]string{
	"darwin/amd64":  "uv-x86_64-apple-darwin.tar.gz",
	"darwin/arm64":  "uv-aarch64-apple-darwin.tar.gz",
	"linux/amd64":   "uv-x86_64-unknown-linux-musl.tar.gz",
	"linux/arm64":   "uv-aarch64-unknown-linux-musl.tar.gz",
	"windows/amd64": "uv-x86_64-pc-windows-msvc.zip",
	"windows/arm64": "uv-aarch64-pc-windows-msvc.zip",
}

// uvArchive returns the uv release archive for goos/goarch and the SHA-256
// the release publishes for it.
func uvArchive(goos, goarch string) (name, sum string, err error) {
	name, ok := uvArchives[goos+"/"+goarch]
	if !ok {
		return "", "", fmt.Errorf("no uv build is pinned for %s/%s", goos, goarch)
	}
	for _, line := range strings.Split(string(uvSHA256Sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return name, fields[0], nil
		}
	}
	return "", "", fmt.Errorf("environment/uv-sha256.sum has no checksum for %s", name)
}

// uvReleaseURL is where uv's release archives are downloaded from.
const uvReleaseURL = "https://github.com/astral-sh/uv/releases/download"

// maxUVArchiveSize bounds a uv download. uv 0.12.20's largest archive is
// 23 MB.
const maxUVArchiveSize = 128 << 20

// uvDownloadClient bounds a whole download, so a stalled connection cannot
// hold the setup forever.
var uvDownloadClient = &http.Client{Timeout: 30 * time.Minute}

// uvExecutableName is the name of the uv executable on this platform.
func uvExecutableName() string {
	if runtime.GOOS == "windows" {
		return "uv.exe"
	}
	return "uv"
}

// uvBinaryPath is where the pinned uv is kept: beside the environment
// directory envDir, so ReinstallPyEnv does not download it again.
func uvBinaryPath(envDir, version string) string {
	return filepath.Join(filepath.Dir(envDir), "uv-"+version+"_"+runtime.GOOS+"_"+runtime.GOARCH, uvExecutableName())
}

// ensureUV returns the path of the pinned uv for the environment directory
// envDir, downloading and verifying it the first time.
func ensureUV(ctx context.Context, envDir string) (string, error) {
	version, err := pinnedUVVersion()
	if err != nil {
		return "", err
	}
	dest := uvBinaryPath(envDir, version)
	if info, err := os.Stat(dest); err == nil && info.Mode().IsRegular() {
		return dest, nil
	}
	name, sum, err := uvArchive(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}
	url := uvReleaseURL + "/" + version + "/" + name
	insyra.LogInfo("py", "init", "Downloading uv %s from %s", version, url)
	if err := installUV(ctx, uvDownloadClient, url, sum, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// installUV downloads the uv release archive at url, checks that its SHA-256
// is wantSum, and writes the uv executable inside it to dest. Nothing is
// written when the download fails, the checksum differs or the archive holds
// no uv.
func installUV(ctx context.Context, client *http.Client, url, wantSum, dest string) error {
	archive, err := downloadBounded(ctx, client, url, maxUVArchiveSize)
	if err != nil {
		return err
	}
	got := sha256.Sum256(archive)
	if hex.EncodeToString(got[:]) != strings.ToLower(wantSum) {
		return fmt.Errorf("%s has SHA-256 %x, but the uv release publishes %s for it; refusing to use it", url, got, wantSum)
	}
	var exe []byte
	if strings.HasSuffix(url, ".zip") {
		exe, err = uvFromZip(archive)
	} else {
		exe, err = uvFromTarGz(archive)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", url, err)
	}
	return writeExecutable(dest, exe)
}

// downloadBounded returns the body of a GET of url, refusing one longer than
// limit bytes.
func downloadBounded(ctx context.Context, client *http.Client, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: %s", url, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%s is larger than %d bytes", url, limit)
	}
	return body, nil
}

// uvFromTarGz returns the file named uv inside a uv release .tar.gz, which
// keeps it in a directory named after the target.
func uvFromTarGz(archive []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, errors.New("the archive holds no uv executable")
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag == tar.TypeReg && path.Base(hdr.Name) == "uv" {
			return io.ReadAll(tr)
		}
	}
}

// uvFromZip returns uv.exe from a uv release .zip, which keeps it at the top.
func uvFromZip(archive []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || path.Base(f.Name) != "uv.exe" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		return io.ReadAll(rc)
	}
	return nil, errors.New("the archive holds no uv.exe")
}

// writeExecutable writes data to dest through a temporary file in the same
// directory, so no one ever runs a partly written executable.
func writeExecutable(dest string, data []byte) error {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".uv-download-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // nothing left to remove after the rename
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	// Flush before the rename, so a crash cannot leave an empty file under
	// the final name.
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dest); err != nil {
		// Another process may have put the same verified file there first.
		if info, serr := os.Stat(dest); serr == nil && info.Mode().IsRegular() {
			return nil
		}
		return err
	}
	return nil
}

// envMarker is the file in the environment directory that records which pin
// set the environment was last synced to.
const envMarker = ".insyra-env.sha256"

// pinFingerprint identifies the embedded pin set.
func pinFingerprint() string {
	h := sha256.New()
	h.Write(envPyproject)
	h.Write([]byte{0})
	h.Write(envLock)
	return hex.EncodeToString(h.Sum(nil))
}

// venvPython returns the interpreter inside the virtual environment venvDir.
func venvPython(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(venvDir, "bin", "python")
}

// envInSync reports whether envDir was synced to the embedded pins and still
// has its interpreter.
func envInSync(envDir string) bool {
	marker, err := os.ReadFile(filepath.Join(envDir, envMarker))
	if err != nil || strings.TrimSpace(string(marker)) != pinFingerprint() {
		return false
	}
	_, err = os.Stat(venvPython(filepath.Join(envDir, ".venv")))
	return err == nil
}

// syncEnvironment writes the pinned project into envDir and has uv make its
// virtual environment match the lock, keeping packages added with
// PipInstall. The marker is written last, so an interrupted sync is redone.
func syncEnvironment(ctx context.Context, uv, envDir string) error {
	python, err := pinnedPythonVersion()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		return err
	}
	marker := filepath.Join(envDir, envMarker)
	if err := os.Remove(marker); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for name, data := range map[string][]byte{"pyproject.toml": envPyproject, "uv.lock": envLock} {
		if err := os.WriteFile(filepath.Join(envDir, name), data, 0o644); err != nil {
			return err
		}
	}
	// --frozen installs exactly what the embedded lock records, from the
	// URLs it records; --locked would re-check the lock against the user's
	// own uv settings, such as a mirror index, and refuse it.
	cmd := exec.CommandContext(ctx, uv, "sync", "--frozen", "--inexact", "--managed-python", "--python", python)
	cmd.Dir = envDir
	cmd.Env = uvSyncEnv(filepath.Join(envDir, ".venv"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	utils.ApplyHideWindow(cmd)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("uv sync failed: %w. stderr: %s", err, strings.TrimSpace(stderr.String()))
	}
	return os.WriteFile(marker, []byte(pinFingerprint()+"\n"), 0o644)
}

// uvSyncEnv is the environment uv sync runs in: the caller's, without the
// settings that would move the virtual environment away from venv or keep uv
// from the pinned, uv-managed Python. UV_PYTHON_PREFERENCE conflicts with
// --managed-python whatever its value, and UV_PYTHON_DOWNLOADS=never would
// stop uv fetching the pinned Python.
func uvSyncEnv(venv string) []string {
	env := make([]string, 0, len(os.Environ())+2)
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		switch strings.ToUpper(name) { // Windows names are case-insensitive
		case "UV_PROJECT_ENVIRONMENT", "UV_PYTHON_PREFERENCE", "UV_PYTHON_DOWNLOADS":
			continue
		}
		env = append(env, kv)
	}
	return append(env, "UV_PROJECT_ENVIRONMENT="+venv, "UV_PYTHON_DOWNLOADS=automatic")
}
