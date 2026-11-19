package gocrawler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

func worker(ctx context.Context, client *http.Client, link *url.URL, maxBodyBytes ByteSize) workerResult {
	wr, r := getHTML(ctx, client, link, int64(maxBodyBytes))
	if r == nil {
		return wr
	}

	links, err := parseHTML(ctx, r, wr.FinalURL)
	if err != nil {
		wr.Err = err
		return wr
	}

	wr.links = links
	return wr
}

func getHTML(ctx context.Context, client *http.Client, link *url.URL, maxBodyBytes int64) (workerResult, io.Reader) {
	wr := workerResult{
		URL: link,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link.String(), nil)
	if err != nil {
		wr.Err = err
		return wr, nil
	}

	resp, err := client.Do(req)
	if err != nil {
		wr.Err = err
		if resp != nil {
			wr.FinalURL = resp.Request.URL
			wr.StatusCode = resp.StatusCode
			wr.ContentType = resp.Header.Get("Content-Type")
		}
		return wr, nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	contentType := resp.Header.Get("Content-Type")

	wr.FinalURL = resp.Request.URL
	wr.StatusCode = resp.StatusCode
	wr.ContentType = contentType

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		wr.Err = ErrNotOkResponse
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	isHTML := err == nil && mediaType == "text/html"
	if isHTML {
		charset, ok := params["charset"]
		if ok && strings.ToLower(charset) != "utf-8" {
			wr.Err = errors.Join(wr.Err, ErrUnsupportedCharset)
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
	wr.BytesRead = bytesRead
	if err != nil {
		wr.Err = errors.Join(wr.Err, err)
	}
	if bytesRead > maxBodyBytes {
		wr.Err = errors.Join(wr.Err, ErrTooLargeBody)
	}
	if wr.Err != nil {
		return wr, nil
	}

	if discard {
		return wr, nil
	}
	return wr, &buf
}
