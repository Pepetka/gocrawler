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
)

func processPage(ctx context.Context, client *http.Client, link *url.URL, maxBodyBytes ByteSize) pageResultWithLinks {
	wr, r := fetchPage(ctx, client, link, int64(maxBodyBytes))
	if r == nil {
		return wr
	}

	links, err := extractLinks(ctx, r, wr.ResponseURL)
	if err != nil {
		wr.Err = fmt.Errorf("extract links from HTML: %w", err)
		return wr
	}

	wr.links = links
	return wr
}

func fetchPage(ctx context.Context, client *http.Client, link *url.URL, maxBodyBytes int64) (pageResultWithLinks, io.Reader) {
	wr := pageResultWithLinks{
		PageResult: PageResult{RequestedURL: link},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link.String(), nil)
	if err != nil {
		wr.Err = fmt.Errorf("create HTTP request: %w", err)
		return wr, nil
	}

	resp, err := client.Do(req)
	if err != nil {
		wr.Err = err
		if resp != nil {
			wr.ResponseURL = resp.Request.URL
			wr.StatusCode = resp.StatusCode
			wr.ContentType = resp.Header.Get("Content-Type")
		}
		return wr, nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	contentType := resp.Header.Get("Content-Type")

	wr.ResponseURL = resp.Request.URL
	wr.StatusCode = resp.StatusCode
	wr.ContentType = contentType

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		wr.Err = fmt.Errorf("%w %d (expected 2xx)", ErrUnexpectedHTTPStatus, resp.StatusCode)
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	isHTML := err == nil && mediaType == "text/html"
	if isHTML {
		charset, ok := params["charset"]
		if ok && strings.ToLower(charset) != "utf-8" {
			wr.Err = errors.Join(wr.Err, fmt.Errorf("%w %q (expected UTF-8)", ErrUnsupportedHTMLCharset, charset))
		}
	}

	var buf bytes.Buffer
	var dst io.Writer = &buf
	discard := !isHTML || wr.Err != nil
	if discard {
		dst = io.Discard
	}
	lr := io.LimitReader(resp.Body, maxBodyBytes+1)
	bytesRead, err := io.Copy(dst, lr)
	wr.BodyBytesRead = bytesRead
	if err != nil {
		wr.Err = errors.Join(wr.Err, fmt.Errorf("read response body: %w", err))
	}
	if bytesRead > maxBodyBytes {
		wr.Err = errors.Join(wr.Err, fmt.Errorf("%w: limit is %d bytes (read %d bytes)", ErrBodyTooLarge, maxBodyBytes, bytesRead))
	}
	if wr.Err != nil {
		return wr, nil
	}

	if discard {
		return wr, nil
	}
	return wr, &buf
}
