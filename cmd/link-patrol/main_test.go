package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/rednafi/link-patrol/src"
)

func TestCLIHelpCommand(t *testing.T) {
	t.Parallel()
	// Mock os.Exit to prevent the test runner from exiting
	mockExit := func(code int) {}

	// Capture the output by using a bytes.Buffer
	var out bytes.Buffer
	w := tabwriter.NewWriter(&out, 0, 4, 4, ' ', 0)

	// Simulate executing the help command
	args := os.Args[0:1] // Keep the program name only
	args = append(args, "--help")
	os.Args = args

	src.CLI(w, "0.1.0-test", mockExit)

	// Flush to make sure all output is written
	w.Flush()

	// Verify the output contains the usage
	output := out.String()
	if !strings.Contains(output, "USAGE:") {
		t.Errorf("Expected usage in output, got %s", output)
	}
}

func TestCLIVersionCommand(t *testing.T) {
	t.Parallel()
	// Mock os.Exit to prevent the test runner from exiting
	mockExit := func(code int) {}

	// Capture the output by using a bytes.Buffer
	var out bytes.Buffer
	w := tabwriter.NewWriter(&out, 0, 4, 4, ' ', 0)

	// Simulate executing the version command
	args := os.Args[0:1] // Keep the program name only
	args = append(args, "--version")
	os.Args = args

	src.CLI(w, "0.1.0-test", mockExit)

	// Flush to make sure all output is written
	w.Flush()

	// Verify the output contains the version
	output := out.String()
	if !strings.Contains(output, "0.1.0-test") {
		t.Errorf("Expected version '0.1.0-test' in output, got %s", output)
	}
}

func TestCLIInvalidCommand(t *testing.T) {
	t.Parallel()
	// Mock os.Exit to prevent the test runner from exiting
	mockExit := func(code int) {}

	// Capture the output by using a bytes.Buffer
	var out bytes.Buffer
	w := tabwriter.NewWriter(&out, 0, 4, 4, ' ', 0)

	// Simulate executing an invalid command
	args := os.Args[0:1] // Keep the program name only
	args = append(args, "invalid-command")
	os.Args = args

	src.CLI(w, "0.1.0-test", mockExit)

	// Flush to make sure all output is written
	w.Flush()

	// Verify that the CLI prints the usage
	output := out.String()
	if !strings.Contains(output, "USAGE:") {
		t.Errorf("Expected error message in output, got %s", output)
	}
}
