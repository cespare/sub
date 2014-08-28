package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

const binaryDetectionBytes = 8000 // Same as git

// isBinary guesses whether a file is binary by reading the first X bytes and seeing if there are any nulls.
// Assumes the file starts seeked the beginning.
func isBinary(file *os.File) bool {
	defer file.Seek(0, 0)
	buf := make([]byte, binaryDetectionBytes)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return false
		}
		if n == 0 {
			break
		}
		for i := 0; i < n; i++ {
			if buf[i] == 0x00 {
				return true
			}
		}
		buf = buf[n:]
	}
	return false
}

// isRegular determines whether the file is a regular file or not.
func isRegular(filename string) bool {
	stat, err := os.Lstat(filename)
	if err != nil {
		return false
	}
	return stat.Mode().IsRegular()
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// tempFile is similar to ioutil.TempFile except that it creates filenames by appending a suffix to a given
// base name. (This makes it easy to generate temp files that have a 1:1 correspondence with some other set of
// files, regardless of whether those filenames are absolute or relative).
func tempFile(base, suffix string, mode os.FileMode) (f *os.File, err error) {
	for i := 0; i < 10000; i++ {
		name := base + suffix + strconv.Itoa(rand.Intn(1e9))
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, mode)
		if os.IsExist(err) {
			continue
		}
		break
	}
	return
}

const (
	ColorReset  = 0
	ColorRed    = 31
	ColorGreen  = 32
	ColorYellow = 33
	ColorBlue   = 34
)

var tty = isatty(os.Stdout.Fd())

// Copied from code.google.com/p/go.crypto/ssh/terminal.
func isatty(fd uintptr) bool {
	termios := syscall.Termios{}
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, ioctlReadTermios, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}

func colorize(s string, color int) string {
	if !tty {
		return s
	}
	return fmt.Sprintf("\x1b[%d;1m%s\x1b[%dm", color, s, ColorReset)
}

type modifier func([]byte) []byte

func highlighter(color int) modifier {
	if !tty {
		return func(b []byte) []byte { return b }
	}
	return func(b []byte) []byte {
		return []byte(fmt.Sprintf("\x1b[%d;1m%s\x1b[%dm", color, b, ColorReset))
	}
}

func highlight(b []byte, color int, ranges [][]int) []byte {
	return modifyRanges(b, ranges, highlighter(color))
}

func replacer(r *regexp.Regexp, replace []byte) modifier {
	return func(b []byte) []byte {
		submatches := r.FindSubmatchIndex(b)
		return r.Expand(nil, replace, b, submatches)
	}
}

func substitute(b []byte, find *regexp.Regexp, replace []byte, ranges [][]int) []byte {
	return modifyRanges(b, ranges, replacer(find, replace))
}

func subAndHighlight(b []byte, find *regexp.Regexp, replace []byte, color int, ranges [][]int) []byte {
	return modifyRanges(b, ranges, func(b2 []byte) []byte {
		replaced := replacer(find, replace)(b2)
		return highlighter(color)(replaced)
	})
}

// Assumes that ranges are sorted and non-overlapping.
func modifyRanges(b []byte, ranges [][]int, f modifier) []byte {
	idx := 0
	result := make([]byte, 0, len(b)) // Heuristic
	for _, interval := range ranges {
		low, high := interval[0], interval[1]
		result = append(result, b[idx:low]...)
		result = append(result, f(b[low:high])...)
		idx = high
	}
	return append(result, b[idx:]...)
}
