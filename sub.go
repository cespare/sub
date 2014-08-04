package main

import (
	"bufio"
	"fmt"
	"github.com/mattn/go-colorable"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"time"

	flag "github.com/cespare/pflag"
)

const (
	binaryDetectionBytes = 8000 // Same as git
)

var (
	dryRun  bool
	verbose bool
	out     io.Writer
)

func init() {
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "Print out what would be changed without changing any files.")
	flag.BoolVarP(&verbose, "verbose", "v", false, "Print out detailed information about each match.")
	flag.Usage = func() { usage(0) }
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	out = colorable.NewColorableStdout()
}

func usage(status int) {
	fmt.Printf(`Usage:
		%s [OPTIONS] <FIND> <REPLACE> <FILE1> <FILE2> ...
where OPTIONS are
`, os.Args[0])
	flag.PrintDefaults()
	os.Exit(status)
}

// isBinary guesses whether a file is binary by reading the first X bytes and seeing if there are any nulls.
// Assumes the file is at the beginning.
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

func main() {
	args := flag.Args()
	if len(args) < 3 {
		usage(1)
	}
	runApp(args)
}

func runApp(args []string) {
	findPattern := args[0]
	replacePattern := args[1]
	files := args[2:]

	find, err := regexp.Compile(findPattern)
	if err != nil {
		fmt.Println("Bad pattern for FIND:", err)
		os.Exit(1)
	}
	replace := []byte(replacePattern)

fileLoop:
	for _, filename := range files {
		if !isRegular(filename) {
			if verbose {
				fmt.Fprintf(os.Stderr, "Skipping %s (not a regular file).\n", filename)
			}
			continue
		}
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		defer file.Close()
		if isBinary(file) {
			if verbose {
				fmt.Fprintf(os.Stderr, "Skipping %s (binary file).\n", filename)
			}
			continue
		}
		stat, err := file.Stat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not stat file %s: %s\n", filename, err)
			continue
		}
		if stat.Mode().Perm()&0222 == 0 {
			fmt.Fprintf(os.Stderr, "Skipping write-protected file %s\n", filename)
			continue
		}

		var temp *os.File
		if !dryRun {
			temp, err = ioutil.TempFile(".", filename)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			temp.Chmod(stat.Mode())
			defer temp.Close()
		}

		matched := false
		reader := bufio.NewReader(file)
		line := 0
		done := false

		for !done {
			line++
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					done = true
					if len(line) == 0 {
						break
					}
				} else {
					fmt.Fprintln(os.Stderr, err)
					break
				}
			}

			indices := find.FindAllIndex(line, -1)
			if indices == nil {
				if !dryRun {
					_, err := temp.Write(line)
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
						continue fileLoop
					}
				}
				continue
			}
			if !matched {
				// Only print out the filename in blue if we're in verbose mode.
				if verbose {
					fmt.Fprintln(out, colorize(filename, ColorBlue))
				} else {
					fmt.Println(filename)
				}
				matched = true
			}
			if verbose {
				highlighted := highlight(line, ColorRed, indices)
				replacedAndHighlighted := subAndHighlight(line, find, replace, ColorGreen, indices)

				fmt.Fprint(out, colorize("- ", ColorRed))
				out.Write(highlighted)
				fmt.Fprint(out, colorize("+ ", ColorGreen))
				out.Write(replacedAndHighlighted)
			}
			if !dryRun {
				replaced := substitute(line, find, replace, indices)
				_, err := temp.Write(replaced)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue fileLoop
				}
			}
		}

		if !dryRun {
			// We'll .Close these twice, but that's fine.
			temp.Close()
			file.Close()
			if err := renameFile(temp.Name(), filename); err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
		}
	}
}
