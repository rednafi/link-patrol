package src

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadMarkdown tests the readFile function
func TestReadMarkdown(t *testing.T) {
	t.Parallel()
	tmpDir := os.TempDir()
	filepath := tmpDir + "/testfile.md"
	expected := []byte("test content")

	// Create a temporary file with test content
	file, err := os.Create(filepath)
	require.NoError(t, err, "failed to create test file")
	defer func() {
		// Close the file
		file.Close()

		// Remove the temporary file
		err := os.Remove(filepath)
		require.NoError(t, err, "failed to remove test file")
	}()

	// Write the test content to the file
	_, err = file.Write(expected)
	require.NoError(t, err, "failed to write to test file")

	// Call the function under test
	actual, err := readMarkdown(filepath)
	require.NoError(t, err, "unexpected error")

	// Compare the actual result with the expected result
	assert.Equal(t, string(expected), string(actual), "unexpected result")
}

// TestReadMarkdown_NonExistentFile tests the readFile function with a non-existent file
func TestReadMarkdown_NonExistentFile(t *testing.T) {
	t.Parallel()
	tmpDir := os.TempDir()
	filepath := tmpDir + "/non-existent-file.md"
	_, err := readMarkdown(filepath)

	require.Error(t, err, "non existent file should return an error")
}

// TestReadMarkdown_NonMarkdownFile tests the readFile function with a non-markdown file
func TestReadMarkdown_NonMarkdownFile(t *testing.T) {
	t.Parallel()
	tmpDir := os.TempDir()
	filepath := tmpDir + "/testfile.txt"

	// Create a temporary file
	file, err := os.Create(filepath)
	require.NoError(t, err, "failed to create test file")
	defer func() {
		// Close the file
		file.Close()

		// Remove the temporary file
		err := os.Remove(filepath)
		require.NoError(t, err, "failed to remove test file")
	}()

	// Call the function under test
	_, err = readMarkdown(filepath)

	// Check if an error was returned
	require.Error(t, err, "Expected an error for non-markdown file")
}

func TestFindLinks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		markdown []byte
		want     []string
	}{
		{
			name: "Basic Functionality",
			markdown: []byte(
				"[link](http://example.com) ![image](https://example.com/image.jpg)",
			),
			want: []string{
				"http://example.com",
				"https://example.com/image.jpg",
			},
		},
		{
			name:     "No Links",
			markdown: []byte("No links here."),
			want:     []string{},
		},
		{
			name: "Mixed Content",
			markdown: []byte(
				"# Heading\n\n[link](http://example.com)\n\n" +
					"Text here\n\n![image](https://example.com/image.jpg)",
			),
			want: []string{
				"http://example.com",
				"https://example.com/image.jpg",
			},
		},
		{
			name: "Non-HTTP/S Links",
			markdown: []byte(
				`[http](http://example.com) [https](https://example.com)
				[ftp](ftp://example.com) [mailto](mailto:example@example.com)`,
			),
			want: []string{"http://example.com", "https://example.com"},
		},
		{
			name: "Nested Elements",
			markdown: []byte(
				"> [link](http://example.com)\n\n* ![image](https://example.com/image.jpg)",
			),
			want: []string{
				"http://example.com",
				"https://example.com/image.jpg",
			},
		},
		{
			name:     "Invalid Markdown Syntax",
			markdown: []byte("[Invalid link](http://example.com"),
			want:     []string{},
		},
		{
			name: "Large Input",
			markdown: []byte(
				"[link1](http://example.com) ... [linkN](http://exampleN.com)",
			),
			want: []string{"http://example.com", "http://exampleN.com"},
		},
		{
			name: "Special Characters in URLs",
			markdown: []byte(
				"[link](http://example.com?query=value&param=value)",
			),
			want: []string{"http://example.com?query=value&param=value"},
		},

		{
			name:     "Unicode and Encoding",
			markdown: []byte("[链接](http://例子.公司)"),
			want:     []string{"http://例子.公司"},
		},
		{
			name:     "Nil",
			markdown: nil,
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := findLinks(tt.markdown)

			// Treat nil slices as equivalent to empty slices
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}

			assert.Equal(
				t,
				tt.want,
				got,
				"findLinks() did not return expected result",
			)
		})
	}
}

// TestCheckLink_Success tests the checkUrl function with a successful HTTP request
func TestCheckLink_Success(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer ts.Close()

	lr := checkLink(ts.URL, 1*time.Second)

	assert.Equal(t, http.StatusOK, lr.StatusCode, "Status code should be 200")
	assert.Equal(t, "OK", lr.Message, "Error message should be 'OK'")
}

// TestCheckLink_ClientError tests the checkUrl function with a client error
func TestCheckLink_ClientError(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}),
	)
	defer ts.Close()

	lr := checkLink(ts.URL, 1*time.Second)

	assert.Equal(
		t,
		http.StatusNotFound,
		lr.StatusCode,
		"Status code should be 404",
	)
	assert.Equal(t, "Not Found", lr.Message)
}

// TestCheckLink_ServerError tests the checkUrl function with a server error
func TestCheckLink_ServerError(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer ts.Close()

	lr := checkLink(ts.URL, 1*time.Second)

	assert.Equal(
		t,
		http.StatusInternalServerError,
		lr.StatusCode,
		"Status code should be 500",
	)
	assert.Equal(t, "Internal Server Error", lr.Message)
}

// TestCheckLink_ConnectionError tests the checkUrl function with a connection error
func TestCheckLink_ConnectionError(t *testing.T) {
	t.Parallel()
	lr := checkLink("http://localhost:12345", 1*time.Second)

	assert.Equal(t, 0, lr.StatusCode, "Status code should be 0")
	assert.Contains(
		t,
		lr.Message,
		"connection refused",
		"Error message should contain 'connection refused'",
	)
}

// TestCheckLink_InvalidLink tests the checkUrl function with an invalid URL format
func TestCheckLink_InvalidLink(t *testing.T) {
	t.Parallel()
	lr := checkLink(":%", 1*time.Second)

	assert.Equal(t, 0, lr.StatusCode, "Status code should be 0")
	assert.Equal(t, "parse \":%\": missing protocol scheme", lr.Message)
}

// Test for printFilepath function
func TestPrintFilepath(t *testing.T) {
	t.Parallel()
	expectedOutput := "Filepath: testfile.md\n\n"

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	defer w.Flush()

	printFilepath(w, "testfile.md", false)
	assert.Equal(
		t,
		expectedOutput,
		buf.String(),
		"printFilepath() did not return expected result",
	)
}

// Test for printLinkRecordTab function
func TestPrintLinkRecordTab(t *testing.T) {
	t.Parallel()
	linkRecord := linkRecord{"http://example.com", 200, true, "OK"}
	expectedOutput := "- Location   : http://example.com\n" +
		"  Status Code: 200\n" +
		"  OK         : true\n" +
		"  Message    : OK\n\n"

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	defer w.Flush()

	_ = printLinkRecordTab(w, linkRecord)
	assert.Equal(
		t,
		expectedOutput,
		buf.String(),
		"printLinkRecordTab() did not return expected result",
	)
}

// Test for printLinkRecordJSON function
func TestPrintLinkRecordJSON(t *testing.T) {
	t.Parallel()
	linkRecord := linkRecord{"http://example.com", 200, true, "OK"}
	expectedOutput := "{\n" +
		"  \"location\": \"http://example.com\",\n" +
		"  \"statusCode\": 200,\n" +
		"  \"ok\": true,\n" +
		"  \"message\": \"OK\"\n" +
		"}\n"

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	defer w.Flush()

	_ = printLinkRecordJSON(w, linkRecord)
	assert.Equal(
		t,
		expectedOutput,
		buf.String(),
		"printLinkRecordJSON() did not return expected result",
	)
}

// Test for printLinkRecord function
func TestPrintLinkRecord(t *testing.T) {
	t.Parallel()
	linkRecord := linkRecord{"http://example.com", 200, true, "OK"}
	expectedOutput := "- Location   : http://example.com\n" +
		"  Status Code: 200\n" +
		"  OK         : true\n" +
		"  Message    : OK\n\n"

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	defer w.Flush()

	_ = printLinkRecord(w, linkRecord, false)
	assert.Equal(
		t,
		expectedOutput,
		buf.String(),
		"printLinkRecord() did not return expected result",
	)

	// Test with JSON output
	expectedOutput = "{\n" +
		"  \"location\": \"http://example.com\",\n" +
		"  \"statusCode\": 200,\n" +
		"  \"ok\": true,\n" +
		"  \"message\": \"OK\"\n" +
		"}\n"

	var buf2 bytes.Buffer
	w2 := tabwriter.NewWriter(&buf2, 0, 0, 1, ' ', 0)
	defer w2.Flush()

	_ = printLinkRecord(w2, linkRecord, true)
	assert.Equal(
		t,
		expectedOutput,
		buf2.String(),
		"printLinkRecord() did not return expected result",
	)
}

func TestCheckLinks(t *testing.T) {
	t.Parallel()
	// Create a test tabwriter.Writer
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Create a test server that always returns a specific status code
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer ts.Close()

	// Create a list of test URLs
	urls := []string{ts.URL + "/ok", ts.URL + "/invalid-url"}

	// Set the timeout and error flag for testing
	timeout := time.Second
	errOK := false
	asJSON := false

	// Call the checkLinks function
	_ = checkLinks(w, urls, timeout, errOK, asJSON)

	output := buf.String()

	// Verify the output
	expectedOutput1 := "- Location   : " + ts.URL + "/ok\n" +
		"  Status Code: 200\n" +
		"  OK         : true\n" +
		"  Message    : OK\n\n" +
		"- Location   : " + ts.URL + "/invalid-url\n" +
		"  Status Code: 200\n" +
		"  OK         : true\n" +
		"  Message    : OK\n\n"
	expectedOutput2 := "- Location   : " + ts.URL + "/invalid-url\n" +
		"  Status Code: 200\n" +
		"  OK         : true\n" +
		"  Message    : OK\n\n" +
		"- Location   : " + ts.URL + "/ok\n" +
		"  Status Code: 200\n" +
		"  OK         : true\n" +
		"  Message    : OK\n\n"

	assert.Contains(
		t,
		[]string{expectedOutput1, expectedOutput2},
		output,
		"checkLinks() did not return expected result",
	)
}

func TestCheckLinks_RaisesError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		description     string
		serverResponses map[string]int // URL path to response code
		ignoreErrors    bool
		expectError     bool
	}{
		{
			description: "should error with some bad links",
			serverResponses: map[string]int{
				"/good": http.StatusOK,
				"/bad":  http.StatusInternalServerError,
			},
			ignoreErrors: false,
			expectError:  true,
		},
		{
			description: "should not error with all good links",
			serverResponses: map[string]int{
				"/good1": http.StatusOK,
				"/good2": http.StatusOK,
			},
			ignoreErrors: false,
			expectError:  false,
		},
		{
			description: "should not error with bad links when ignoring errors",
			serverResponses: map[string]int{
				"/good": http.StatusOK,
				"/bad":  http.StatusInternalServerError,
			},
			ignoreErrors: true,
			expectError:  false,
		},
	}

	createTestServer := func(responses map[string]int) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if code, ok := responses[r.URL.Path]; ok {
					w.WriteHeader(code)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}))
	}

	createURLs := func(server *httptest.Server, paths map[string]int) []string {
		var urls []string
		for path := range paths {
			urls = append(urls, server.URL+path)
		}
		return urls
	}

	runCheckLinks := func(
		w *tabwriter.Writer, urls []string, timeout time.Duration, ignoreErrors bool,
	) bool {
		err := checkLinks(w, urls, timeout, ignoreErrors, false)
		return err != nil
	}

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			server := createTestServer(test.serverResponses)
			defer server.Close()

			urls := createURLs(server, test.serverResponses)

			w := tabwriter.NewWriter(log.Writer(), 0, 0, 0, ' ', 0)
			errOccurred := runCheckLinks(
				w,
				urls,
				5*time.Second,
				test.ignoreErrors,
			)

			assert.Equal(t, test.expectError, errOccurred)
		})
	}
}

// Benchmark for checkUrls
func BenchmarkCheckUrls(b *testing.B) {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		}),
	)
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
		_ = checkLinks(w, testUrls, 1*time.Second, true, false)
	}
}
