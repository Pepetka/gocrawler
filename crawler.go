// Package gocrawler.
package gocrawler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

var (
	ErrNotOkResponse      = errors.New("not ok response")
	ErrUnsupportedCharset = errors.New("unsupported charset")
	ErrTooLargeBody       = errors.New("too large body")
)

type baseResult struct {
	url         *url.URL
	finalURL    *url.URL
	statusCode  int
	contentType string
	bytesRead   int64
	duration    time.Duration
	err         error
}

type WorkResult struct {
	URL         *url.URL
	FinalURL    *url.URL
	Depth       int
	StatusCode  int
	ContentType string
	BytesRead   int64
	Duration    time.Duration
	Err         error
}

type workerResult struct {
	links []*url.URL
	baseResult
}

func Crawler(
	ctx context.Context,
	concurrency int,
	timeout time.Duration,
	maxDepth int,
	maxPages int,
	maxBodyBytes uint64,
	urls []*url.URL,
) (workResults []WorkResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("crawler: %w", err)
		}
	}()

	allUniqLinks := make(map[string]struct{})
	links := normalizeUniqLinks(urls, allUniqLinks)
	if len(links) > maxPages {
		links = links[:maxPages]
	}

	var sumPages int

	client := &http.Client{
		Timeout: timeout,
	}

	for i := 0; len(links) > 0; i++ {
		select {
		case <-ctx.Done():
			return workResults, ctx.Err()
		default:
		}

		currentLinks := make([]*url.URL, len(links))
		copy(currentLinks, links)

		brs, newLinks, pagesProcessed, err := workersRunner(
			ctx,
			concurrency,
			currentLinks,
			func(link *url.URL) workerResult {
				return worker(ctx, client, link, maxBodyBytes)
			},
		)

		for _, br := range brs {
			workResults = append(workResults, WorkResult{
				URL:         br.url,
				FinalURL:    br.finalURL,
				Depth:       i,
				StatusCode:  br.statusCode,
				ContentType: br.contentType,
				BytesRead:   br.bytesRead,
				Duration:    br.duration,
				Err:         br.err,
			})
		}
		sumPages += pagesProcessed

		if err != nil {
			return workResults, err
		}

		if i == maxDepth {
			break
		}

		pagesLeft := maxPages - sumPages
		normalizedLinks := normalizeUniqLinks(newLinks, allUniqLinks)

		if len(normalizedLinks) > pagesLeft {
			normalizedLinks = normalizedLinks[:pagesLeft]
		}

		links = normalizedLinks
	}

	return workResults, nil
}

func workersRunner(
	ctx context.Context,
	concurrency int,
	links []*url.URL,
	worker func(link *url.URL) workerResult,
) ([]baseResult, []*url.URL, int, error) {
	var wg sync.WaitGroup
	var wgSub sync.WaitGroup
	in := make(chan *url.URL)
	out := make(chan workerResult, concurrency)

	currentPages := 0

	wg.Go(func() {
		defer close(in)
		for _, l := range links {
			select {
			case <-ctx.Done():
				return
			default:
			}

			currentPages++

			select {
			case <-ctx.Done():
				return
			case in <- l:
			}
		}
	})

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
					wr.duration = time.Since(start)
					out <- wr
				}
			}
		})
	}

	wgSub.Go(func() {
		wg.Wait()
		close(out)
	})

	newLinks := make([]*url.URL, 0)
	brs := make([]baseResult, 0)
	for o := range out {
		newLinks = append(newLinks, o.links...)
		brs = append(brs, o.baseResult)
	}

	wgSub.Wait()

	return brs, newLinks, currentPages, ctx.Err()
}

func worker(ctx context.Context, client *http.Client, link *url.URL, maxBodyBytes uint64) workerResult {
	wr, r := getHTML(ctx, client, link, int64(maxBodyBytes))
	if r == nil {
		return wr
	}

	links, err := getLinksFromHTML(r, wr.finalURL)
	if err != nil {
		wr.err = err
		return wr
	}

	wr.links = links
	return wr
}

func getHTML(ctx context.Context, client *http.Client, link *url.URL, maxBodyBytes int64) (workerResult, io.Reader) {
	wr := workerResult{
		url: link,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link.String(), nil)
	if err != nil {
		wr.err = err
		return wr, nil
	}

	resp, err := client.Do(req)
	if err != nil {
		wr.err = err
		return wr, nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	contentType := resp.Header.Get("Content-Type")

	wr.finalURL = resp.Request.URL
	wr.statusCode = resp.StatusCode
	wr.contentType = contentType

	var buf bytes.Buffer
	lr := io.LimitReader(resp.Body, maxBodyBytes+1)
	bytesRead, err := buf.ReadFrom(lr)
	wr.bytesRead = bytesRead

	if err != nil {
		wr.err = err
	}
	if bytesRead > maxBodyBytes {
		wr.err = errors.Join(wr.err, ErrTooLargeBody)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		wr.err = errors.Join(wr.err, ErrNotOkResponse)
	}
	if wr.err != nil {
		return wr, nil
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	charset, ok := params["charset"]
	if err != nil {
		return wr, nil
	}
	if mediaType != "text/html" {
		return wr, nil
	}
	if ok && strings.ToLower(charset) != "utf-8" {
		wr.err = ErrUnsupportedCharset
		return wr, nil
	}

	return wr, &buf
}

func getLinksFromHTML(r io.Reader, base *url.URL) ([]*url.URL, error) {
	links := make([]*url.URL, 0)

	root, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	for node := range root.Descendants() {
		if node.Type != html.ElementNode || node.Data != "a" {
			continue
		}

		var href *url.URL
		for _, h := range node.Attr {
			if h.Key == "href" {
				p, parseErr := url.Parse(h.Val)
				if parseErr == nil {
					href = p
				}
			}
		}

		if href == nil {
			continue
		}

		link := base.ResolveReference(href)
		links = append(links, link)
	}

	return links, nil
}
