package gocrawler

import "net/url"

func normalizeUniqLinks(urls []*url.URL, uniqLinks map[string]struct{}) []*url.URL {
	links := make([]*url.URL, 0, len(urls))
	for _, link := range urls {
		normalized, ok := normalizeUniqLink(link, uniqLinks)
		if ok {
			uniqLinks[normalized.String()] = struct{}{}
			links = append(links, normalized)
		}
	}
	return links
}

func normalizeUniqLink(link *url.URL, uniqLinks map[string]struct{}) (*url.URL, bool) {
	if link.Scheme != "http" && link.Scheme != "https" || link.Hostname() == "" {
		return nil, false
	}
	link.Fragment = ""
	link.RawFragment = ""
	if _, ok := uniqLinks[link.String()]; !ok {
		return link, true
	}
	return nil, false
}
