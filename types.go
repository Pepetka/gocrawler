package gocrawler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

var (
	// ErrInvalidInput indicates invalid crawl options or seed URLs.
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnexpectedHTTPStatus indicates an HTTP response status outside the 2xx range.
	ErrUnexpectedHTTPStatus = errors.New("unexpected HTTP status")

	// ErrUnsupportedHTMLCharset indicates an HTML response with an unsupported
	// charset. The supported declared charset is UTF-8.
	ErrUnsupportedHTMLCharset = errors.New("unsupported HTML charset")

	// ErrBodyTooLarge indicates a response body that exceeds [Options.MaxBodyBytes].
	ErrBodyTooLarge = errors.New("response body too large")

	// ErrRedirectOutsideAllowedOrigins indicates a redirect to an origin outside
	// the set defined by the seed URLs.
	ErrRedirectOutsideAllowedOrigins = errors.New("redirect target is outside the allowed seed origins")

	// ErrTooManyRedirects indicates that another redirect would exceed the limit
	// of ten HTTP requests for a page, including the initial request.
	ErrTooManyRedirects = errors.New("too many redirects")
)

// ByteSize represents a size in bytes.
type ByteSize uint64

const (
	// B is one byte.
	B ByteSize = 1 << (10 * iota)
	// KiB is one kibibyte (1024 bytes).
	KiB
	// MiB is one mebibyte (1024 KiB).
	MiB
	// GiB is one gibibyte (1024 MiB).
	GiB
)

// StopReason describes how a crawl ended.
type StopReason string

const (
	// ReasonCompleted means the crawl finished normally, including at the depth limit.
	ReasonCompleted StopReason = "completed"
	// ReasonPageLimit means the page limit prevented scheduling another URL.
	ReasonPageLimit StopReason = "max-pages"
	// ReasonContextDone means the crawl context was canceled or its deadline expired.
	ReasonContextDone StopReason = "canceled"
)

// Report contains the page results and summary of a crawl.
type Report struct {
	// Results contains one entry per page started, grouped by link depth.
	Results []PageResult
	// Started is the number of pages whose processing began. It equals len(Results).
	Started int
	// Succeeded counts pages whose result has a nil error.
	Succeeded int
	// Failed counts pages whose result has a non-nil error.
	Failed int
	// Duration is the elapsed time of the crawl after input validation.
	Duration time.Duration
	// StopReason describes how the crawl ended. It is empty if input validation fails.
	StopReason StopReason
}

// PageResult records the outcome of processing one scheduled page URL.
type PageResult struct {
	// RequestedURL is the page URL before redirects, with its fragment removed.
	RequestedURL *url.URL
	// ResponseURL is the URL associated with the response returned by the HTTP
	// client. It is nil when the client returns no response.
	ResponseURL *url.URL
	// Depth is the number of links followed from a seed URL. Seeds have depth zero.
	Depth int
	// StatusCode is the HTTP response status code, or zero if unavailable.
	StatusCode int
	// ContentType is the value of the response's Content-Type header, if present.
	ContentType string
	// BodyBytesRead counts bytes read from the response body, including any
	// extra byte read to detect a body that exceeds [Options.MaxBodyBytes].
	BodyBytesRead int64
	// Duration is the time spent fetching and processing the page.
	Duration time.Duration
	// Err is the page processing error, or nil on success. Use [errors.Is]
	// to check for a specific sentinel error.
	Err error
}

type pageResultWithLinks struct {
	links []*url.URL
	PageResult
}

// Options configures a crawl.
type Options struct {
	// Concurrency is the maximum number of pages processed in parallel.
	// It must be at least one.
	Concurrency int
	// HTTPClient supplies the transport, cookie jar, timeout, and redirect callback.
	// A nil value selects [http.DefaultClient]. The crawler applies its own redirect limits.
	HTTPClient *http.Client
	// PageTimeout sets a deadline for fetching each page and extracting its links.
	// It must be positive.
	PageTimeout time.Duration
	// MaxDepth is the maximum link depth and must be non-negative.
	// A depth of zero limits the crawl to the seed URLs.
	MaxDepth int
	// MaxPages caps the number of distinct page URLs, including seeds, and must
	// be at least one. A redirect chain counts as one page.
	MaxPages int
	// MaxBodyBytes is the maximum accepted response body size in bytes.
	// The limit must be between 1 and [GiB], inclusive.
	MaxBodyBytes ByteSize
}

// Validate returns an error describing the first invalid option, or nil if all options are valid.
func (opts Options) Validate() error {
	if opts.Concurrency < 1 {
		return fmt.Errorf("concurrency must be at least 1 (got %d)", opts.Concurrency)
	}
	if opts.PageTimeout < 1 {
		return fmt.Errorf("timeout must be greater than zero (got %s)", opts.PageTimeout)
	}
	if opts.MaxDepth < 0 {
		return fmt.Errorf("max-depth must be at least 0 (got %d)", opts.MaxDepth)
	}
	if opts.MaxPages < 1 {
		return fmt.Errorf("max-pages must be at least 1 (got %d)", opts.MaxPages)
	}
	if opts.MaxBodyBytes < 1 || opts.MaxBodyBytes > GiB {
		return fmt.Errorf("max-body-bytes must be between 1 and %d bytes (got %d)", GiB, opts.MaxBodyBytes)
	}
	return nil
}
