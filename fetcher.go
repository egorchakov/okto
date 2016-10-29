package main

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"

	"golang.org/x/net/html"
)

/*
Tags and corresponding attributes to consider when looking for links and assets.
Partially taken from https://stackoverflow.com/questions/2725156/complete-list-of-html-tag-attributes-which-have-a-url-value
*/
var (
	linkTags = map[string][]string{
		"a":      []string{"href"},
		"head":   []string{"profile"},
		"iframe": []string{"longdesc", "src"},
		"q":      []string{"cite"},
	}

	assetTags = map[string][]string{
		"link":   []string{"href"},
		"img":    []string{"src"},
		"script": []string{"src"},
	}
)

const userAgent = "okto"

type FetchResult struct {
	links  []string
	assets []string
}

type Fetcher interface {
	Fetch(string) (*FetchResult, error)
}

var errFetchFailed = errors.New("Failed to fetch URL")

type fetcher struct {
	client   *http.Client
	throttle <-chan time.Time
}

func NewFetcher(timeout time.Duration, rps uint64) Fetcher {
	var f = fetcher{client: &http.Client{Timeout: timeout}}

	if rps > 0 {
		rate := time.Second / time.Duration(rps)
		f.throttle = time.Tick(rate)
		logrus.WithField("one_request_per", rate).Info("Outbound request rate limit")

	} else {
		logrus.Warn("No outbound request rate limit")
	}

	return &f
}

// get performs a GET request with throttling (if enabled).
func (f *fetcher) get(rawURL string) (*http.Response, error) {
	if f.throttle != nil {
		<-f.throttle
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)

	return f.client.Do(req)
}

// Fetch GETs the URL and processes the response.
func (f *fetcher) Fetch(rawURL string) (*FetchResult, error) {
	var result *FetchResult

	resp, err := f.get(rawURL)

	if err != nil {
		logrus.WithField("url", rawURL).WithError(err).Error()
		return nil, errFetchFailed
	}

	logrus.WithFields(logrus.Fields{
		"url":         rawURL,
		"status_code": resp.StatusCode,
	}).Debug("Fetched")

	links, assets := f.processBody(resp.Body)

	logrus.WithFields(logrus.Fields{
		"url":    rawURL,
		"links":  links,
		"assets": assets,
	}).Debug("Processed")

	result = &FetchResult{
		links:  links,
		assets: assets,
	}

	return result, nil
}

// processBody extracts relevant links and assets.
func (f *fetcher) processBody(body io.Reader) (links, assets []string) {
	tokenizer := html.NewTokenizer(body)
	for {
		token := tokenizer.Next()

		switch {

		case token == html.ErrorToken:
			return

		case token == html.StartTagToken:
			t := tokenizer.Token()
			links = append(links, extractAttrs(&t, linkTags)...)
			assets = append(assets, extractAttrs(&t, assetTags)...)
		}
	}
}

func extractAttrs(token *html.Token, tags map[string][]string) []string {
	var attrs = make([]string, 0)

	if relevantAttributes, ok := tags[token.Data]; ok {
		for _, attr := range relevantAttributes {
			if attrVal := getAttr(token, attr); attrVal != "" {
				attrs = append(attrs, attrVal)
				break
			}
		}
	}

	return attrs
}

func getAttr(t *html.Token, key string) string {
	var attr string

	for _, a := range t.Attr {
		if a.Key == key {
			attr = a.Val
			break
		}
	}

	return attr
}
