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

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if errors.Is(err, gocrawler.ErrInput) {
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

	report, err := gocrawler.Crawl(ctx, startLinks, opts)
	if errors.Is(err, gocrawler.ErrInput) {
		return err
	}
	var pagesErr error
	if report.Failed > 0 {
		pagesErr = errors.New("cmd: page processing error")
	}

	writeErr := writeReport(os.Stdout, report)

	return errors.Join(err, pagesErr, writeErr)
}
