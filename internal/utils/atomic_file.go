package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// WriteFileAtomically writes to a sibling temporary file and renames it into
// place only after the callback and close succeed.
//
// A path that is a symbolic link is resolved first, so the file it points to
// is replaced and the link stays a link, as writing through os.Create did. An
// existing file keeps its permission bits; a new one is created through the
// umask.
func WriteFileAtomically(path string, write func(io.Writer) error) error {
	path, err := resolveFinalSymlinks(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	mode := os.FileMode(0o666)
	existing := false
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
		existing = true
	} else if !os.IsNotExist(err) {
		return err
	}
	suffix, err := randomSuffix()
	if err != nil {
		return err
	}
	tmp, err := os.OpenFile(filepath.Join(dir, ".insyra-"+suffix+".tmp"), os.O_RDWR|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := write(tmp); err != nil {
		_ = tmp.Close()
		return err
	}
	if existing {
		if err := tmp.Chmod(mode); err != nil {
			_ = tmp.Close()
			return err
		}
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	keep = true
	return nil
}

func resolveFinalSymlinks(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	seen := make(map[string]struct{})
	for depth := 0; ; depth++ {
		if depth > 40 {
			return "", fmt.Errorf("resolve symlink %q: too many levels", path)
		}
		abs = filepath.Clean(abs)
		if _, ok := seen[abs]; ok {
			return "", fmt.Errorf("resolve symlink %q: loop detected", path)
		}
		seen[abs] = struct{}{}
		info, err := os.Lstat(abs)
		if os.IsNotExist(err) {
			return abs, nil
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return abs, nil
		}
		link, err := os.Readlink(abs)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(link) {
			link = filepath.Join(filepath.Dir(abs), link)
		}
		abs = link
	}
}

func randomSuffix() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate atomic temp name: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}
