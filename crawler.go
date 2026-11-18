// Package gocrawler.
package gocrawler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

func Crawl(
	ctx context.Context,
	startLinks []*url.URL,
	opts Options,
) (report Report, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("crawler: %w", err)
		}
	}()

	if err = opts.Validate(); err != nil {
		return report, err
	}

	report.StopReason = CompletedReason

	seen := make(map[string]struct{})
	links, _ := admitLinks(startLinks, seen, opts.MaxPages)

	client := opts.Client
	if client == nil {
		client = http.DefaultClient
	}

	start := time.Now()

loop:
	for depth := 0; len(links) > 0; depth++ {
		select {
		case <-ctx.Done():
			report.StopReason = CanceledReason
			break loop
		default:
		}

		w := func(link *url.URL) workerResult {
			timeoutCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
			defer cancel()
			return worker(timeoutCtx, client, link, opts.MaxBodyBytes)
		}

		res, rawLinks, succeeded, err := workersRunner(
			ctx,
			opts.Concurrency,
			links,
			depth,
			w,
		)

		report.Results = append(report.Results, res...)
		report.Succeeded += succeeded

		if err != nil {
			report.StopReason = CanceledReason
			break loop
		}
		if depth == opts.MaxDepth {
			break loop
		}

		remainingSlots := opts.MaxPages - len(seen)
		newLinks, exceeded := admitLinks(rawLinks, seen, remainingSlots)

		if exceeded {
			report.StopReason = MaxPagesReason
		}

		links = newLinks
	}

	report.Duration = time.Since(start)
	report.Started = len(report.Results)
	report.Failed = report.Started - report.Succeeded

	return report, ctx.Err()
}
