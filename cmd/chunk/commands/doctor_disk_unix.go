// +build !windows

package commands

import "syscall"

// getAvailableDiskSpace returns available disk space in GB for Unix-like systems
func getAvailableDiskSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}

	// Available blocks * block size
	availableGB := (stat.Bavail * uint64(stat.Bsize)) / (1024 * 1024 * 1024)
	return availableGB, nil
}
