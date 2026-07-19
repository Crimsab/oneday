// Package setup contains the local setup and readiness primitives used by the
// terminal CLI. It deliberately does not own configuration schema or runtime
// state.
package setup

import (
	"fmt"
	"os"
	"path/filepath"
)

const restrictiveFileMode os.FileMode = 0o600

// WriteFileAtomic replaces path only after the complete new value has reached
// disk. Setup files can contain secret references, so it never creates a
// backup and always writes with owner-only permissions.
func WriteFileAtomic(path string, data []byte) error {
	if path == "" {
		return fmt.Errorf("writing setup file: path is empty")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating setup directory: %w", err)
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return fmt.Errorf("writing setup file: %s is a directory", path)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("checking setup file: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temporary setup file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(restrictiveFileMode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("restricting temporary setup file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temporary setup file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("syncing temporary setup file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temporary setup file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replacing setup file: %w", err)
	}
	cleanup = false

	// Best-effort directory sync closes the rename durability gap on Unix. It
	// is intentionally omitted here for portability; the file itself is synced
	// before rename and the operation remains atomic on the target filesystem.
	return nil
}
