// +build linux darwin

package main

import (
	"os"
	"syscall"
	"unsafe"
)

// Copied from code.google.com/p/go.crypto/ssh/terminal.
func isatty(fd uintptr) bool {
	termios := syscall.Termios{}
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, ioctlReadTermios, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}

// Work around to Win os.Rename implementation:
// http://code.google.com/p/go/issues/detail?id=3366
func renameFile(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}
