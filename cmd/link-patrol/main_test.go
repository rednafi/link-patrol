package main

import (
	"bytes"
	"os"
	"testing"
	"text/tabwriter"

	"github.com/rednafi/link-patrol/src"
	"github.com/stretchr/testify/assert"
)

func MakeMockMdFile() string {
	// Create sample markdown file in system's temp directory with prefix "sample_1"
	// The actual filename will have a unique suffix to ensure uniqueness
	file, err := os.CreateTemp("", "sample_1*.md")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Write some content to the file
	_, err = file.WriteString(`This is an [embedded](https://doesnt.exist) URL.

This is a [reference style] URL.

This is a footnote[^1] URL.

[reference style]: https://not.either

[^1]: https://wot.wot/
`)
	if err != nil {
		panic(err)
	}

	// Return the full path of the temporary file
	return file.Name()
}

func TestCLIHelpCommand(t *testing.T) {
	// Mock os.Exit to prevent the test runner from exiting
	mockExit := func(code int) {}

	// Capture the output by using a bytes.Buffer
	var out bytes.Buffer
	w := tabwriter.NewWriter(&out, 0, 4, 4, ' ', 0)
	defer w.Flush()

	// Simulate executing the help command
	args := os.Args[0:1] // Keep the program name only
	args = append(args, "--help")
	os.Args = args

	src.CLI(w, "0.1.0-test", mockExit)

	// Verify the output contains the usage
	output := out.String()

	assert.Contains(t, output, "USAGE:")
}

func TestCLIVersionCommand(t *testing.T) {
	// Mock os.Exit to prevent the test runner from exiting
	mockExit := func(code int) {}

	// Capture the output by using a bytes.Buffer
	var out bytes.Buffer
	w := tabwriter.NewWriter(&out, 0, 4, 4, ' ', 0)
	defer w.Flush()

	// Simulate executing the version command
	args := os.Args[0:1] // Keep the program name only
	args = append(args, "--version")
	os.Args = args

	src.CLI(w, "0.1.0-test", mockExit)

	// Verify the output contains the version
	output := out.String()
	assert.Contains(t, output, "0.1.0-test")
}

func TestCLIInvalidCommand(t *testing.T) {
	// Mock os.Exit to prevent the test runner from exiting
	mockExit := func(code int) {}

	// Capture the output by using a bytes.Buffer
	var out bytes.Buffer
	w := tabwriter.NewWriter(&out, 0, 4, 4, ' ', 0)
	defer w.Flush()

	// Simulate executing an invalid command
	args := os.Args[0:1] // Keep the program name only
	args = append(args, "invalid-command")
	os.Args = args

	src.CLI(w, "0.1.0-test", mockExit)

	// Verify that the CLI prints the usage
	output := out.String()
	assert.Contains(t, output, "USAGE:")
}

func TestCLIPrintTab(t *testing.T) {
	// Mock os.Exit to prevent the test runner from exiting
	mockExit := func(code int) {}

	// Create a sample markdown file
	filePath := MakeMockMdFile()

	// Capture the output by using a bytes.Buffer
	var out bytes.Buffer
	w := tabwriter.NewWriter(&out, 0, 4, 4, ' ', 0)
	defer w.Flush()

	// Simulate executing -f and -e flags
	args := os.Args[0:1] // Keep the program name only
	args = append(args, "-f", filePath, "-e", "-t", "5s")
	os.Args = args

	src.CLI(w, "0.1.0-test", mockExit)

	// Verify that the CLI prints the usage
	output := out.String()

	assert.Contains(t, output, "OK         : false")
	assert.Contains(
		t,
		output,
		"https://doesnt.exist",
	)
	assert.Contains(t, output, "https://wot.wot/")
	assert.Contains(
		t,
		output,
		"https://wot.wot/",
	)
	assert.Contains(t, output, "Location   : https://not.either")
}

func TestCLIPrintJSON(t *testing.T) {
	// Mock os.Exit to prevent the test runner from exiting
	mockExit := func(code int) {}

	// Create a sample markdown file
	filePath := MakeMockMdFile()

	// Capture the output by using a bytes.Buffer
	var out bytes.Buffer
	w := tabwriter.NewWriter(&out, 0, 4, 4, ' ', 0)
	defer w.Flush()

	// Simulate executing -f and -j flags
	args := os.Args[0:1] // Keep the program name only
	args = append(args, "-f", filePath, "-j", "-t", "5s")
	os.Args = args

	src.CLI(w, "0.1.0-test", mockExit)

	// Verify that the CLI prints the usage
	output := out.String()
	assert.Contains(t, output, "{\n  \"location\": \"https://not.either\"")
	assert.Contains(t, output, `"statusCode": 0`)
	assert.Contains(t, output, `"ok": false`)
}
