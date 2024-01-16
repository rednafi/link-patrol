package src

import (
	//"flag"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func readFile(filepath string) ([]byte, error) {
	// Read markdown file from filepath
	markdown, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return markdown, nil
}

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
	ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
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

	return links
}

type urlState struct {
	url       string
	statusCode int
	errMsg     string
}

func checkUrl(url string) urlState {
	resp, err := http.Get(url)

	if err != nil {
		return urlState{url: url, statusCode: 0, errMsg: err.Error()}
	}

	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		return urlState{url: url, statusCode: resp.StatusCode, errMsg: err.Error()}
	}

	return urlState{url: url, statusCode: resp.StatusCode, errMsg: ""}
}

func checkUrls(wg *sync.WaitGroup, urls []string) []urlState {
	var urlStates []urlState

	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			urlStates = append(urlStates, checkUrl(url))
			wg.Done()
		}(url)
	}
	wg.Wait()
	return urlStates
}

func printHeader(w *tabwriter.Writer) {
	fmt.Fprintf(w, "Link patrol\n")
	fmt.Fprintf(w, "===========\n\n")
}

func printUrlState(w *tabwriter.Writer, urlStates []urlState) {
	fmt.Fprintf(w, "URL\tStatus Code\tError\n")
	fmt.Fprintf(w, "---\t-----------\t-----\n\n")

	for _, urlState := range urlStates {
		fmt.Fprintf(w, "%s\t%d\t%s\n", urlState.url, urlState.statusCode, urlState.errMsg)
	}
}

func Cli(w *tabwriter.Writer, version string, exitFunc func(int)) {
	// Flush the writer at the end of the function
	defer w.Flush()

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine = fs

	// Set the default output to the passed tabwriter
	fs.SetOutput(w)

	// Define the flags
	filepath := flag.String("filepath", "", "Path to markdown file")
	flag.StringVar(filepath, "f", "", "Path to markdown file")

	help := flag.Bool("help", false, "Print usage")
	flag.BoolVar(help, "h", false, "Print usage")

	flag.Usage = func() {
		fmt.Fprintf(w, "Usage of %s:\n", os.Args[0])
		fmt.Fprint(w, `  -f, --filepath [filepath]
		Path to markdown file
  -h, --help
		Print usage`)
	}

	printHeader(w)
	flagUsageOld := flag.Usage
	fs.Usage = func() {
		flagUsageOld()
		w.Flush()
	}

	err := fs.Parse(os.Args[1:])
	if err != nil {
		exitFunc(2)
	}

	if len(os.Args) < 2 || *help {
		flag.Usage()
		return
	}

	if *filepath != "" {
		// Read markdown file from filepath
		markdown, err := readFile(*filepath)
		if err != nil {
			log.Fatal(err)
		}

		urls := extractUrls(markdown)
		wg := &sync.WaitGroup{}
		urlStates := checkUrls(wg, urls)

		printUrlState(w, urlStates)
	}
}
