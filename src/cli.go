package src

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Read markdown file from filepath
func readMarkdown(filepath string) ([]byte, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	// Refuse if it's not a markdown file
	if !strings.HasSuffix(filepath, ".md") {
		return nil, fmt.Errorf("file is not a markdown file")
	}
	return file, nil
}

// Extract URLs from markdown content
func findLinks(markdown []byte) []string {
	var links []string

	// Parse the markdown content
	reader := text.NewReader(markdown)
	parser := goldmark.DefaultParser()
	document := parser.Parse(reader)

	// Function to add link if it's an HTTP/S URL
	addLinkIfHTTP := func(destination []byte) {
		url := string(destination)
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			links = append(links, url)
		}
	}

	// Traverse the AST to find link and image nodes
	err := ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch n := node.(type) {
			case *ast.Link:
				addLinkIfHTTP(n.Destination)
			case *ast.Image:
				addLinkIfHTTP(n.Destination)
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		log.Fatal(err)
	}

	return links
}

// Check the state of a URL and save the result in a struct
type linkRecord struct {
	URL        string
	StatusCode int
	ErrMsg     string
}

// Check the state of a URL
func checkLink(url string, timeout time.Duration) linkRecord {
	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		// Check if the error is a timeout
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return linkRecord{
				URL:        url,
				StatusCode: 0,
				ErrMsg:     fmt.Sprintf("Request timed out after %s", timeout),
			}
		}
		return linkRecord{
			URL:        url,
			StatusCode: 0,
			ErrMsg:     err.Error(),
		}
	}
	defer resp.Body.Close()

	return linkRecord{URL: url, StatusCode: resp.StatusCode, ErrMsg: ""}
}

// Print a pretty header
func printHeader(w *tabwriter.Writer) {
	defer w.Flush()
	fmt.Fprintf(w, "\nLink patrol\n===========\n\n")
}

// Print the filepath
func printFilepath(w *tabwriter.Writer, filepath string) {
	defer w.Flush()
	fmt.Fprintf(w, "Filepath: %s\n\n", filepath)
}

// Print the URL states
func printLinkRecord(w *tabwriter.Writer, linkRecord linkRecord) {
	defer w.Flush()

	const tpl = `- URL        : {{.URL}}
  Status Code: {{if eq .StatusCode 0}}-{{else}}{{.StatusCode}}{{end}}
  Error      : {{if .ErrMsg}}{{.ErrMsg}}{{else}}-{{end}}

`
	t, err := template.New("linkRecord").Parse(tpl)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = t.Execute(w, linkRecord)
	if err != nil {
		log.Fatal(err)
		return
	}
}

// Check the state of a list of URLs
func checkLinks(w *tabwriter.Writer, urls []string, timeout time.Duration, errOk bool) {
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var hasError bool

	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			urlState := checkLink(url, timeout)

			// Print the URL state
			mutex.Lock()
			printLinkRecord(w, urlState)

			// Raise error if the URL state has an error status code
			if urlState.StatusCode >= 400 {
				hasError = true
			}

			mutex.Unlock()
			wg.Done()
		}(url)
	}
	wg.Wait()

	if hasError && !errOk {
		log.Fatal("Some URLs are invalid or unreachable")
	}
}

// Orchestrate the whole process
func orchestrate(w *tabwriter.Writer, filepath string, timeout time.Duration, errOk bool) {
	defer w.Flush()
	// Read markdown file from filepath
	markdown, err := readMarkdown(filepath)
	if err != nil {
		log.Fatal(err)
	}

	urls := findLinks(markdown)
	checkLinks(w, urls, timeout, errOk)
}

// CLI
func Cli(w *tabwriter.Writer, version string, exitFunc func(int)) {
	defer w.Flush()
	printHeader(w)

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
	}

	// Main Action
	app.Action = func(c *cli.Context) error {
		defer w.Flush()

		filepath := c.String("filepath")
		timeout := c.Duration("timeout")
		errOk := c.Bool("error-ok")

		if filepath == "" {
			// Show help if no filepath is provided
			_ = cli.ShowAppHelp(c)
			return fmt.Errorf("filepath is required")
		}

		// Proceed with orchestration as filepath is provided
		printFilepath(w, filepath)
		orchestrate(w, filepath, timeout, errOk) // Your orchestrate function
		return nil
	}

	// Handle execution
	err := app.Run(os.Args)
	if err != nil {
		exitFunc(2) // Your exitFunc function
	}
}
