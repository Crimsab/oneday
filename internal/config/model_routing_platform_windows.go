//go:build windows

package config

import (
	"os"
	"syscall"
)

func fileOwnerIDs(_ os.FileInfo) (int, int, bool) {
	return -1, -1, false
}

func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return false
	}
	_ = syscall.CloseHandle(handle)
	return true
}

func fsyncDir(_ string) error {
	return nil
}
