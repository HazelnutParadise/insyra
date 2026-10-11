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
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		path = resolved
	}
	dir, base := filepath.Dir(path), filepath.Base(path)
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
	tmp, err := os.OpenFile(filepath.Join(dir, "."+base+".tmp-"+suffix), os.O_RDWR|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if err := write(tmp); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if existing {
		if err := tmp.Chmod(mode); err != nil {
			_ = tmp.Close()
			cleanup()
			return err
		}
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

func randomSuffix() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate atomic temp name: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}
