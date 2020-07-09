package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/ogier/pflag"
)

type config struct {
	dry  bool
	verb bool

	find    *regexp.Regexp
	replace []byte
	stdout  io.Writer
	stderr  io.Writer
}

func (c *config) run(filename string) (err error) {
	if !isRegular(filename) {
		if c.verb {
			fmt.Fprintf(c.stderr, "Skipping %s (not a regular file).\n", filename)
		}
		return nil
	}
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if isBinary(f) {
		if c.verb {
			fmt.Fprintf(c.stderr, "Skipping %s (binary file).\n", filename)
		}
		return nil
	}
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	if stat.Mode().Perm()&0222 == 0 {
		if c.verb {
			fmt.Fprintf(c.stderr, "Skipping write-protected file %s\n", filename)
		}
		return nil
	}

	var temp *os.File
	if !c.dry {
		temp, err = tempFile(filename, ".sub-tmp", stat.Mode())
		if err != nil {
			return err
		}
		defer func() {
			// Best-effort cleanup; if err == nil then temp is gone.
			temp.Close()
			os.Remove(temp.Name())
		}()
	}

	matched := false
	scanner := bufio.NewScanner(f)
	scanner.Split(scanLines)
	scanner.Buffer(make([]byte, 100e3), 10e6)
	for scanner.Scan() {
		line := scanner.Bytes()
		// Run the regex against the line text itself (not including
		// terminating marker).
		indices := c.find.FindAllIndex(trimCR(line), -1)
		if indices == nil {
			if !c.dry {
				if _, err := temp.Write(line); err != nil {
					return err
				}
			}
			continue
		}
		if !matched {
			// Only print out the filename in blue if we're in verbose mode.
			if c.verb {
				fmt.Fprintln(c.stdout, colorize(filename, ColorBlue))
			} else {
				fmt.Fprintln(c.stdout, filename)
			}
			matched = true
		}
		if c.verb {
			highlighted := highlight(line, ColorRed, indices)
			replacedAndHighlighted := subAndHighlight(line, c.find, c.replace, ColorGreen, indices)

			fmt.Fprint(c.stdout, colorize("- ", ColorRed))
			c.stdout.Write(highlighted)
			fmt.Fprint(c.stdout, colorize("+ ", ColorGreen))
			c.stdout.Write(replacedAndHighlighted)
		}
		if !c.dry {
			replaced := substitute(line, c.find, c.replace, indices)
			if _, err := temp.Write(replaced); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(c.stderr, err)
	}

	if !c.dry {
		if err := temp.Close(); err != nil {
			return err
		}
		if err := os.Rename(temp.Name(), filename); err != nil {
			return err
		}
	}
	return nil
}

func trimCR(line []byte) []byte {
	n := len(line)
	if n > 0 && line[n-1] == '\n' {
		if n > 1 && line[n-2] == '\r' {
			return line[:n-1]
		}
		return line[:n-1]
	}
	return line
}

// scanLines is like bufio.ScanLines except that it includes any end-of-line
// marker in the yielded tokens.
func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[:i+1], nil
	}
	if atEOF {
		// We have a final, non-terminated line.
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func usage(status int) {
	fmt.Printf(`Usage:
  %s [OPTIONS] <FIND> <REPLACE> <FILE1> <FILE2> ...
where OPTIONS are
`, os.Args[0])
	pflag.PrintDefaults()
	fmt.Println("If no files are listed, sub reads filenames from standard input, one name per line.")
	os.Exit(status)
}

func main() {
	var conf config
	pflag.BoolVarP(&conf.dry, "dry-run", "d", false, "Print out what would be changed without changing any files.")
	pflag.BoolVarP(&conf.verb, "verbose", "v", false, "Print out detailed information about each match.")
	pflag.Usage = func() { usage(0) }
	pflag.Parse()

	files := make(chan string)
	args := pflag.Args()
	switch {
	case len(args) < 2:
		usage(1)
	case len(args) == 2:
		// Take filenames from stdin, one per line.
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				files <- scanner.Text()
			}
			if err := scanner.Err(); err != nil {
				fmt.Println("Error reading from stdin:", err)
				os.Exit(1)
			}
			close(files)
		}()
	default:
		// Take filenames from the args after find and replace.
		go func() {
			for _, filename := range args[2:] {
				files <- filename
			}
			close(files)
		}()
	}
	findPat := args[0]
	replacePat := args[1]

	var err error
	conf.find, err = regexp.Compile(findPat)
	if err != nil {
		fmt.Println("Bad pattern for FIND:", err)
		os.Exit(1)
	}
	conf.replace = []byte(replacePat)
	conf.stdout = os.Stdout
	conf.stderr = os.Stderr

	for filename := range files {
		if err := conf.run(filename); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
