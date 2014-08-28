# sub

Sub is a simple tool for doing find/replace across files.

Sub is a tool that modifies your files. Use it at your own risk. (In particular, commit or back up changes
before you have sub go to town on your data.)

Sub (probably) doesn't work on Windows.

## Installation

You'll need Go installed. Then:

    $ go get github.com/cespare/sub

## Usage

`sub -h` for help. This prints out:

```
Usage:
    ./sub [OPTIONS] <FIND> <REPLACE> <FILE1> <FILE2> ...
where OPTIONS are
  -d, --dry-run=false: Print out what would be changed without changing any files.
  -v, --verbose=false: Print out detailed information about each match.
```

I usually use `-dv` the first time so that I see what sub is going to do without having it make any changes.

## Examples

In order to give sub a list of files, I generally make use of my shell's globbing capabilities. For instance,
I often use `*.go` to indicate all `.go` files in the current directory. I also make use of ZSH's recursive
globbing; so `**/*.go` to indicate all `.go` files in all subdirectories. If you're a bash user, you can get
the same feature via `globstar` (see `man bash`).

```
# Replace instances of 'foo' with 'bar' in all .txt files.
sub foo bar *.txt

# Replace instances of Foobar, case insensitive, with xxx in all .c files.
sub '(?i)(Foobar)' xxx *.c

# Surround all numbers in parentheses in all .txt files, recursively.
sub '\d+' '($0)' **/*.txt

# Replace sell -> buy, seller -> buyer, selling -> buying, etc.
sub 'sell(\S*)' 'buy$1' *.txt
```

## Screenshot

![screenshot](http://i.imgur.com/0ZOSUlo.png)

## Notes

See the documentation for the regular expression syntax used by the `FIND` pattern here:

http://golang.org/pkg/regexp/syntax/

See documentation for the expansion syntax used in the `REPLACE` pattern here:

http://golang.org/pkg/regexp/#Regexp.Expand
