// +build windows

package main

import (
	"io/ioutil"
	"os"
)

func isatty(fd uintptr) bool {
	return true
}

// Work around to Win os.Rename implementation:
// http://code.google.com/p/go/issues/detail?id=3366
func renameFile(oldpath, newpath string) error {
    b, err := ioutil.ReadFile(oldpath)
    if err != nil {
        return err
    }
    err = ioutil.WriteFile(newpath, b, 0644)
    if err != nil {
        return err
    }
    return os.Remove(oldpath)
}
