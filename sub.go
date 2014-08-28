package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"

	flag "github.com/cespare/pflag"
)

type config struct {
	dry  bool
	verb bool

	find    *regexp.Regexp
	replace []byte
}

func (c *config) run(filename string) (err error) {
	if !isRegular(filename) {
		if c.verb {
			fmt.Fprintf(os.Stderr, "Skipping %s (not a regular file).\n", filename)
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
			fmt.Fprintf(os.Stderr, "Skipping %s (binary file).\n", filename)
		}
		return nil
	}
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	if stat.Mode().Perm()&0222 == 0 {
		if c.verb {
			fmt.Fprintf(os.Stderr, "Skipping write-protected file %s\n", filename)
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
	done := false
	reader := bufio.NewReader(f)

lineLoop:
	for !done {
		line, err := reader.ReadBytes('\n')
		switch err {
		case nil:
		case io.EOF:
			done = true
			if len(line) == 0 {
				break lineLoop
			}
		default:
			fmt.Fprintln(os.Stderr, err)
			break lineLoop
		}

		indices := c.find.FindAllIndex(line, -1)
		if indices == nil {
			if !c.dry {
				_, err := temp.Write(line)
				if err != nil {
					return err
				}
			}
			continue
		}
		if !matched {
			// Only print out the filename in blue if we're in verbose mode.
			if c.verb {
				fmt.Println(colorize(filename, ColorBlue))
			} else {
				fmt.Println(filename)
			}
			matched = true
		}
		if c.verb {
			highlighted := highlight(line, ColorRed, indices)
			replacedAndHighlighted := subAndHighlight(line, c.find, c.replace, ColorGreen, indices)

			fmt.Print(colorize("- ", ColorRed))
			os.Stdout.Write(highlighted)
			fmt.Print(colorize("+ ", ColorGreen))
			os.Stdout.Write(replacedAndHighlighted)
		}
		if !c.dry {
			replaced := substitute(line, c.find, c.replace, indices)
			_, err := temp.Write(replaced)
			if err != nil {
				return err
			}
		}
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

func usage(status int) {
	fmt.Printf(`Usage:
  %s [OPTIONS] <FIND> <REPLACE> <FILE1> <FILE2> ...
where OPTIONS are
`, os.Args[0])
	flag.PrintDefaults()
	fmt.Println("If no files are listed, sub reads filenames from standard input, one name per line.")
	os.Exit(status)
}

func main() {
	var conf config
	flag.BoolVarP(&conf.dry, "dry-run", "d", false, "Print out what would be changed without changing any files.")
	flag.BoolVarP(&conf.verb, "verbose", "v", false, "Print out detailed information about each match.")
	flag.Usage = func() { usage(0) }
	flag.Parse()

	files := make(chan string)
	args := flag.Args()
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

	for filename := range files {
		if err := conf.run(filename); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
