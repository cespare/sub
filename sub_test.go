package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestRunApp(t *testing.T) {
	source := "001.txt"
	contents := fmt.Sprintf("Hello\n%s, hello Golang, hello again\n", source)
    err := ioutil.WriteFile(source, []byte(contents), 0644)
	if err != nil {
		t.Errorf(`unexpected error: %v`, err)
	}

	args := []string {"hello", "Foo", source}
	runApp(args)

	res, err := ioutil.ReadFile(source)
	if err != nil {
		t.Errorf(`unexpected error: %v`, err)
	}
	actual := string(res)
	expected := fmt.Sprintf("Hello\n%s, Foo Golang, Foo again\n", source)
	if actual != expected {
		t.Errorf(`expected %s, got %s`, expected, actual)
	}

	err = os.Remove(source)
	if err != nil {
		t.Errorf(`unexpected error: %v`, err)
	}
}
