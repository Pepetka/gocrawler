# gocrawler

A concurrent HTTP and HTTPS web crawler written in Go. Use it as a
command-line tool or as a library.

## Features

- Crawls one link depth at a time, processing pages in parallel.
- Follows links and redirects within the origins of the seed URLs.
- Deduplicates page URLs after removing fragments.
- Limits crawl depth, page count, time per page, and response body size.
- Reports HTTP status, bytes read, redirects, and page errors, with a partial
  report if the crawl is canceled.

## Installation

Requires Go 1.27.4 or later.

```sh
go install github.com/pepetka/gocrawler/cmd/gocrawler@latest
```

### Build from source

```sh
git clone https://github.com/pepetka/gocrawler.git
cd gocrawler
go build -o bin/gocrawler ./cmd/gocrawler
```

After building from source, use `./bin/gocrawler` in the examples below.

## Usage

```text
gocrawler [flags] [seed-url ...]
```

```sh
gocrawler -max-depth=1 -max-pages=10 https://example.com/
gocrawler -help
```

Put flags before the seed URLs. Each seed must be an absolute HTTP or HTTPS
URL. Seed URLs define the allowed origins by scheme, hostname, and effective
port. The crawler extracts links from successful `text/html` responses.
If an HTML response declares a charset, it must be UTF-8. Each redirect chain
can make up to ten HTTP requests, including the initial request.

| Flag              | Default           | Description                                                                                 |
| ----------------- | ----------------- | ------------------------------------------------------------------------------------------- |
| `-concurrency`    | `5`               | Maximum number of pages processed in parallel.                                              |
| `-timeout`        | `5s`              | Time limit per page, including redirects, body reads, and link extraction.                  |
| `-max-depth`      | `2`               | Maximum link depth. Use `0` to crawl only the seed URLs.                                    |
| `-max-pages`      | `100`             | Maximum number of distinct page URLs, including seeds. A redirect chain counts as one page. |
| `-max-body-bytes` | `2097152` (2 MiB) | Maximum response body size in bytes. The limit must be between 1 byte and 1 GiB.            |

### Output and exit codes

The CLI writes the crawl summary to standard output, along with a page table,
redirects, and page errors when present. It reports diagnostics on standard
error. Passwords in displayed URLs are masked.

| Exit code | Meaning                                                                                |
| --------- | -------------------------------------------------------------------------------------- |
| `0`       | Success, including a crawl stopped by a configured limit that produced no page errors. |
| `1`       | A page failed, the crawl was canceled, or another runtime error occurred.              |
| `2`       | The flags or seed URLs are invalid.                                                    |

## Library usage

```sh
go get github.com/pepetka/gocrawler
```

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/pepetka/gocrawler"
)

func main() {
	seed, err := gocrawler.ParseSeedURL("https://example.com/")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	report, err := gocrawler.Crawl(ctx, []*url.URL{seed}, gocrawler.Options{
		Concurrency:  4,
		PageTimeout:  5 * time.Second,
		MaxDepth:     1,
		MaxPages:     10,
		MaxBodyBytes: 2 * gocrawler.MiB,
	})
	fmt.Printf("Succeeded: %d of %d (%s)\n", report.Succeeded, report.Started, report.StopReason)
	if err != nil {
		log.Fatal(err)
	}
}
```

`Crawl` validates the options and seeds before making requests. Invalid input
returns an error matching `ErrInvalidInput` through `errors.Is`. If the crawl
context is canceled or its deadline expires, `Crawl` returns the partial report
and wraps `ctx.Err()`. A per-page timeout is a page failure recorded in
`PageResult.Err` and counted in `Report.Failed`.

## License

[MIT](LICENSE)
