<div align="center">
<pre align="center">
<h1 align="center">
;; link-patrol ;;
</h1>
<h4 align="center">
Detect dead links in markdown files
</h4>
</pre>
</div>

## Installation

-   On MacOS, brew install:

    ```sh
    brew tap rednafi/link-patrol https://github.com/rednafi/link-patrol \
       && brew install link-patrol
    ```

-   Or elsewhere, go install:

    ```sh
    go install github.com/rednafi/link-patrol/cmd/link-patrol
    ```

## Quickstart

### Usage

```sh
link-patrol -h
```

```txt
NAME:
   Link patrol - detect dead links in markdown files

USAGE:
   link-patrol [global options] command [command options]

VERSION:
   0.5

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --filepath value, -f value  path to the markdown file
   --timeout value, -t value   timeout for each HTTP request (default: 5s)
   --error-ok, -e              always exit with code 0 (default: false)
   --json, -j                  output as JSON (default: false)
   --help, -h                  show help
   --version, -v               print the version
```

### List URL status

Here's the content of a sample markdown file:

```md
This is an [embedded](https://example.com) URL.

This is a [reference style] URL.

This is a footnote[^1] URL.

[reference style]: https://reference.com
[^1]: https://gen.xyz/
```

Run the following command to list the URL status with a 2 second timeout for each request:

```sh
link-patrol -f examples/sample_1.md -t 2s
```

By default, it'll exit with a non-zero code if any of the URLs is invalid or unreachable.
Here's how the output looks:

```txt
Filepath: examples/sample_1.md

- Location   : https://reference.com
  Status Code: 403
  OK         : false
  Message    : Forbidden

- Location   : https://example.com
  Status Code: 200
  OK         : true
  Message    : OK

- Location   : https://gen.xyz/
  Status Code: -
  OK         : false
  Message    : Request timed out after 2s

2024/02/01 21:19:21 one or more URLs have error status codes
exit status 1
```

### Ignore errors

Set the `--error-ok / -e` flag to force the CLI to always exit with code 0:

```sh
link-patrol -f examples/sample_1.md -e
```

### Print as JSON

Use the `--json / -j` flag to format the output as JSON:

```sh
link-patrol -f examples/sample_2.md -t 5s --json | jq
```

```json
{
  "location": "https://referencestyle.com",
  "statusCode": 0,
  "ok": false,
  "message": "... no such host"
}
{
  "location": "https://example.com",
  "statusCode": 200,
  "ok": true,
  "message": "OK"
}
{
  "location": "https://example.com/image.jpg",
  "statusCode": 404,
  "ok": false,
  "message": "Not Found"
}
```

### Check multiple files

Do some shell-fu:

```sh
find examples -name '*.md' -exec link-patrol -f {} -t 4s -e \;
```
