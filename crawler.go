// Package gocrawler.
package gocrawler

import (
	"context"
	"errors"
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
		return report, fmt.Errorf("%w: %w", ErrInput, err)
	}

	allowedOrigins, err := buildAllowedOrigins(startLinks)
	if err != nil {
		return report, fmt.Errorf("%w: %w", ErrInput, err)
	}

	seen := make(map[string]struct{})
	links, exceeded := admitLinks(startLinks, seen, opts.MaxPages)
	if exceeded {
		return report, fmt.Errorf("%w: %w", ErrInput, errors.New("number of unique URLs is greater than max-pages"))
	}

	report.StopReason = CompletedReason

	client := buildClient(opts.Client, allowedOrigins)

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

		allowed := filterByAllowed(allowedOrigins, rawLinks)

		remainingSlots := opts.MaxPages - len(seen)
		newLinks, exceeded := admitLinks(allowed, seen, remainingSlots)

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

func buildClient(external *http.Client, allowedOrigins map[string]struct{}) *http.Client {
	base := external
	if base == nil {
		base = http.DefaultClient
	}
	externalCheck := base.CheckRedirect

	client := &http.Client{
		Transport: base.Transport,
		Jar:       base.Jar,
		Timeout:   base.Timeout,
	}

	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return ErrTooManyRedirects
		}

		if externalCheck != nil {
			if err := externalCheck(req, via); err != nil {
				return err
			}
		}

		destOrigin, err := originOf(req.URL)
		if err != nil {
			return err
		}
		if _, ok := allowedOrigins[destOrigin]; !ok {
			return ErrDisallowRedirect
		}
		return nil
	}

	return client
}
