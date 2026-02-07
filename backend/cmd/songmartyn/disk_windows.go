//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

func getDiskInfoPlatform(path string) (total, free, used uint64) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	pathPtr, _ := syscall.UTF16PtrFromString(path)
	var freeBytesAvailable, totalBytes, totalFreeBytes uint64

	ret, _, _ := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if ret == 0 {
		return 0, 0, 0
	}

	return totalBytes, freeBytesAvailable, totalBytes - freeBytesAvailable
}
