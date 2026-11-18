package gocrawler

import "net/url"

func admitLinks(rawLinks []*url.URL, seen map[string]struct{}, remainingSlots int) ([]*url.URL, bool) {
	links := make([]*url.URL, 0, len(rawLinks))
	slots := 0
	exceeded := false
	for _, link := range rawLinks {
		normalized := normalizeLink(link)
		if normalized == nil {
			continue
		}
		if _, ok := seen[normalized.String()]; !ok {
			if slots == remainingSlots {
				exceeded = true
				break
			}

			seen[normalized.String()] = struct{}{}
			links = append(links, normalized)
			slots++
		}
	}
	return links, exceeded
}

func normalizeLink(link *url.URL) *url.URL {
	l := *link
	if l.Scheme != "http" && l.Scheme != "https" || l.Hostname() == "" {
		return nil
	}
	l.Fragment = ""
	l.RawFragment = ""
	return &l
}
