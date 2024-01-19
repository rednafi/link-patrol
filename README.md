## Link patrol

## Usage


```sh
link-petrol -h
```

```
NAME:
   Link patrol - Test the URLs in your markdown files

USAGE:
   Link patrol [global options] command [command options]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --filepath value, -f value  path to the markdown file
   --timeout value, -t value   timeout for each HTTP request (default: 5s)
   --help, -h                  show help
```

Here's a sample markdown file (examples/sample_1.md):

```md
This is an [embedded](https://example.com) URL.

This is a [reference style] URL.

This is a footnote[^1] URL.

[reference style]: https://reference.com
[^1]: https://gen.xyz/
```

Check the URLs with the following command with a 2 second timeout:

```sh
link-patrol -f examples/sample_1.md -t 2s
```

This returns:

```txt
Link patrol
===========

Filepath: examples/sample_1.md

- URL        : https://example.com
  Status Code: 200
  Error      : -

- URL        : https://gen.xyz/
  Status Code: 200
  Error      : -

- URL        : https://reference.com
  Status Code: 403
  Error      : -

2024/01/19 05:31:17 Some URLs are invalid or unreachable
exit status 1
```
