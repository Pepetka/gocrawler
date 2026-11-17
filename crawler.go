// Package gocrawler.
package gocrawler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func Crawl(
	ctx context.Context,
	startLinks []*url.URL,
	opts Options,
) (workResults []Result, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("crawler: %w", err)
		}
	}()

	if err = opts.Validate(); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	links := admitLinks(startLinks, seen, opts.MaxPages)

	client := opts.Client
	if client == nil {
		client = http.DefaultClient
	}

	for depth := 0; len(links) > 0; depth++ {
		select {
		case <-ctx.Done():
			return workResults, ctx.Err()
		default:
		}

		w := func(link *url.URL) workerResult {
			timeoutCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
			defer cancel()
			return worker(timeoutCtx, client, link, opts.MaxBodyBytes)
		}

		results, rawLinks, err := workersRunner(
			ctx,
			opts.Concurrency,
			links,
			depth,
			w,
		)

		workResults = append(workResults, results...)

		if err != nil {
			return workResults, err
		}

		if depth == opts.MaxDepth {
			break
		}

		remainingSlots := opts.MaxPages - len(seen)
		newLinks := admitLinks(rawLinks, seen, remainingSlots)
		links = newLinks
	}

	return workResults, nil
}
