// +build windows

package main

import (
	"os"
)

func isatty(fd uintptr) bool {
	return true
}

// Work around to Win os.Rename implementation:
// http://code.google.com/p/go/issues/detail?id=3366
func renameFile(oldpath, newpath string) error {
	err := os.Remove(newpath)
	if err != nil {
		return err
	}
	return os.Rename(oldpath, newpath)
}
