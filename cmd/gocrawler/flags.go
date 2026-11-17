package main

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"time"

	"github.com/pepetka/gocrawler"
)

func parseOpts() (gocrawler.Options, []*url.URL, error) {
	opts := gocrawler.Options{}
	flag.IntVar(&opts.Concurrency, "concurrency", 5, "number of parallel workers")
	flag.DurationVar(&opts.Timeout, "duration", 5*time.Second, "working time limitation")
	flag.IntVar(&opts.MaxDepth, "max-depth", 2, "maximum depth")
	flag.IntVar(&opts.MaxPages, "max-pages", 100, "maximum number of pages")
	var maxBodyBytes uint64
	flag.Uint64Var(&maxBodyBytes, "max-body-bytes", uint64(2*gocrawler.MB), "maximum number of pages")
	flag.Parse()

	opts.MaxBodyBytes = gocrawler.ByteSize(maxBodyBytes)

	if err := opts.Validate(); err != nil {
		return opts, nil, errors.Join(errInput, err)
	}

	args := flag.Args()
	links := make([]*url.URL, 0, len(args))
	for _, arg := range args {
		parsed, err := url.Parse(arg)
		if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Hostname() == "" {
			return opts, nil, fmt.Errorf("%w: invalid url %s", errInput, parsed)
		}
		links = append(links, parsed)
	}

	return opts, links, nil
}
