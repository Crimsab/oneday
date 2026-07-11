//go:build !windows

package config

import (
	"os"
	"syscall"
)

func fileOwnerIDs(info os.FileInfo) (int, int, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return -1, -1, false
	}
	return int(stat.Uid), int(stat.Gid), true
}

func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

func fsyncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
