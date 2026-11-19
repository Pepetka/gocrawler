package main

import (
	"flag"
	"fmt"
	"net/url"
	"time"

	"github.com/pepetka/gocrawler"
)

func parseArgs() (gocrawler.Options, []*url.URL, error) {
	opts := gocrawler.Options{}
	flag.Usage = func() {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), `Usage: gocrawler [flags] [seed-url ...]

Crawl HTTP and HTTPS URLs within the origins of the seed URLs (scheme, host, and port).

Flags:`)
		flag.PrintDefaults()
	}
	flag.IntVar(&opts.Concurrency, "concurrency", 5, "maximum number of pages processed in parallel")
	flag.DurationVar(&opts.PageTimeout, "timeout", 5*time.Second, "per-page time limit for fetching the response (including redirects) and extracting links (e.g. 500ms, 5s)")
	flag.IntVar(&opts.MaxDepth, "max-depth", 2, "maximum link depth to crawl (seed URLs are at depth 0)")
	flag.IntVar(&opts.MaxPages, "max-pages", 100, "maximum number of distinct URLs to crawl, including seeds (a redirect chain counts as one page)")
	var maxBodyBytes uint64
	flag.Uint64Var(&maxBodyBytes, "max-body-bytes", uint64(2*gocrawler.MiB), "maximum response body size in bytes (1 byte to 1 GiB)")
	flag.Parse()

	opts.MaxBodyBytes = gocrawler.ByteSize(maxBodyBytes)

	if err := opts.Validate(); err != nil {
		return opts, nil, fmt.Errorf("%w: %w", gocrawler.ErrInvalidInput, err)
	}

	args := flag.Args()
	links := make([]*url.URL, 0, len(args))
	for i, arg := range args {
		parsed, err := gocrawler.ParseSeedURL(arg)
		if err != nil {
			return opts, nil, fmt.Errorf("%w: seed URL %d: %w", gocrawler.ErrInvalidInput, i+1, err)
		}
		links = append(links, parsed)
	}

	return opts, links, nil
}
