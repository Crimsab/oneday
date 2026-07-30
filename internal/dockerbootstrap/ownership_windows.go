//go:build windows

package dockerbootstrap

func normalizePrivateFiles(_ string, _ ...string) error {
	return nil
}
