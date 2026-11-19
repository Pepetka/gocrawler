package gocrawler

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseSeedURL parses and validates an absolute HTTP or HTTPS seed URL.
func ParseSeedURL(raw string) (*url.URL, error) {
	link, err := parseURL(raw)
	if err != nil {
		return nil, err
	}
	if _, err := originOf(link); err != nil {
		return nil, err
	}
	return link, nil
}

func parseURL(raw string) (*url.URL, error) {
	link, err := url.Parse(raw)
	if err != nil {
		return nil, err.(*url.Error).Err
	}
	return link, nil
}

func admitURLs(rawLinks []*url.URL, seen map[string]struct{}, remainingSlots int) ([]*url.URL, bool) {
	links := make([]*url.URL, 0, len(rawLinks))
	slots := 0
	exceeded := false
	for _, link := range rawLinks {
		normalized := normalizeCrawlURL(link)
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

func normalizeCrawlURL(link *url.URL) *url.URL {
	l := *link
	if l.Scheme != "http" && l.Scheme != "https" || l.Hostname() == "" {
		return nil
	}
	l.Fragment = ""
	l.RawFragment = ""
	return &l
}

func seedOrigins(seeds []*url.URL) (map[string]struct{}, error) {
	output := make(map[string]struct{})
	for i, link := range seeds {
		origin, err := originOf(link)
		if err != nil {
			return nil, fmt.Errorf("seed URL %d: %w", i+1, err)
		}
		output[origin] = struct{}{}
	}
	return output, nil
}

func filterByOrigin(allowed map[string]struct{}, links []*url.URL) []*url.URL {
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
		return "", errors.New("URL is nil")
	}

	host := link.Hostname()
	scheme := link.Scheme
	var port uint64
	if scheme != "http" && scheme != "https" || host == "" || link.Opaque != "" {
		return "", errors.New("expected an absolute HTTP or HTTPS URL with a hostname")
	}
	// Port returns an empty string for both a missing port and a malformed port.
	if _, err := parseURL(link.Redacted()); err != nil {
		return "", err
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
			return "", fmt.Errorf("URL port must be between 0 and 65535: %w", err)
		}
		port = p
	}
	origin := fmt.Sprintf("%s://%s:%d", scheme, host, port)
	origin = strings.ToLower(origin)
	return origin, nil
}
