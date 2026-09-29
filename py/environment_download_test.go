package py

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// tarGz builds a .tar.gz holding files, laid out like a uv release archive.
func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// zipOf builds a .zip holding files, laid out like a uv release archive.
func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// serve returns a server that answers every request with body.
func serve(t *testing.T, body []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestInstallUVTakesTheExecutableOutOfEitherArchive(t *testing.T) {
	for _, tc := range []struct {
		archive string
		body    []byte
	}{
		{"uv-x86_64-unknown-linux-musl.tar.gz", tarGz(t, map[string]string{
			"uv-x86_64-unknown-linux-musl/uvx": "not uv",
			"uv-x86_64-unknown-linux-musl/uv":  "the uv executable",
		})},
		{"uv-x86_64-pc-windows-msvc.zip", zipOf(t, map[string]string{
			"uvx.exe": "not uv",
			"uv.exe":  "the uv executable",
		})},
	} {
		srv := serve(t, tc.body)
		dest := filepath.Join(t.TempDir(), "bin", "uv")
		if err := installUV(context.Background(), srv.Client(), srv.URL+"/0.0.0/"+tc.archive, sha256Hex(tc.body), dest); err != nil {
			t.Fatalf("%s: %v", tc.archive, err)
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "the uv executable" {
			t.Errorf("%s: wrote %q, want the uv executable", tc.archive, got)
		}
		info, err := os.Stat(dest)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
			t.Errorf("%s: %s is not executable: %v", tc.archive, dest, info.Mode())
		}
	}
}

// SEC-10 of #289: the install script this replaces ran whatever the server
// sent.
func TestInstallUVRefusesAnArchiveWhoseChecksumDiffers(t *testing.T) {
	body := tarGz(t, map[string]string{"uv-x86_64-unknown-linux-musl/uv": "a tampered uv"})
	srv := serve(t, body)
	dir := t.TempDir()
	dest := filepath.Join(dir, "uv")
	want := strings.Repeat("0", 64)

	err := installUV(context.Background(), srv.Client(), srv.URL+"/0.0.0/uv-x86_64-unknown-linux-musl.tar.gz", want, dest)
	if err == nil {
		t.Fatal("installUV accepted an archive whose checksum differs")
	}
	if !strings.Contains(err.Error(), sha256Hex(body)) || !strings.Contains(err.Error(), want) {
		t.Errorf("the error %q does not name both checksums", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("a refused archive left %d files behind", len(entries))
	}
}

func TestInstallUVRefusesAnArchiveWithoutUV(t *testing.T) {
	body := tarGz(t, map[string]string{"uv-x86_64-unknown-linux-musl/uvx": "not uv"})
	srv := serve(t, body)
	dest := filepath.Join(t.TempDir(), "uv")
	err := installUV(context.Background(), srv.Client(), srv.URL+"/0.0.0/uv-x86_64-unknown-linux-musl.tar.gz", sha256Hex(body), dest)
	if err == nil {
		t.Fatal("installUV accepted an archive without uv in it")
	}
	if _, serr := os.Stat(dest); !errors.Is(serr, os.ErrNotExist) {
		t.Errorf("a refused archive left %s behind", dest)
	}
}

func TestInstallUVReportsAFailedDownload(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	err := installUV(context.Background(), srv.Client(), srv.URL+"/0.0.0/uv.tar.gz", strings.Repeat("0", 64), filepath.Join(t.TempDir(), "uv"))
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("a 404 gave %v, want an error with the status", err)
	}
}

func TestInstallUVStopsWithItsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	srv := serve(t, []byte("never read"))
	err := installUV(ctx, srv.Client(), srv.URL+"/0.0.0/uv.tar.gz", strings.Repeat("0", 64), filepath.Join(t.TempDir(), "uv"))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled download gave %v, want context.Canceled", err)
	}
}

// The executable is kept beside the environment directory, so deleting the
// environment does not delete it.
func TestUVIsKeptBesideTheEnvironment(t *testing.T) {
	envDir := filepath.Join("root", ".insyra_env", "py25c_x")
	got := uvBinaryPath(envDir, "1.2.3")
	if filepath.Dir(filepath.Dir(got)) != filepath.Dir(envDir) {
		t.Errorf("uv is kept at %s, want it beside %s", got, envDir)
	}
	if !strings.Contains(got, "uv-1.2.3_"+runtime.GOOS+"_"+runtime.GOARCH) {
		t.Errorf("uv's directory %s does not name the version and platform", got)
	}
}
