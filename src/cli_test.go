package src

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"text/tabwriter"
)

// TestReadFile tests the readFile function
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

// TestReadFile_NonExistentFile tests the readFile function with a non-existent file
func TestReadFile_NonExistentFile(t *testing.T) {
	tmpDir := os.TempDir()
	filepath := tmpDir + "/non-existent-file.md"
	_, err := readFile(filepath)

	if err == nil {
		t.Errorf("Expected an error, got no error")
	}
}

// TestReadFile_NonMarkdownFile tests the readFile function with a non-markdown file
func TestReadFile_NonMarkdownFile(t *testing.T) {
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
	_, err = readFile(filepath)

	if err == nil {
		t.Errorf("Expected an error, got no error")
	}
}

// TestCheckUrl_Success tests the checkUrl function with a successful HTTP request
func TestCheckUrl_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	state := checkUrl(ts.URL)

	if state.statusCode != http.StatusOK || state.errMsg != "" {
		t.Errorf("Expected status 200 with no error, got status %d with error '%s'", state.statusCode, state.errMsg)
	}
}

// TestCheckUrl_ClientError tests the checkUrl function with a client error
func TestCheckUrl_ClientError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	state := checkUrl(ts.URL)

	if state.statusCode != http.StatusNotFound || state.errMsg != "" {
		t.Errorf("Expected status 404 with no error, got status %d with error '%s'", state.statusCode, state.errMsg)
	}
}

// TestCheckUrl_ServerError tests the checkUrl function with a server error
func TestCheckUrl_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	state := checkUrl(ts.URL)

	if state.statusCode != http.StatusInternalServerError || state.errMsg != "" {
		t.Errorf("Expected status 500 with no error, got status %d with error '%s'", state.statusCode, state.errMsg)
	}
}

// TestCheckUrl_ConnectionError tests the checkUrl function with a connection error
func TestCheckUrl_ConnectionError(t *testing.T) {
	state := checkUrl("http://localhost:12345")

	if state.errMsg == "" {
		t.Errorf("Expected a connection error, got no error")
	}
}

// TestCheckUrl_InvalidUrl tests the checkUrl function with an invalid URL format
func TestCheckUrl_InvalidUrl(t *testing.T) {
	state := checkUrl(":%")

	if state.errMsg == "" {
		t.Errorf("Expected an invalid URL error, got no error")
	}
}

// TestCheckUrls tests the checkUrls function with various URL scenarios
func TestCheckUrls(t *testing.T) {
	// Create a test server that responds based on URL path
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

	// Define test URLs
	testUrls := []string{
		ts.URL + "/ok",
		ts.URL + "/notfound",
		ts.URL + "/error",
		"http://localhost:12345", // Connection error
		":%",                     // Invalid URL
	}

	// Call checkUrls
	var wg sync.WaitGroup
	urlStates := checkUrls(&wg, testUrls)

	// Verify the results
	if len(urlStates) != len(testUrls) {
		t.Errorf("Expected %d url states, got %d", len(testUrls), len(urlStates))
	}

	// Checking each URL state
	urlToStatus := make(map[string]int)

	for _, state := range urlStates {
		urlToStatus[state.url] = state.statusCode
	}

	expectedUrlToStatus := map[string]int{
		testUrls[0]: http.StatusOK,
		testUrls[1]: http.StatusNotFound,
		testUrls[2]: http.StatusInternalServerError,
		testUrls[3]: 0,
		testUrls[4]: 0,
	}

	// Assert urlToStatus == expectedUrlToStatus
	for url, status := range urlToStatus {
		if status != expectedUrlToStatus[url] {
			t.Errorf("Expected status %d for url %s, got %d", expectedUrlToStatus[url], url, status)
		}
	}
}

// Test for printHeader function
func TestPrintHeader(t *testing.T) {
	expectedOutput := "\n\nLink patrol\n===========\n\n"

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	printHeader(w)
	w.Flush()

	if buf.String() != expectedOutput {
		t.Errorf("printHeader() = %q, want %q", buf.String(), expectedOutput)
	}
}

// Test for printFilepath function
func TestPrintFilepath(t *testing.T) {
	expectedOutput := "/tmp/testfile.md\n----------------\n\n"

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	printFilepath(w, "/tmp/testfile.md")
	w.Flush()

	if buf.String() != expectedOutput {
		t.Errorf("printFilepath() = %q, want %q", buf.String(), expectedOutput)
	}
}

// Test for printUrlState function
func TestPrintUrlState(t *testing.T) {
	urlStates := []urlState{
		{"http://example.com", 200, "OK"},
		{"http://testsite.com", 404, "Not Found"},
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	printUrlState(w, urlStates)
	w.Flush()

	expectedOutput := "URL Status Code Error\n--- ----------- -----\n\nhttp://example.com  200 OK\nhttp://testsite.com 404 Not Found\n"
	actualOutput := buf.String()

	if actualOutput != expectedOutput {
		t.Errorf("printUrlState() = %q, want %q", actualOutput, expectedOutput)
	}
}
