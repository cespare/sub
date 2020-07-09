package main

import (
	"io/ioutil"
	"os"
	"regexp"
	"testing"
)

func TestBasic(t *testing.T) {
	for _, testCase := range []struct {
		before  string
		after   string
		find    string
		replace string
	}{
		{"foo\n", "bar\n", "foo", "bar"},
		{"foo bar foo", "bar bar bar", "foo", "bar"},
		{"foo bar foo\n", "bar\n", "foo.*", "bar"},
		{"foobar\n", "boobar\n", `f(o+)`, "b$1"},
		{"foo\n", "bar\n", "foo", "bar"},
		{"xyz xyz\n", "xyz abc\n", "xyz$", "abc"},
		{"abc\r\n123\nabc\nxyz", "def\r\n123\ndef\nxyz", "abc", "def"},
	} {
		temp := writeTemp(t, testCase.before)
		conf := testConf(testCase.find, testCase.replace)
		if err := conf.run(temp); err != nil {
			t.Error(err)
		}
		b, err := ioutil.ReadFile(temp)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != testCase.after {
			t.Errorf("got %q; want %q (find=%q, replace=%q)\n", b, testCase.after, testCase.find, testCase.replace)
		}
	}
}

func testTemp() (*os.File, error) { return ioutil.TempFile("", "sub-test-") }

func writeTemp(t *testing.T, s string) (filename string) {
	f, err := testTemp()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	if _, err := f.WriteString(s); err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func testConf(find, replace string) config {
	return config{
		find:    regexp.MustCompile(find),
		replace: []byte(replace),
		stdout:  ioutil.Discard,
		stderr:  ioutil.Discard,
	}
}
