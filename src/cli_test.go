package src

import (
	"fmt"
	"os"
	"testing"
)

func TestReadFile(t *testing.T) {
	tmpDir := os.TempDir()
	filepath := tmpDir + "/testfile.md"
	expected := []byte("test content")

	// Create a temporary file with test content
	file, err := os.Create(filepath)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() {
		// Close the file
		file.Close()

		// Remove the temporary file
		err := os.Remove(filepath)
		if err != nil {
			t.Fatalf("failed to remove test file: %v", err)
		}
	}()

	// Write the test content to the file
	_, err = file.Write(expected)
	if err != nil {
		t.Fatalf("failed to write to test file: %v", err)
	}

	// Call the function under test
	actual, err := readFile(filepath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Compare the actual result with the expected result
	if string(actual) != string(expected) {
		t.Errorf("unexpected result, got: %s, want: %s", string(actual), string(expected))
	}
}

func TestExtractUrls(t *testing.T) {
	markdown := []byte(
		"\nThis is an [embedded](https://example.com) URL.\n\nThis is a [reference style] URL.\n\nThis is a footnote[^1] URL.\n\n\n[reference style]: https://reference.com\n[^1]: https://gen.xyz/\n")

	expected := []string{
		"https://example.com",
		"https://reference.com",
		"https://gen.xyz/",
	}

	actual := extractUrls(markdown)

	fmt.Println(actual)
	if len(actual) != len(expected) {
		t.Errorf("unexpected number of URLs, got: %d, want: %d", len(actual), len(expected))
	}

	for i := range expected {
		if actual[i] != expected[i] {
			t.Errorf("unexpected URL at index %d, got: %s, want: %s", i, actual[i], expected[i])
		}
	}
}
