package src

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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
	ErrMsg     string `json:"errMsg"`
}

// checkLink makes an HTTP request to url using the provided timeout.
// Returns a linkRecord with the result.
func checkLink(url string, timeout time.Duration) linkRecord {
	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		// Check for timeout error.
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return linkRecord{
				Location:   url,
				StatusCode: 0,
				ErrMsg:     fmt.Sprintf("Request timed out after %s", timeout),
			}
		}

		return linkRecord{
			Location:   url,
			StatusCode: 0,
			ErrMsg:     err.Error(),
		}
	}

	defer resp.Body.Close()

	return linkRecord{
		Location:   url,
		StatusCode: resp.StatusCode,
		ErrMsg:     "",
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
  Error      : {{if .ErrMsg}}{{.ErrMsg}}{{else}}-{{end}}

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

			result := checkLink(url, timeout)

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
	errOK bool,
	asJSON bool,
) {
	printFilepath(w, filepath, asJSON)

	markdown, err := readMarkdown(filepath)
	if err != nil {
		log.Fatal(err)
	}

	links, err := findLinks(markdown)
	if err != nil {
		log.Fatal(err)
	}

	if err := checkLinks(w, links, timeout, errOK, asJSON); err != nil {
		log.Fatal(err)
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
	}

	// Main Action
	app.Action = func(c *cli.Context) error {
		filepath := c.String("filepath")
		timeout := c.Duration("timeout")
		errOK := c.Bool("error-ok")
		asJSON := c.Bool("json")

		if filepath == "" {
			// Show help if no filepath is provided
			_ = cli.ShowAppHelp(c)
			return fmt.Errorf("filepath is required")
		}

		// Proceed with orchestration as filepath is provided
		orchestrate(w, filepath, timeout, errOK, asJSON)
		return nil
	}

	// Handle execution
	err := app.Run(os.Args)
	if err != nil {
		exitFunc(2)
	}
}
