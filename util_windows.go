// +build windows

package main

func isatty(fd uintptr) bool {
	return true
}
