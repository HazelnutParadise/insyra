package insyra

import (
	"io"
	"os"
	"path/filepath"
)

// writeFileAtomically runs write against a temporary file in the target's
// directory and renames it into place only after every write and the close
// succeeded. A failure part-way therefore never leaves a truncated or empty
// file at path, and no temp file is left behind.
func writeFileAtomically(path string, write func(w io.Writer) error) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
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
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return err
	}
	return nil
}
