package src

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"text/tabwriter"
	"time"
)

// TestReadMarkdown tests the readFile function
func TestReadMarkdown(t *testing.T) {
	t.Parallel()
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
	actual, err := readMarkdown(filepath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Compare the actual result with the expected result
	if string(actual) != string(expected) {
		t.Errorf("unexpected result, got: %s, want: %s", string(actual), string(expected))
	}
}

// TestReadMarkdown_NonExistentFile tests the readFile function with a non-existent file
func TestReadMarkdown_NonExistentFile(t *testing.T) {
	t.Parallel()
	tmpDir := os.TempDir()
	filepath := tmpDir + "/non-existent-file.md"
	_, err := readMarkdown(filepath)

	if err == nil {
		t.Errorf("Expected an error, got no error")
	}
}

// TestReadMarkdown_NonMarkdownFile tests the readFile function with a non-markdown file
func TestReadMarkdown_NonMarkdownFile(t *testing.T) {
	t.Parallel()
	tmpDir := os.TempDir()
	filepath := tmpDir + "/testfile.txt"

	// Create a temporary file
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

	// Call the function under test
	_, err = readMarkdown(filepath)

	if err == nil {
		t.Errorf("Expected an error, got no error")
	}
}

// TestCheckLink_Success tests the checkUrl function with a successful HTTP request
func TestCheckLink_Success(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	state := checkLink(ts.URL, 1*time.Second)

	if state.StatusCode != http.StatusOK || state.ErrMsg != "" {
		t.Errorf(
			"Expected status 200 with no error, got status %d with error '%s'",
			state.StatusCode, state.ErrMsg)
	}
}

// TestCheckLink_ClientError tests the checkUrl function with a client error
func TestCheckLink_ClientError(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	state := checkLink(ts.URL, 1*time.Second)

	if state.StatusCode != http.StatusNotFound || state.ErrMsg != "" {
		t.Errorf(
			"Expected status 404 with no error, got status %d with error '%s'",
			state.StatusCode, state.ErrMsg,
		)
	}
}

// TestCheckLink_ServerError tests the checkUrl function with a server error
func TestCheckLink_ServerError(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	state := checkLink(ts.URL, 1*time.Second)

	if state.StatusCode != http.StatusInternalServerError || state.ErrMsg != "" {
		t.Errorf(
			"Expected status 500 with no error, got status %d with error '%s'",
			state.StatusCode, state.ErrMsg,
		)
	}
}

// TestCheckLink_ConnectionError tests the checkUrl function with a connection error
func TestCheckLink_ConnectionError(t *testing.T) {
	t.Parallel()
	state := checkLink("http://localhost:12345", 1*time.Second)

	if state.ErrMsg == "" {
		t.Errorf("Expected a connection error, got no error")
	}
}

// TestCheckLink_InvalidLink tests the checkUrl function with an invalid URL format
func TestCheckLink_InvalidLink(t *testing.T) {
	t.Parallel()
	state := checkLink(":%", 1*time.Second)

	if state.ErrMsg == "" {
		t.Errorf("Expected an invalid URL error, got no error")
	}
}

// Test for printHeader function
func TestPrintHeader(t *testing.T) {
	t.Parallel()
	expectedOutput := "\nLink patrol\n===========\n\n"

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	printHeader(w)
	w.Flush()

	if buf.String() != expectedOutput {
		t.Errorf("printHeader() = %q, want %q", buf.String(), expectedOutput)
	}
}

func TestCheckLinks(t *testing.T) {
	// Create a test tabwriter.Writer
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)

	// Create a test server that always returns a specific status code
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Create a list of test URLs
	urls := []string{ts.URL + "/ok", ts.URL + "/invalid-url"}

	// Set the timeout and error flag for testing
	timeout := time.Second
	errOk := false

	// Call the checkLinks function
	checkLinks(w, urls, timeout, errOk)

	// Flush the tabwriter.Writer to get the output
	w.Flush()
	output := buf.String()

	// Verify the output
	expectedOutput1 := "- URL        : " + ts.URL + "/ok\n" +
		"  Status Code: 200\n" +
		"  Error      : -\n\n" +
		"- URL        : " + ts.URL + "/invalid-url\n" +
		"  Status Code: 200\n" +
		"  Error      : -\n\n"
	expectedOutput2 := "- URL        : " + ts.URL + "/invalid-url\n" +
		"  Status Code: 200\n" +
		"  Error      : -\n\n" +
		"- URL        : " + ts.URL + "/ok\n" +
		"  Status Code: 200\n" +
		"  Error      : -\n\n"

	if output != expectedOutput1 && output != expectedOutput2 {
		t.Errorf(
			"checkLinks() = %q, want %q or %q",
			output, expectedOutput1, expectedOutput2,
		)
	}
}

// Test for printUrlState function
func TestPrintLinkRecord(t *testing.T) {
	t.Parallel()
	linkRecords := []linkRecord{
		{"http://example.com", 200, "OK"},
		{"http://testsite.com", 404, "Not Found"},
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)

	for _, linkRecord := range linkRecords {
		printLinkRecord(w, linkRecord)
	}
	w.Flush()

	expectedOutput := "- URL        : http://example.com\n" +
		"  Status Code: 200\n" +
		"  Error      : OK\n\n" +
		"- URL        : http://testsite.com\n" +
		"  Status Code: 404\n" +
		"  Error      : Not Found\n\n"
	actualOutput := buf.String()

	if actualOutput != expectedOutput {
		t.Errorf("printUrlState() = %q, want %q", actualOutput, expectedOutput)
	}
}

// Benchmark for checkUrls
func BenchmarkCheckUrls(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
		case "/notfound":
			w.WriteHeader(http.StatusNotFound)
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	testUrls := []string{
		ts.URL + "/ok",
		ts.URL + "/notfound",
		ts.URL + "/error",
		"http://localhost:12345", // Connection error
		":%",                     // Invalid URL
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)

	for i := 0; i < b.N; i++ {
		checkLinks(w, testUrls, 1*time.Second, true)
	}
}
