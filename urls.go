package gocrawler

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

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

func buildAllowedOrigins(seeds []*url.URL) (map[string]struct{}, error) {
	output := make(map[string]struct{})
	for i, link := range seeds {
		origin, err := originOf(link)
		if err != nil {
			return nil, fmt.Errorf("seed[%d]: %w", i, err)
		}
		output[origin] = struct{}{}
	}
	return output, nil
}

func filterByAllowed(allowed map[string]struct{}, links []*url.URL) []*url.URL {
	filtered := make([]*url.URL, 0, len(links))
	for _, link := range links {
		origin, err := originOf(link)
		if err != nil {
			continue
		}
		if _, ok := allowed[origin]; ok {
			filtered = append(filtered, link)
		}
	}
	return filtered
}

func originOf(link *url.URL) (string, error) {
	if link == nil {
		return "", errors.New("nil link")
	}

	host := link.Hostname()
	scheme := link.Scheme
	var port uint64
	if scheme != "http" && scheme != "https" || host == "" {
		return "", fmt.Errorf("invalid url %s", link)
	}
	if link.Port() == "" {
		if scheme == "https" {
			port = 443
		} else {
			port = 80
		}
	} else {
		p, err := strconv.ParseUint(link.Port(), 10, 16)
		if err != nil {
			return "", fmt.Errorf("invalid port: %w", err)
		}
		port = p
	}
	origin := fmt.Sprintf("%s://%s:%d", scheme, host, port)
	origin = strings.ToLower(origin)
	return origin, nil
}
