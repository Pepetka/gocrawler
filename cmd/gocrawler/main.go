package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pepetka/gocrawler"
)

var (
	errInput = errors.New("cmd: input error")
	errPages = errors.New("cmd: page processing error")
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
	opts, startLinks, err := parseOpts()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	results, err := gocrawler.Crawl(ctx, startLinks, opts)
	var pagesErr error
	for _, wr := range results {
		if wr.Err != nil {
			pagesErr = errPages
		}
	}

	printResults(results)

	return errors.Join(err, pagesErr)
}
