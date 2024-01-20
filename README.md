<div align="center">
<img src="./art/logo.webp" width="800" height="400" alt="Image Description">
</div>

## Installation

* On MacOS, brew install:

   ```sh
   brew tap rednafi/link-patrol https://github.com/rednafi/link-patrol \
      && brew install link-patrol
   ```

* Or elsewhere, go install:

   ```sh
   go install github.com/rednafi/link-patrol/cmd/link-patrol
   ```

## Quickstart

### Usage

```sh
link-patrol -h
```

```txt
Link patrol
===========

NAME:
   Link patrol - detect dead links in markdown files

USAGE:
   link-patrol [global options] command [command options]

VERSION:
   sentinel

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --filepath value, -f value  path to the markdown file
   --timeout value, -t value   timeout for each HTTP request (default: 5s)
   --error-ok, -e              always exit with code 0 (default: false)
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

Run the following command to list thr URL statuses with a 2 second timeout for each request:

```sh
link-patrol -f examples/sample_1.md -t 2s
```

By default it'll exit with a non-zero code if any of URLs is invalid or unreachable. Here's
how the output looks:

```txt
Link patrol
===========

Filepath: examples/sample_1.md

- URL        : https://reference.com
  Status Code: 403
  Error      : -

- URL        : https://example.com
  Status Code: 200
  Error      : -

- URL        : https://gen.xyz/
  Status Code: 200
  Error      : -

2024/01/20 03:41:55 Some URLs are invalid or unreachable
exit status 1
```

### Ignore errors

Set the `-e / --error-ok` flag to force the CLI to always exit with code 0.

```sh
go run cmd/link-patrol/main.go -f examples/sample_1.md --error-ok
```

### Check multiple files

Do some shell-foo:

```sh
find examples -name '*.md' -exec link-patrol -f {} -t 4s -e \;
```
