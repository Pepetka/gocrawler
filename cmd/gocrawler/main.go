package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/pepetka/gocrawler"
)

var (
	errInput = errors.New("cmd: input error")
	errPages = errors.New("cmd: page processing error")
)

type ByteSize uint64

const (
	B ByteSize = 1 << (10 * iota)
	KB
	MB
	GB
	TB
	PB
	EB
)

func main() {
	if err := run(); err != nil {
		fmt.Fprint(os.Stderr, err)
		if errors.Is(err, errInput) {
			os.Exit(2)
		} else {
			os.Exit(1)
		}
	}
}

func run() error {
	pa, err := parseArgs()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wrs, err := gocrawler.Crawler(ctx, pa.concurrency, pa.timeout, pa.maxDepth, pa.maxPages, pa.maxBodyBytes, pa.urls)
	var pagesErr error
	for _, wr := range wrs {
		if wr.Err != nil {
			pagesErr = errPages
		}
		fmt.Println(wr)
	}

	return errors.Join(err, pagesErr)
}

type programArgs struct {
	concurrency  int
	timeout      time.Duration
	maxDepth     int
	maxPages     int
	maxBodyBytes uint64
	urls         []*url.URL
}

func parseArgs() (programArgs, error) {
	var pa programArgs
	flag.IntVar(&pa.concurrency, "concurrency", 5, "number of parallel workers")
	flag.DurationVar(&pa.timeout, "duration", 5*time.Second, "working time limitation")
	flag.IntVar(&pa.maxDepth, "max-depth", 2, "maximum depth")
	flag.IntVar(&pa.maxPages, "max-pages", 100, "maximum number of pages")
	flag.Uint64Var(&pa.maxBodyBytes, "max-body-bytes", uint64(2*MB), "maximum number of pages")
	flag.Parse()

	if pa.concurrency < 1 {
		return pa, fmt.Errorf("%w: invalid concurrency %d", errInput, pa.concurrency)
	}
	if pa.timeout < 1 {
		return pa, fmt.Errorf("%w: invalid timeout %s", errInput, pa.timeout)
	}
	if pa.maxDepth < 0 {
		return pa, fmt.Errorf("%w: invalid maxDepth %d", errInput, pa.maxDepth)
	}
	if pa.maxPages < 1 {
		return pa, fmt.Errorf("%w: invalid maxPages %d", errInput, pa.maxPages)
	}
	if pa.maxBodyBytes < uint64(1*B) {
		return pa, fmt.Errorf("%w: invalid maxBodyBytes %d", errInput, pa.maxBodyBytes)
	}

	args := flag.Args()
	urls := make([]*url.URL, 0, len(args))
	for _, arg := range args {
		parsed, err := url.Parse(arg)
		if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Hostname() == "" {
			return pa, fmt.Errorf("%w: invalid url %s", errInput, parsed)
		}
		urls = append(urls, parsed)
	}
	pa.urls = urls

	return pa, nil
}
