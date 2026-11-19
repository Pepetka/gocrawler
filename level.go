package gocrawler

import (
	"context"
	"net/url"
	"sync"
	"time"
)

func crawlLevel(
	ctx context.Context,
	concurrency int,
	links []*url.URL,
	depth int,
	worker func(link *url.URL) pageResultWithLinks,
) ([]PageResult, []*url.URL, int, error) {
	in, wgSub := feedURLs(ctx, links)

	out := startWorkers(ctx, wgSub, concurrency, in, worker)

	results, newLinks, succeeded := collectLevelResults(out, wgSub, depth)

	return results, newLinks, succeeded, ctx.Err()
}

func feedURLs(ctx context.Context, links []*url.URL) (<-chan *url.URL, *sync.WaitGroup) {
	var wg sync.WaitGroup
	in := make(chan *url.URL)

	wg.Go(func() {
		defer close(in)
	loop:
		for _, l := range links {
			select {
			case <-ctx.Done():
				break loop
			default:
			}

			select {
			case <-ctx.Done():
				break loop
			case in <- l:
			}
		}
	})

	return in, &wg
}

func startWorkers(
	ctx context.Context,
	wgSub *sync.WaitGroup,
	concurrency int,
	in <-chan *url.URL,
	worker func(link *url.URL) pageResultWithLinks,
) <-chan pageResultWithLinks {
	out := make(chan pageResultWithLinks, concurrency)
	var wg sync.WaitGroup

	for range concurrency {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				select {
				case <-ctx.Done():
					return
				case l, ok := <-in:
					if !ok {
						return
					}

					start := time.Now()
					wr := worker(l)
					wr.Duration = time.Since(start)
					out <- wr
				}
			}
		})
	}

	wgSub.Go(func() {
		wg.Wait()
		close(out)
	})

	return out
}

func collectLevelResults(out <-chan pageResultWithLinks, wg *sync.WaitGroup, depth int) ([]PageResult, []*url.URL, int) {
	links := make([]*url.URL, 0)
	results := make([]PageResult, 0)
	succeeded := 0
	for wr := range out {
		wr.Depth = depth
		links = append(links, wr.links...)
		results = append(results, wr.PageResult)
		if wr.Err == nil {
			succeeded++
		}
	}
	wg.Wait()
	return results, links, succeeded
}
