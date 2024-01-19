package src

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Read markdown file from filepath
func readFile(filepath string) ([]byte, error) {
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
func extractUrls(markdown []byte) []string {
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
type UrlState struct {
	Url        string
	StatusCode int
	ErrMsg     string
}

// Check the state of a URL
func checkUrl(url string, timeout time.Duration) UrlState {
	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(url)

	if err != nil {
		// Check if the error is a timeout
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return UrlState{
				Url:        url,
				StatusCode: 0,
				ErrMsg:     fmt.Sprintf("Request timed out after %s", timeout),
			}
		}
		return UrlState{
			Url:        url,
			StatusCode: 0,
			ErrMsg:     err.Error(),
		}
	}
	defer resp.Body.Close()

	return UrlState{Url: url, StatusCode: resp.StatusCode, ErrMsg: ""}
}


// Check the state of a list of URLs
func checkUrls(wg *sync.WaitGroup, urls []string, timeout time.Duration) []UrlState {
	var urlStates []UrlState
	var mutex sync.Mutex

	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			urlState := checkUrl(url, timeout)

			mutex.Lock()
			urlStates = append(urlStates, urlState)
			mutex.Unlock()
		}(url)
	}

	wg.Wait()
	return urlStates
}

// Print a pretty header
func printHeader(w *tabwriter.Writer) {
	defer w.Flush()
	fmt.Fprintf(w, "\nLink patrol\n===========\n\n")
}

// Print the URL states
func printUrlState(w *tabwriter.Writer, urlStates []UrlState) {
	defer w.Flush()

	const tpl = `- URL        : {{.Url}}
  Status Code: {{if eq .StatusCode 0}}-{{else}}{{.StatusCode}}{{end}}
  Error      : {{if .ErrMsg}}{{.ErrMsg}}{{else}}-{{end}}

`
	t, err := template.New("UrlState").Parse(tpl)
	if err != nil {
		log.Fatal(err)
		return
	}

	for _, urlState := range urlStates {
		err := t.Execute(w, urlState)
		if err != nil {
			log.Fatal(err)
			return
		}
	}
}

func raiseErrorIfUrlStateHasErrorStatus(urlStates []UrlState) {
	statusCodeMap := make(map[int]struct{})
	for _, urlState := range urlStates {
		statusCodeMap[urlState.StatusCode] = struct{}{}
	}

	for code := 400; code <= 599; code++ {
		if _, exists := statusCodeMap[code]; exists {
			log.Fatal("Some URLs are invalid or unreachable")
			break
		}
	}
}

// Orchestrate the whole process
func orchestrate(w *tabwriter.Writer, filepath *string, timeout time.Duration) {
	defer w.Flush()
	// Read markdown file from filepath
	markdown, err := readFile(*filepath)
	if err != nil {
		log.Fatal(err)
	}

	urls := extractUrls(markdown)
	wg := &sync.WaitGroup{}
	urlStates := checkUrls(wg, urls, timeout)

	printUrlState(w, urlStates)
	raiseErrorIfUrlStateHasErrorStatus(urlStates)
}

// CLI
func Cli(w *tabwriter.Writer, version string, exitFunc func(int)) {
    defer w.Flush()

    // Set up command line flags
    fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
    fs.SetOutput(w)

    // Define CLI options with long and short flags
    var filepath string
    fs.StringVar(&filepath, "filepath", "", "Path to markdown file")
    fs.StringVar(&filepath, "f", "", "Path to markdown file (short)")

    var timeout time.Duration
    fs.DurationVar(&timeout, "timeout", 5*time.Second, "Timeout for HTTP requests")
    fs.DurationVar(&timeout, "t", 5*time.Second, "Timeout for HTTP requests (short)")

    var help bool
    fs.BoolVar(&help, "help", false, "Print usage")
    fs.BoolVar(&help, "h", false, "Print usage (short)")

	printHeader(w)


    // Parse flags and handle errors
    if err := fs.Parse(os.Args[1:]); err != nil {
        exitFunc(2)
    }

    // Check for no arguments or help flag
    if len(os.Args) < 2 || help {
        fs.Usage()
        return
    }

    // Handle filepath option
    if filepath != "" {
        orchestrate(w, &filepath, timeout)
        return
    }
}
