package gocrawler

import (
	"context"
	"net/url"
	"sync"
	"time"
)

func workersRunner(
	ctx context.Context,
	concurrency int,
	links []*url.URL,
	depth int,
	worker func(link *url.URL) workerResult,
) ([]Result, []*url.URL, error) {
	in, wgSub := distributor(ctx, links)

	out := runner(ctx, wgSub, concurrency, in, worker)

	results, newLinks := collector(out, wgSub, depth)

	return results, newLinks, ctx.Err()
}

func distributor(ctx context.Context, links []*url.URL) (<-chan *url.URL, *sync.WaitGroup) {
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

func runner(ctx context.Context, wgSub *sync.WaitGroup, concurrency int, in <-chan *url.URL, worker func(link *url.URL) workerResult) <-chan workerResult {
	out := make(chan workerResult, concurrency)
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

func collector(out <-chan workerResult, wg *sync.WaitGroup, depth int) ([]Result, []*url.URL) {
	links := make([]*url.URL, 0)
	results := make([]Result, 0)
	for wr := range out {
		wr.Depth = depth
		links = append(links, wr.links...)
		results = append(results, wr.Result)
	}
	wg.Wait()
	return results, links
}
