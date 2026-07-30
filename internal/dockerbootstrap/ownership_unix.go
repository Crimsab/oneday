//go:build unix

package dockerbootstrap

import (
	"fmt"
	"os"
	"syscall"
)

func normalizePrivateFiles(root string, paths ...string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	for _, path := range paths {
		if err := os.Chown(path, int(stat.Uid), int(stat.Gid)); err != nil {
			return fmt.Errorf("preserving host ownership for %s: %w", path, err)
		}
	}
	return nil
}
