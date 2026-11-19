// Gocrawler crawls HTTP and HTTPS URLs and prints a crawl report.
//
//	gocrawler [flags] [seed-url ...]
//
// Run gocrawler -help for flag descriptions and defaults. The exit status is 0
// on success, 2 for invalid input, and 1 for other errors.
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
		if errors.Is(err, gocrawler.ErrInvalidInput) {
			os.Exit(2)
		} else {
			os.Exit(1)
		}
	}
}

func run() error {
	opts, seedURLs, err := parseArgs()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	report, err := gocrawler.Crawl(ctx, seedURLs, opts)
	if errors.Is(err, gocrawler.ErrInvalidInput) {
		return err
	}
	var pagesErr error
	if report.Failed > 0 {
		pagesErr = fmt.Errorf("failed pages: %d of %d (see Errors in the crawl report)", report.Failed, report.Started)
	}

	writeErr := writeReport(os.Stdout, report)
	if writeErr != nil {
		writeErr = fmt.Errorf("write crawl report: %w", writeErr)
	}

	return errors.Join(err, pagesErr, writeErr)
}
