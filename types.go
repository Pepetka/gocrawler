package gocrawler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

var (
	ErrInput = errors.New("input error")

	ErrNotOkResponse      = errors.New("not ok response")
	ErrUnsupportedCharset = errors.New("unsupported charset")
	ErrTooLargeBody       = errors.New("too large body")
	ErrDisallowRedirect   = errors.New("disallow redirect")
	ErrTooManyRedirects   = errors.New("too many redirects")
)

type ByteSize uint64

const (
	B ByteSize = 1 << (10 * iota)
	KB
	MB
	GB
)

type StopReason string

const (
	CompletedReason StopReason = "completed"
	MaxPagesReason  StopReason = "max-pages"
	CanceledReason  StopReason = "canceled"
)

type Report struct {
	Results    []Result
	Started    int
	Succeeded  int
	Failed     int
	Duration   time.Duration
	StopReason StopReason
}

type Result struct {
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
	Result
}

type Options struct {
	Concurrency  int
	Client       *http.Client
	Timeout      time.Duration
	MaxDepth     int
	MaxPages     int
	MaxBodyBytes ByteSize
}

func (opts Options) Validate() error {
	if opts.Concurrency < 1 {
		return fmt.Errorf("invalid concurrency %d", opts.Concurrency)
	}
	if opts.Timeout < 1 {
		return fmt.Errorf("invalid timeout %s", opts.Timeout)
	}
	if opts.MaxDepth < 0 {
		return fmt.Errorf("invalid maxDepth %d", opts.MaxDepth)
	}
	if opts.MaxPages < 1 {
		return fmt.Errorf("invalid maxPages %d", opts.MaxPages)
	}
	if opts.MaxBodyBytes < 1 || opts.MaxBodyBytes > GB {
		return fmt.Errorf("invalid maxBodyBytes %d", opts.MaxBodyBytes)
	}
	return nil
}
