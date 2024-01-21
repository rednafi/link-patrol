package src

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

// readMarkdown reads a markdown file from a filepath
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

// linkType is a custom type to represent the type of a link
type linkType string

// Constants for LinkType
const (
	urlType      linkType = "url"
	filepathType linkType = "filepath"
)

// Check the state of a URL and save the result in a struct
type linkRecord struct {
	Type       linkType // literal url or filepath
	Location   string   // value of the url or filepath
	StatusCode int      // status code of the url
	Ok         bool     // true if the link is reachable
	ErrMsg     string   // error message if the link is unreachable
}

// newLinkRecord checks if the input is a valid HTTP/HTTPS URL or a properly formatted
// filepath and returns a linkRecord struct.
func newLinkRecord(link string) linkRecord {

	// Check for HTTP/HTTPS URL
	u, err := url.ParseRequestURI(link)
	if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return linkRecord{
			Type:       urlType,
			Location:   link,
			StatusCode: 0,
			Ok:         false,
			ErrMsg:     "",
		}
	}

	// Check for filepath
	currDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	link = filepath.Clean(link)
	link = strings.TrimRight(link, string(filepath.Separator))

	if !strings.HasSuffix(link, ".md") {
		link = link + ".md"
	}

	link = filepath.Join(currDir, link)

	if strings.HasPrefix(link, "/") ||
		strings.HasPrefix(link, "./") || strings.HasPrefix(link, "../") {
		return linkRecord{
			Type:       filepathType,
			Location:   link,
			StatusCode: 0,
			Ok:         false,
			ErrMsg:     "",
		}
	}

	// Check for Windows-style paths
	if filepath.IsAbs(link) {
		return linkRecord{
			Type:       "filepath",
			Location:   link,
			StatusCode: 0,
			Ok:         false,
			ErrMsg:     "",
		}
	}
	return linkRecord{}
}

// Extract URLs from markdown content
func findLinks(markdown []byte, skipRelative bool) ([]string, error) {
	var links []string

	// Parse the markdown content
	reader := text.NewReader(markdown)
	parser := goldmark.DefaultParser()
	document := parser.Parse(reader)

	addLink := func (destination []byte) {
		link := string(destination)
		lr := newLinkRecord(link)

		// Skip adding the link if it's a file path and we're skipping relative links
		if lr.Type == filepathType && skipRelative {
			return
		}

		// Add the link if it's an HTTP/S URL or a file path
		if lr.Type == urlType || lr.Type == filepathType {
			links = append(links, link)
			return
		}
	}

	// Traverse the AST to find link and image nodes
	err := ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch n := node.(type) {
			case *ast.Link:
				addLink(n.Destination)
			case *ast.Image:
				addLink(n.Destination)
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return links, err
	}

	return links, nil
}

// checkUrl checks the state of a URL
func checkUrl(lr linkRecord, timeout time.Duration) linkRecord {

	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(lr.Location)
	if err != nil {
		lr.Ok = false
		// Check if the error is a timeout
		if err, ok := err.(net.Error); ok && err.Timeout() {
			lr.ErrMsg = fmt.Sprintf("Request timed out after %s", timeout)
			return lr
		}
		lr.ErrMsg = err.Error()
		return lr
	}
	defer resp.Body.Close()

	// Set lr.Ok to false if the status code is an error code
	if lr.StatusCode >= 400 {
		lr.Ok = false
		lr.ErrMsg = "Unreachable URL"
	}

	lr.StatusCode = resp.StatusCode
	lr.Ok = true

	return lr
}

// checkFilepath checks the state of a filepath
func checkFilepath(lr linkRecord) linkRecord {
	if _, err := os.Stat(lr.Location); err == nil {
		lr.Ok = true
		return lr
	}
	lr.Ok = false
	lr.ErrMsg = fmt.Sprintf("Filepath %s does not exist", lr.Location)
	return lr
}

// Check the state of a URL or filepath
func checkLink(link string, timeout time.Duration) linkRecord {
	switch lr := newLinkRecord(link); lr.Type {
	case "url":
		return checkUrl(lr, timeout)
	case "filepath":
		return checkFilepath(lr)
	default:
		return linkRecord{}
	}
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
func printLinkRecord(w *tabwriter.Writer, lr linkRecord) {
	defer w.Flush()

	const tpl = `- Type        : {{.Type}}
  location    : {{.Location}}
  Status Code : {{if eq .StatusCode 0}}-{{else}}{{.StatusCode}}{{end}}
  Ok          : {{.Ok}}
  Error       : {{if .ErrMsg}}{{.ErrMsg}}{{else}}-{{end}}

`
	t, err := template.New("linkRecord").Parse(tpl)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = t.Execute(w, lr)
	if err != nil {
		log.Fatal(err)
		return
	}
}

// Check the state of a list of URLs
func checkLinks(
	w *tabwriter.Writer, links []string, timeout time.Duration, errOk bool,
) error {
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var hasError bool

	for _, url := range links {
		wg.Add(1)
		go func(url string) {
			lr := checkLink(url, timeout)

			// Print the URL state
			mutex.Lock()
			printLinkRecord(w, lr)

			// Raise error if the URL state has an error status code
			if lr.StatusCode >= 400 {
				hasError = true
			}
			mutex.Unlock()
			wg.Done()
		}(url)
	}
	wg.Wait()

	if hasError && !errOk {
		return fmt.Errorf("one or more links are unreachable")
	}
	return nil
}

// Orchestrate the whole process
func orchestrate(w *tabwriter.Writer, filepath string, timeout time.Duration, skipRelative bool, errOk bool) {
	defer w.Flush()

	printFilepath(w, filepath)

	// Read markdown file from filepath
	markdown, err := readMarkdown(filepath)
	if err != nil {
		log.Fatal(err.Error())
	}

	urls, err := findLinks(markdown, skipRelative)
	if err != nil {
		log.Fatal(err)
	}

	err = checkLinks(w, urls, timeout, errOk)
	if err != nil {
		log.Fatal(err)
	}
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
			Value:   30 * time.Second,
			Usage:   "timeout for each HTTP request",
		},
		&cli.BoolFlag{
			Name:    "skip-relative",
			Aliases: []string{"s"},
			Value:   false,
			Usage:   "skip relative paths",
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
		skipRelative := c.Bool("skip-relative")
		errOk := c.Bool("error-ok")

		if filepath == "" {
			// Show help if no filepath is provided
			_ = cli.ShowAppHelp(c)
			return fmt.Errorf("filepath is required")
		}

		// Proceed with orchestration as filepath is provided
		orchestrate(w, filepath, timeout, skipRelative, errOk)
		return nil
	}

	// Handle execution
	err := app.Run(os.Args)
	if err != nil {
		exitFunc(2)
	}
}
