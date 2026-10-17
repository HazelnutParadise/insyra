package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// WriteFileAtomically writes beside the resolved target and renames only once
// writing and closing have succeeded. Symbolic links are followed, including
// links whose final target does not exist.
func WriteFileAtomically(path string, write func(w io.Writer) error) (err error) {
	target, err := resolveFinalSymlinks(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(target)
	mode := os.FileMode(0o666)
	existing := false
	if info, statErr := os.Stat(target); statErr == nil {
		mode = info.Mode().Perm()
		existing = true
	} else if !os.IsNotExist(statErr) {
		return statErr
	}

	var tmp *os.File
	for range 100 {
		var random [8]byte
		if _, err = rand.Read(random[:]); err != nil {
			return err
		}
		tmpPath := filepath.Join(dir, ".insyra-"+hex.EncodeToString(random[:])+".tmp")
		tmp, err = os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if err == nil {
			break
		}
		if !os.IsExist(err) {
			return err
		}
	}
	if tmp == nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()
	if existing {
		// OpenFile applied the umask; an existing file keeps its own mode.
		if err = tmp.Chmod(mode); err != nil {
			return err
		}
	}
	if err = write(tmp); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpPath, target); err != nil {
		return err
	}
	committed = true
	return nil
}

// resolveFinalSymlinks follows every existing symlink component while allowing
// the final target (and later path components) not to exist yet.
func resolveFinalSymlinks(path string) (string, error) {
	current, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	seen := map[string]bool{}
	for links := 0; links <= 40; links++ {
		current = filepath.Clean(current)
		volume := filepath.VolumeName(current)
		parts := strings.Split(strings.TrimPrefix(current[len(volume):], string(filepath.Separator)), string(filepath.Separator))
		built := volume + string(filepath.Separator)
		changed := false
		for i, part := range parts {
			if part == "" {
				continue
			}
			built = filepath.Join(built, part)
			info, statErr := os.Lstat(built)
			if os.IsNotExist(statErr) {
				continue
			}
			if statErr != nil {
				return "", statErr
			}
			if info.Mode()&os.ModeSymlink == 0 {
				continue
			}
			if links == 40 {
				return "", fmt.Errorf("too many symbolic links resolving %q", path)
			}
			link, readErr := os.Readlink(built)
			if readErr != nil {
				return "", readErr
			}
			if !filepath.IsAbs(link) {
				link = filepath.Join(filepath.Dir(built), link)
			}
			for _, rest := range parts[i+1:] {
				if rest != "" {
					link = filepath.Join(link, rest)
				}
			}
			current, err = filepath.Abs(link)
			if err != nil {
				return "", err
			}
			if seen[current] {
				return "", fmt.Errorf("symbolic link loop resolving %q", path)
			}
			seen[current] = true
			changed = true
			break
		}
		if !changed {
			return current, nil
		}
	}
	return "", fmt.Errorf("too many symbolic links resolving %q", path)
}
