//go:build !windows

package main

import "syscall"

func getDiskInfoPlatform(path string) (total, free, used uint64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0
	}

	total = stat.Blocks * uint64(stat.Bsize)
	free = stat.Bavail * uint64(stat.Bsize)
	used = total - free
	return
}
