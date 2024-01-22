package src

import (
	"bytes"
	"fmt"

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

func TestNewLinkRecord(t *testing.T) {
	tests := []struct {
		name     string
		location string
		wantType linkType
		wantOk   bool
		wantErr  string
	}{
		{
			name:     "Valid HTTP URL",
			location: "http://example.com",
			wantType: urlType,
			wantOk:   false,
			wantErr:  "",
		},
		{
			name:     "Valid HTTPS URL",
			location: "https://example.com",
			wantType: urlType,
			wantOk:   false,
			wantErr:  "",
		},
		{
			name:     "Invalid URL",
			location: "htt://example.com",
			wantType: invalidType,
			wantOk:   false,
			wantErr:  "",
		},
		{
			name:     "Valid Relative Filepath",
			location: "./test.md",
			wantType: filepathType,
			wantOk:   false,
			wantErr:  "",
		},
		{
			name:     "Valid Absolute Filepath",
			location: "/tmp/test.md",
			wantType: filepathType,
			wantOk:   false,
			wantErr:  "",
		},
		{
			name:     "Invalid Filepath",
			location: "invalidpath/test",
			wantType: filepathType,
			wantOk:   false,
			wantErr:  "",
		},
		{
			name:     "Windows Style Absolute Path",
			location: "C:\\test.md",
			wantType: invalidType,
			wantOk:   false,
			wantErr:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newLinkRecord(tt.location)
			if got.ErrMsg != tt.wantErr {
				t.Errorf("newLinkRecord() error = %v, wantErr %v", got.ErrMsg, tt.wantErr)
			}
			if got.Type != tt.wantType {
				t.Errorf("newLinkRecord() Type = %v, wantType %v %v", got.Type, tt.wantType, tt.location)
			}
			if got.Ok != tt.wantOk {
				t.Errorf("newLinkRecord() Ok = %v, wantOk %v", got.Ok, tt.wantOk)
			}
		})
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

	if state.StatusCode != http.StatusNotFound && state.ErrMsg != "Unreachable URL" {
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

	if state.StatusCode != http.StatusInternalServerError && state.ErrMsg != "Unreachable URL" {
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
	lr := checkLink(":%", 1*time.Second)

	if lr.ErrMsg == "" {
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
	t.Parallel()
	// Create a test tabwriter.Writer
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)

	// Create a test server returning a status code
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// List of test URLs
	urls := []string{ts.URL + "/ok", ts.URL + "/invalid-url"}

	// Set timeout and error flag for testing
	timeout := time.Second
	errOk := false

	// Call checkLinks function
	_ = checkLinks(w, urls, timeout, errOk)

	// Flush tabwriter.Writer for output
	w.Flush()
	output := buf.String()

	// Verify the output
	expOutput1 := fmt.Sprintf("- Type        : url\n  location    : %s/invalid-url\n"+
		"  Status Code : 200\n  Ok          : true\n  Error       : -\n\n"+
		"- Type        : url\n  location    : %s/ok\n  Status Code : 200\n"+
		"  Ok          : true\n  Error       : -\n\n", ts.URL, ts.URL)

	expOutput2 := fmt.Sprintf("- Type        : url\n  location    : %s/ok\n"+
		"  Status Code : 200\n  Ok          : true\n  Error       : -\n\n"+
		"- Type        : url\n  location    : %s/invalid-url\n  Status Code : 200\n"+
		"  Ok          : true\n  Error       : -\n\n", ts.URL, ts.URL)

	if output != expOutput1 && output != expOutput2 {
		t.Errorf("checkLinks() = %q, want %q or %q",
			output, expOutput1, expOutput2)
	}
}

// Test for printLinkRecord function
func TestPrintLinkRecord(t *testing.T) {
	t.Parallel()

	// Create a slice of linkRecords
	linkRecords := []linkRecord{
		{"url", "http://example.com", 200, true, ""},
		{"url", "http://testsite.com", 404, false, "Not Found"},
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)

	// Process each linkRecord
	for _, lr := range linkRecords {
		printLinkRecord(w, lr)
	}
	w.Flush()

	// Define expected output
	expOutput := "- Type        : url\n  location    : http://example.com\n" +
		"  Status Code : 200\n  Ok          : true\n  Error       : -\n\n" +
		"- Type        : url\n  location    : http://testsite.com\n" +
		"  Status Code : 404\n  Ok          : false\n  Error       : Not Found\n\n"
	actualOutput := buf.String()

	// Assert output correctness
	if actualOutput != expOutput {
		t.Errorf("printLinkRecord() = %q, want %q", actualOutput, expOutput)
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
		_ = checkLinks(w, testUrls, 1*time.Second, true)
	}
}
