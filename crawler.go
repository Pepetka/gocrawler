// Package gocrawler crawls HTTP and HTTPS URLs concurrently, one link depth at a time.
// It follows links and redirects within the origins of the seed URLs.
// An origin consists of the scheme, hostname, and effective port.
package gocrawler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Crawl starts a crawl from seedURLs using opts. Seed URLs must be absolute HTTP
// or HTTPS URLs with a hostname. An empty seed list is allowed.
//
// Invalid options or seed URLs cause Crawl to return an error wrapping
// [ErrInvalidInput] before making any requests. If ctx is canceled or its deadline
// expires, Crawl returns the results collected so far and an error wrapping ctx.Err().
// Errors for individual pages are recorded in [PageResult.Err].
// Crawl extracts links from successful text/html responses.
func Crawl(
	ctx context.Context,
	seedURLs []*url.URL,
	opts Options,
) (report Report, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("crawler: %w", err)
		}
	}()

	if err = opts.Validate(); err != nil {
		return report, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	allowedOrigins, err := seedOrigins(seedURLs)
	if err != nil {
		return report, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	seen := make(map[string]struct{})
	links, exceeded := admitURLs(seedURLs, seen, opts.MaxPages)
	if exceeded {
		return report, fmt.Errorf("%w: unique seed URLs exceed the max-pages limit of %d", ErrInvalidInput, opts.MaxPages)
	}

	report.StopReason = ReasonCompleted

	client := newCrawlClient(opts.HTTPClient, allowedOrigins)

	start := time.Now()

loop:
	for depth := 0; len(links) > 0; depth++ {
		select {
		case <-ctx.Done():
			report.StopReason = ReasonContextDone
			break loop
		default:
		}

		w := func(link *url.URL) pageResultWithLinks {
			timeoutCtx, cancel := context.WithTimeout(ctx, opts.PageTimeout)
			defer cancel()
			return processPage(timeoutCtx, client, link, opts.MaxBodyBytes)
		}

		res, rawLinks, succeeded, err := crawlLevel(
			ctx,
			opts.Concurrency,
			links,
			depth,
			w,
		)

		report.Results = append(report.Results, res...)
		report.Succeeded += succeeded

		if err != nil {
			report.StopReason = ReasonContextDone
			break loop
		}
		if depth == opts.MaxDepth {
			break loop
		}

		allowed := filterByOrigin(allowedOrigins, rawLinks)

		remainingSlots := opts.MaxPages - len(seen)
		newLinks, exceeded := admitURLs(allowed, seen, remainingSlots)

		if exceeded {
			report.StopReason = ReasonPageLimit
		}

		links = newLinks
	}

	report.Duration = time.Since(start)
	report.Started = len(report.Results)
	report.Failed = report.Started - report.Succeeded

	err = ctx.Err()
	if err != nil {
		report.StopReason = ReasonContextDone
	}

	return report, err
}

func newCrawlClient(external *http.Client, allowedOrigins map[string]struct{}) *http.Client {
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

	const maxRequestsPerPage = 10
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRequestsPerPage {
			return fmt.Errorf("%w: at most %d HTTP requests per page, including the initial request", ErrTooManyRedirects, maxRequestsPerPage)
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
			return ErrRedirectOutsideAllowedOrigins
		}
		return nil
	}

	return client
}
