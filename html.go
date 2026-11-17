package gocrawler

import (
	"io"
	"net/url"

	"golang.org/x/net/html"
)

func parseHTML(r io.Reader, base *url.URL) ([]*url.URL, error) {
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
