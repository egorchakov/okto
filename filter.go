package main

import (
	"net/url"

	"github.com/PuerkitoBio/purell"
	"github.com/Sirupsen/logrus"
)

type Filter interface {
	Filter(*url.URL, []string) []*url.URL
}

type filter struct{}

const purellFlags = (purell.FlagLowercaseHost |
	purell.FlagLowercaseScheme |
	purell.FlagRemoveFragment |
	purell.FlagRemoveEmptyQuerySeparator |
	purell.FlagRemoveTrailingSlash |
	purell.FlagSortQuery |
	purell.FlagRemoveDuplicateSlashes)

func NewFilter() Filter {
	return &filter{}
}

func (f *filter) normalize(parent *url.URL, u *url.URL) string {
	// Inherit host if none, otherwise compare
	if u.Host == "" {
		u.Host = parent.Host
	} else if u.Host != parent.Host {
		return ""
	}

	// Inherit scheme if none, otherwise filter
	if u.Scheme == "" {
		u.Scheme = parent.Scheme
	} else if !(u.Scheme == "http" || u.Scheme == "https") {
		return ""
	}

	return purell.NormalizeURL(u, purellFlags)
}

// Filter selects, normalizes and deduplicates children of a URL
func (f *filter) Filter(parent *url.URL, children []string) []*url.URL {
	var result = make([]*url.URL, 0)

	unique := make(map[string]struct{})

	for _, child := range children {
		parsed, err := url.Parse(child)
		if err != nil {
			logrus.WithField("url", parsed).WithError(err).Error("Failed to parse child")
			continue
		}

		normalized := f.normalize(parent, parsed)

		if normalized == "" {
			continue
		}

		if _, ok := unique[normalized]; ok {
			continue
		}

		normalizedParsed, _ := url.Parse(normalized)
		result = append(result, normalizedParsed)
		unique[normalized] = struct{}{}
	}

	return result
}
