package src

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// readMarkdown reads a markdown file from the provided filepath.
// Returns the file contents as a byte slice and an error.
func readMarkdown(filepath string) ([]byte, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Refuse if it's not a markdown file.
	if !strings.HasSuffix(filepath, ".md") {
		return nil, errors.New("file is not a markdown file")
	}

	return file, nil
}

// findLinks parses markdown content and returns a slice of found HTTP/S URLs.
func findLinks(markdown []byte) ([]string, error) {
	var links []string

	// Parse the markdown.
	reader := text.NewReader(markdown)
	parser := goldmark.DefaultParser()
	document := parser.Parse(reader)

	// Add link to result if it's an HTTP/S URL.
	addLinkIfHTTP := func(destination []byte) {
		url := string(destination)
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			links = append(links, url)
		}
	}

	// Walk AST to find link and image nodes.
	if err := ast.Walk(
		document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering {
				switch n := node.(type) {
				case *ast.Link:
					addLinkIfHTTP(n.Destination)
				case *ast.Image:
					addLinkIfHTTP(n.Destination)
				}
			}
			return ast.WalkContinue, nil
		}); err != nil {
		return nil, fmt.Errorf("failed to traverse markdown AST: %w", err)
	}

	return links, nil
}

// linkRecord stores the result of checking a URL.
type linkRecord struct {
	Location   string `json:"location"`
	StatusCode int    `json:"statusCode"`
	OK         bool   `json:"ok"`
	Message    string `json:"message"`
	Attempt    int    `json:"attempt"`
}

func checkLink(
	url string,
	timeout time.Duration,
	maxRetries int,
	startBackoff time.Duration,
	maxBackoff time.Duration,
) linkRecord {
	client := &http.Client{
		Timeout: timeout,
	}

	var resp *http.Response
	var err error

	// This should be synchronous, retrying concurrently doesn't make sense.
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err = client.Get(url)
		if err == nil && resp.StatusCode < 400 {
			defer resp.Body.Close()
			return linkRecord{
				Location:   url,
				StatusCode: resp.StatusCode,
				OK:         true,
				Message:    http.StatusText(resp.StatusCode),
				Attempt:    attempt,
			}
		}

		if attempt < maxRetries {
			backoff := startBackoff * 2

			// Apply jitter by adding a random amount of milliseconds
			jitter := time.Duration(rand.Intn(100)) * time.Millisecond
			actualBackoff := backoff + jitter

			// Cap the backoff time to a maximum value
			if actualBackoff > maxBackoff {
				actualBackoff = maxBackoff
			}

			time.Sleep(actualBackoff)
		}
	}

	statusText := "Unknown error"
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			statusText = "Request timed out after " + timeout.String()
		} else {
			statusText = err.Error()
		}
	} else if resp != nil {
		statusText = http.StatusText(resp.StatusCode)
	}

	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}

	return linkRecord{
		Location:   url,
		StatusCode: statusCode,
		OK:         false,
		Message:    statusText,
		Attempt:    maxRetries,
	}
}

// printFilepath prints the filepath unless outputting JSON.
func printFilepath(w io.Writer, filepath string, asJSON bool) {
	if !asJSON {
		fmt.Fprintf(w, "Filepath: %s\n\n", filepath)
	}
}

// printLinkRecordJSON encodes a linkRecord to JSON.
func printLinkRecordJSON(w io.Writer, lr linkRecord) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	// Convert to camelCase.
	// ...
	return enc.Encode(lr)
}

// printLinkRecordTab prints a linkRecord in tabular format.
func printLinkRecordTab(w io.Writer, lr linkRecord) error {
	tpl := `- Location   : {{.Location}}
  Status Code: {{if eq .StatusCode 0}}-{{else}}{{.StatusCode}}{{end}}
  OK         : {{.OK}}
  Message    : {{if .Message}}{{.Message}}{{else}}-{{end}}
  Attempt    : {{.Attempt}}

`
	t, err := template.New("record").Parse(tpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	if err := t.Execute(w, lr); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	return nil
}

// printLinkRecord prints a linkRecord in JSON or tabular format.
func printLinkRecord(w io.Writer, lr linkRecord, asJSON bool) error {
	if asJSON {
		return printLinkRecordJSON(w, lr)
	}
	return printLinkRecordTab(w, lr)
}

// checkLinks concurrently checks a list of URLs.
// Prints results and returns first error encountered, if any.
func checkLinks(
	w io.Writer,
	urls []string,
	timeout time.Duration,
	maxRetries int,
	startBackoff time.Duration,
	maxBackoff time.Duration,
	errOK bool,
	asJSON bool,
) error {
	var (
		wg    sync.WaitGroup
		mutex sync.Mutex
		err   error
	)

	for _, url := range urls {
		wg.Add(1)

		go func(url string) {
			defer wg.Done()

			result := checkLink(url, timeout, maxRetries, startBackoff, maxBackoff)

			mutex.Lock()
			defer mutex.Unlock()

			if printErr := printLinkRecord(w, result, asJSON); printErr != nil {
				err = printErr
				return
			}

			if result.StatusCode >= 400 && err == nil {
				err = errors.New("one or more URLs have error status codes")
			}
		}(url)
	}

	wg.Wait()

	if err != nil && !errOK {
		return err
	}

	return nil
}

// orchestrate coordinates the full link checking process.
func orchestrate(
	w io.Writer,
	filepath string,
	timeout time.Duration,
	maxRetries int,
	startBackoff time.Duration,
	maxBackoff time.Duration,
	errOK bool,
	asJSON bool,
	exitFunc func(int),
) {
	printFilepath(w, filepath, asJSON)

	markdown, err := readMarkdown(filepath)
	if err != nil {
		fmt.Fprintln(w, err)
		exitFunc(1)
	}

	links, err := findLinks(markdown)
	if err != nil {
		fmt.Fprintln(w, err)
		exitFunc(1)
	}

	if err := checkLinks(
		w, links, timeout, maxRetries, startBackoff, maxBackoff, errOK, asJSON,
	); err != nil {
		fmt.Fprintln(w, err)
		exitFunc(1)
	}
}

// CLI
func CLI(w io.Writer, version string, exitFunc func(int)) {
	app := cli.NewApp()
	app.Name = "Link patrol"
	app.Usage = "detect dead links in markdown files"
	app.Version = version
	app.UsageText = "link-patrol [global options] command [command options]"
	app.HelpName = "Link patrol"
	app.Suggest = true
	app.EnableBashCompletion = true

	// Custom Writer
	app.Writer = w
	app.ErrWriter = w

	// Global Flags
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "filepath",
			Aliases: []string{"f"},
			Usage:   "path to the markdown file",
		},
		&cli.DurationFlag{
			Name:    "timeout",
			Aliases: []string{"t"},
			Value:   5 * time.Second,
			Usage:   "timeout for each HTTP request",
		},
		&cli.BoolFlag{
			Name:    "error-ok",
			Aliases: []string{"e"},
			Value:   false,
			Usage:   "always exit with code 0",
		},
		&cli.BoolFlag{
			Name:    "json",
			Aliases: []string{"j"},
			Value:   false,
			Usage:   "output as JSON",
		},
		&cli.IntFlag{
			Name:  "max-retries",
			Value: 1,
			Usage: "maximum number of retries for each URL",
		},
		&cli.DurationFlag{
			Name:  "start-backoff",
			Value: 1 * time.Second,
			Usage: "initial backoff duration for retries",
		},
		&cli.DurationFlag{
			Name:  "max-backoff",
			Value: 4 * time.Second,
			Usage: "maximum backoff duration for retries",
		},
	}

	// Main Action
	app.Action = func(c *cli.Context) error {
		filepath := c.String("filepath")
		timeout := c.Duration("timeout")
		maxRetries := c.Int("max-retries")
		startBackoff := c.Duration("start-backoff")
		maxBackoff := c.Duration("max-backoff")
		errOK := c.Bool("error-ok")
		asJSON := c.Bool("json")

		if filepath == "" {
			// Show help if no filepath is provided
			_ = cli.ShowAppHelp(c)
			return fmt.Errorf("filepath is required")
		}

		// startBackoff should be at least 1ms
		if startBackoff < time.Millisecond {
			return fmt.Errorf("start-backoff should be at least 1ms")
		}

		// maxBackoff must be greater than or equal to startBackoff
		if maxBackoff < startBackoff {
			return fmt.Errorf(
				"max-backoff should be greater than or equal to start-backoff",
			)
		}

		// Proceed with orchestration as filepath is provided
		orchestrate(
			w,
			filepath,
			timeout,
			maxRetries,
			startBackoff,
			maxBackoff,
			errOK,
			asJSON,
			exitFunc,
		)
		return nil
	}

	// Handle execution
	err := app.Run(os.Args)
	if err != nil {
		exitFunc(2)
	}
}
