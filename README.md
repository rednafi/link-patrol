## Link patrol

## Usage

* Inspect the help section:

   ```sh
   link-petrol -h
   ```

   ```
   NAME:
      Link patrol - detect dead links in markdown files

   USAGE:
      link-patrol [global options] command [command options]

   VERSION:
      sentinel

   AUTHOR:
      Redowan Delowar

   COMMANDS:
      help, h  Shows a list of commands or help for one command

   GLOBAL OPTIONS:
      --filepath value, -f value  path to the markdown file
      --timeout value, -t value   timeout for each HTTP request (default: 5s)
      --error-ok, -e              always exit with code 0 (default: false)
      --help, -h                  show help
      --version, -v               print the version
   ```

* Find the dead urls in a sample markdown file:

   Here's sample file that we'll use (examples/sample_1.md):

   ```md
   This is an [embedded](https://example.com) URL.

   This is a [reference style] URL.

   This is a footnote[^1] URL.

   [reference style]: https://reference.com
   [^1]: https://gen.xyz/
   ```

   Run the following command with a 2 second timeout for each request:

   ```sh
   link-patrol -f examples/sample_1.md -t 2s
   ```

   This returns:

   ```txt
   Link patrol
   ===========

   Filepath: examples/sample_1.md

   - URL        : https://reference.com
   Status Code: 403
   Error      : -

   - URL        : https://gen.xyz/
   Status Code: 200
   Error      : -

   - URL        : https://example.com
   Status Code: 200
   Error      : -

   2024/01/20 03:21:49 Some URLs are invalid or unreachable
   exit status 1
   ```

* Suppress errors:

   ```sh
   go run cmd/link-patrol/main.go -f examples/sample_1.md --error-ok
   ```

   This will force the CLI to exit with code 0.
